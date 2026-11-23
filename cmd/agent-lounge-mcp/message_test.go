package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/export"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

func messageCmd(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := runMessage(context.Background(), args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

const messageHeader = "ID\tSENT\tFROM\tTO\tSCOPE\tSCOPE_ID\tQUEUED\tDELIVERED\tACKED\tEXPIRED"

// msgRow is one message of the command fixture, in the order it was sent.
type msgRow struct {
	id                                int64
	sent                              time.Time
	from, to, kind, scopeID, body     string
	queued, delivered, acked, expired int
}

func (r msgRow) line() string {
	return fmt.Sprintf("%d\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d",
		r.id, r.sent.UTC().Format(time.RFC3339), r.from, r.to, r.kind, r.scopeID,
		r.queued, r.delivered, r.acked, r.expired)
}

// msgDB is a database with one message in each delivery state, a room message, a direct
// message, a message with no recipient, a sub-second timestamp, and two memories.
type msgDB struct {
	path                                           string
	rows                                           []msgRow
	tExpireAt, tDeliveredAt, tRoom, tFrac, tQueued time.Time
	byBody                                         map[string]msgRow
}

func (db msgDB) row(body string) msgRow { return db.byBody[body] }

func (db msgDB) listLines() string {
	lines := []string{messageHeader}
	for i := len(db.rows) - 1; i >= 0; i-- {
		lines = append(lines, db.rows[i].line())
	}
	return strings.Join(lines, "\n") + "\n"
}

func (db msgDB) idsWhere(keep func(msgRow) bool) []int64 {
	var ids []int64
	for i := len(db.rows) - 1; i >= 0; i-- {
		if keep(db.rows[i]) {
			ids = append(ids, db.rows[i].id)
		}
	}
	return ids
}

// seedMessages builds the fixture and closes it before the commands open it.
func seedMessages(t *testing.T) msgDB {
	t.Helper()
	db := msgDB{path: filepath.Join(t.TempDir(), "lounge.db"), byBody: map[string]msgRow{}}
	ctx := context.Background()
	st, err := store.Open(ctx, db.path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	var now time.Time
	st.SetClock(func() time.Time { return now })

	tExpireSend := time.Date(2026, 11, 23, 8, 0, 0, 0, time.UTC)
	db.tExpireAt = time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)
	tSolo := time.Date(2026, 11, 23, 10, 0, 1, 0, time.UTC)
	db.tQueued = time.Date(2026, 11, 23, 10, 0, 2, 0, time.UTC)
	tDeliveredSend := time.Date(2026, 11, 23, 10, 0, 3, 0, time.UTC)
	db.tDeliveredAt = time.Date(2026, 11, 23, 10, 0, 4, 0, time.UTC)
	tAckedSend := time.Date(2026, 11, 23, 10, 0, 5, 0, time.UTC)
	tAckedAt := time.Date(2026, 11, 23, 10, 0, 6, 0, time.UTC)
	db.tRoom = time.Date(2026, 11, 23, 10, 0, 7, 0, time.UTC)
	db.tFrac = time.Date(2026, 11, 23, 10, 0, 7, 500_000_000, time.UTC)

	now = tExpireSend
	repo := &store.Location{Origin: "github.com/org/a"}
	mainBr := "main"
	joinAt := func(user store.User, name string, loc *store.Location, branch *string) store.Session {
		t.Helper()
		sess, _, err := st.JoinSession(ctx, store.JoinParams{UserID: user.ID, Name: name, Location: loc, Branch: branch})
		if err != nil {
			t.Fatal(err)
		}
		return sess
	}
	aliceU, err := st.AddUser(ctx, "alice", "h-alice", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	bobU, err := st.AddUser(ctx, "bob", "h-bob", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	carolU, err := st.AddUser(ctx, "carol", "h-carol", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	onlyU, err := st.AddUser(ctx, "only", "h-only", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	alice := joinAt(aliceU, "dev", repo, &mainBr)
	extra := joinAt(aliceU, "extra", nil, nil)
	bob := joinAt(bobU, "review", repo, &mainBr)
	carol := joinAt(carolU, "ops", repo, &mainBr)
	only := joinAt(onlyU, "solo", nil, nil)

	add := func(sent store.Sent, sentAt time.Time, from, to, kind, scopeID, body string, q, d, a, e int) {
		t.Helper()
		row := msgRow{id: sent.MessageID, sent: sentAt, from: from, to: to, kind: kind, scopeID: scopeID, body: body, queued: q, delivered: d, acked: a, expired: e}
		db.rows = append(db.rows, row)
		db.byBody[body] = row
	}
	sendTo := func(sender store.Session, scope, user, name, body string) store.Sent {
		t.Helper()
		sent, err := st.Send(ctx, sender.ID, scope, store.Target{User: user, Name: name}, body)
		if err != nil {
			t.Fatal(err)
		}
		return sent
	}
	sendRoom := func(sender store.Session, scope, body string) store.Sent {
		t.Helper()
		sent, err := st.Send(ctx, sender.ID, scope, store.Target{Room: store.LoungeRoom}, body)
		if err != nil {
			t.Fatal(err)
		}
		return sent
	}

	add(sendTo(bob, store.ScopeRepository, "alice", "dev", "expire-body"), tExpireSend, "bob/review", "alice/dev", store.ScopeRepository, "github.com/org/a", "expire-body", 0, 0, 0, 1)
	now = db.tExpireAt
	if cleaned, err := st.Cleanup(ctx, store.CleanupPolicy{ExpireAfter: time.Hour}); err != nil || cleaned.Expired != 1 {
		t.Fatalf("expire: %+v %v", cleaned, err)
	}

	now = tSolo
	add(sendRoom(only, store.ScopeUser, "solo-body"), tSolo, "only/solo", store.LoungeRoom, store.ScopeUser, "only", "solo-body", 0, 0, 0, 0)

	now = db.tQueued
	add(sendTo(alice, store.ScopeUser, "alice", "extra", "queued-body"), db.tQueued, "alice/dev", "alice/extra", store.ScopeUser, "alice", "queued-body", 1, 0, 0, 0)
	userBody, repoBody := "user-memory-body", "repo-memory-body"
	userMem, err := st.CreateMemory(ctx, alice.ID, store.ScopeUser, store.MemoryInput{Body: &userBody})
	if err != nil {
		t.Fatal(err)
	}
	repoMem, err := st.CreateMemory(ctx, alice.ID, store.ScopeRepository, store.MemoryInput{Body: &repoBody})
	if err != nil {
		t.Fatal(err)
	}

	now = tDeliveredSend
	delivered := sendTo(alice, store.ScopeRepository, "bob", "review", "delivered-body")
	now = db.tDeliveredAt
	if _, err := st.FetchInbox(ctx, bob.ID, 10); err != nil {
		t.Fatal(err)
	}
	add(delivered, tDeliveredSend, "alice/dev", "bob/review", store.ScopeRepository, "github.com/org/a", "delivered-body", 0, 1, 0, 0)

	now = tAckedSend
	acked := sendTo(alice, store.ScopeRepository, "carol", "ops", "acked-body")
	now = tAckedAt
	if _, err := st.FetchInbox(ctx, carol.ID, 10); err != nil {
		t.Fatal(err)
	}
	if _, missing, err := st.Ack(ctx, carol.ID, []int64{acked.MessageID}); err != nil || len(missing) != 0 {
		t.Fatalf("ack: %v %v", missing, err)
	}
	add(acked, tAckedSend, "alice/dev", "carol/ops", store.ScopeRepository, "github.com/org/a", "acked-body", 0, 0, 1, 0)

	now = db.tRoom
	updated := "repo-memory-updated"
	if _, err := st.UpdateMemory(ctx, alice.ID, repoMem.ID, store.MemoryInput{Body: &updated}); err != nil {
		t.Fatal(err)
	}
	add(sendRoom(alice, store.ScopeBranch, "room-body"), db.tRoom, "alice/dev", store.LoungeRoom, store.ScopeBranch, "github.com/org/a:main", "room-body", 2, 0, 0, 0)

	now = db.tFrac
	add(sendTo(extra, store.ScopeUser, "alice", "dev", "fraction-body"), db.tFrac, "alice/extra", "alice/dev", store.ScopeUser, "alice", "fraction-body", 1, 0, 0, 0)
	if userMem.ID == 0 || repoMem.ID == 0 {
		t.Fatal("memories were not created")
	}
	return db
}

func messageIDs(t *testing.T, stdout string) []int64 {
	t.Helper()
	var ids []int64
	for _, line := range strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")[1:] {
		if strings.HasPrefix(line, "\t") {
			continue
		}
		id, err := strconv.ParseInt(strings.Split(line, "\t")[0], 10, 64)
		if err != nil {
			t.Fatalf("id in %q: %v", line, err)
		}
		ids = append(ids, id)
	}
	return ids
}

func TestMessageListAndShow(t *testing.T) {
	db := seedMessages(t)
	code, stdout, stderr := messageCmd("list", "--db", db.path)
	if code != 0 || stderr != "" || stdout != db.listLines() {
		t.Fatalf("list: %d %q %q\nwant %q", code, stdout, stderr, db.listLines())
	}
	if strings.Contains(stdout, "expire-body") || strings.Contains(stdout, "fraction-body") {
		t.Fatalf("list printed a body:\n%s", stdout)
	}
	// The fraction is stored at 10:00:07.500Z and printed as that second.
	frac := db.row("fraction-body")
	if !strings.Contains(frac.line(), "2026-11-23T10:00:07Z") || strings.Contains(frac.line(), ".") {
		t.Fatalf("displayed sent time: %s", frac.line())
	}

	code, stdout, stderr = messageCmd("list", "--details", "--db", db.path)
	if code != 0 || stderr != "" {
		t.Fatalf("details: %d %s", code, stderr)
	}
	room := db.row("room-body")
	wantRoom := room.line() + "\n" +
		"\tbob/review\tqueued\t" + db.tRoom.Format(time.RFC3339) + "\n" +
		"\tcarol/ops\tqueued\t" + db.tRoom.Format(time.RFC3339) + "\n"
	if !strings.Contains(stdout, wantRoom) {
		t.Fatalf("room details:\n%s\nwant block:\n%s", stdout, wantRoom)
	}
	solo := db.row("solo-body")
	if !strings.Contains(stdout, solo.line()+"\n") {
		t.Fatalf("details omitted the message with no recipient:\n%s", stdout)
	}
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, "\tonly/solo\t") {
			t.Fatalf("a message with no recipient gained a detail line:\n%s", stdout)
		}
	}

	show := func(body, want string) {
		t.Helper()
		code, stdout, stderr := messageCmd("show", strconv.FormatInt(db.row(body).id, 10), "--db", db.path)
		if code != 0 || stderr != "" || stdout != want {
			t.Errorf("show %s: %d %q %q\nwant %q", body, code, stdout, stderr, want)
		}
	}
	exp := db.row("expire-body")
	show("expire-body", fmt.Sprintf("ID: %d\nFrom: bob/review\nTo: alice/dev\nScope: repository github.com/org/a\nSent: %s\nDeliveries:\n\talice/dev\texpired\t%s\n\nexpire-body\n",
		exp.id, exp.sent.Format(time.RFC3339), db.tExpireAt.Format(time.RFC3339)))
	del := db.row("delivered-body")
	show("delivered-body", fmt.Sprintf("ID: %d\nFrom: alice/dev\nTo: bob/review\nScope: repository github.com/org/a\nSent: %s\nDeliveries:\n\tbob/review\tdelivered\t%s\n\ndelivered-body\n",
		del.id, del.sent.Format(time.RFC3339), db.tDeliveredAt.Format(time.RFC3339)))
	show("room-body", fmt.Sprintf("ID: %d\nFrom: alice/dev\nTo: lounge\nScope: branch github.com/org/a:main\nSent: %s\nDeliveries:\n\tbob/review\tqueued\t%s\n\tcarol/ops\tqueued\t%s\n\nroom-body\n",
		room.id, db.tRoom.Format(time.RFC3339), db.tRoom.Format(time.RFC3339), db.tRoom.Format(time.RFC3339)))
	show("solo-body", fmt.Sprintf("ID: %d\nFrom: only/solo\nTo: lounge\nScope: user only\nSent: %s\nDeliveries: none\n\nsolo-body\n",
		solo.id, solo.sent.Format(time.RFC3339)))
	show("fraction-body", fmt.Sprintf("ID: %d\nFrom: alice/extra\nTo: alice/dev\nScope: user alice\nSent: %s\nDeliveries:\n\talice/dev\tqueued\t%s\n\nfraction-body\n",
		frac.id, frac.sent.Format(time.RFC3339), frac.sent.Format(time.RFC3339)))

	if code, _, stderr := messageCmd("show", "999999", "--db", db.path); code != 1 || !strings.Contains(stderr, "no such message") {
		t.Errorf("missing id: %d %s", code, stderr)
	}

	checkIDs := func(args []string, want []int64) {
		t.Helper()
		code, stdout, stderr := messageCmd(append(args, "--db", db.path)...)
		got := messageIDs(t, stdout)
		if code != 0 || stderr != "" || !slices.Equal(got, want) {
			t.Errorf("%v: %d %v %s\nwant %v\n%s", args, code, got, stderr, want, stdout)
		}
	}
	checkIDs([]string{"list", "--state", "expired"}, []int64{exp.id})
	checkIDs([]string{"list", "--state", "delivered"}, []int64{del.id})
	checkIDs([]string{"list", "--state", "acked"}, []int64{db.row("acked-body").id})
	checkIDs([]string{"list", "--state", "queued"}, db.idsWhere(func(r msgRow) bool { return r.queued > 0 }))
	checkIDs([]string{"list", "--user", "only"}, []int64{solo.id})
	checkIDs([]string{"list", "--user", "bob"}, []int64{room.id, del.id, exp.id})
	checkIDs([]string{"list", "--user", "nobody"}, nil)
	checkIDs([]string{"list", "--scope", "repository", "--scope-id", "github.com/org/a"}, []int64{db.row("acked-body").id, del.id, exp.id})
	checkIDs([]string{"list", "--scope", "branch", "--scope-id", "github.com/org/a:main"}, []int64{room.id})
	checkIDs([]string{"list", "--scope", "user", "--scope-id", "alice"}, []int64{frac.id, db.row("queued-body").id})
	checkIDs([]string{"list", "--scope", "user", "--scope-id", ""}, nil)
	// The delivered message was sent at 10:00:03 and delivered at 10:00:04. The bound uses the sent time.
	checkIDs([]string{"list", "--since", "2026-11-23T10:00:03Z", "--until", "2026-11-23T10:00:04Z"}, []int64{del.id})
	checkIDs([]string{"list", "--since", "2026-11-23T10:00:04Z"}, []int64{frac.id, room.id, db.row("acked-body").id})
	checkIDs([]string{"list", "--since", "2026-11-23T19:00:07+09:00"}, []int64{frac.id, room.id})
	checkIDs([]string{"list", "--since", "2026-11-23T10:00:07.5Z"}, []int64{frac.id})
	checkIDs([]string{"list", "--since", "2026-11-23T10:00:07,5Z"}, []int64{frac.id})
	checkIDs([]string{"list", "--until", "2026-11-23T10:00:07.5Z"}, db.idsWhere(func(r msgRow) bool { return r.id != frac.id }))
	checkIDs([]string{"list", "--since", "2026-11-23T10:00:07Z", "--until", "2026-11-23T10:00:07Z"}, nil)
	checkIDs([]string{"list", "--limit", "1"}, []int64{frac.id})
	checkIDs([]string{"list", "--limit", "1", "--before-id", strconv.FormatInt(frac.id, 10)}, []int64{room.id})
	checkIDs([]string{"list", "--before-id", strconv.FormatInt(db.rows[0].id, 10)}, nil)
}

func TestMessageListDefaultLimit(t *testing.T) {
	db := filepath.Join(t.TempDir(), "lounge.db")
	ctx := context.Background()
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	local, err := st.EnsureLocalUser(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ann, _, err := st.JoinSession(ctx, store.JoinParams{UserID: local.ID, Name: "ann"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.JoinSession(ctx, store.JoinParams{UserID: local.ID, Name: "ben"}); err != nil {
		t.Fatal(err)
	}
	ids := make([]int64, 101)
	for i := range ids {
		sent, err := st.Send(ctx, ann.ID, store.ScopeUser, store.Target{User: store.LocalUser, Name: "ben"}, "n")
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = sent.MessageID
	}
	st.Close()

	code, stdout, stderr := messageCmd("list", "--db", db)
	got := messageIDs(t, stdout)
	want := slices.Clone(ids[1:])
	slices.Reverse(want)
	if code != 0 || stderr != "" || !slices.Equal(got, want) {
		t.Fatalf("default limit: %d %d lines %s\nfirst %v last %v", code, len(got), stderr, head(got, 3), head(want, 3))
	}
	code, stdout, stderr = messageCmd("list", "--before-id", strconv.FormatInt(got[len(got)-1], 10), "--db", db)
	if code != 0 || stderr != "" || !slices.Equal(messageIDs(t, stdout), []int64{ids[0]}) {
		t.Fatalf("next page: %d %q %s", code, stdout, stderr)
	}
}

func head(ids []int64, n int) []int64 {
	if len(ids) < n {
		return ids
	}
	return ids[:n]
}

func TestMessageNamesAndRemovedUser(t *testing.T) {
	db := filepath.Join(t.TempDir(), "lounge.db")
	ctx := context.Background()
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	repo := &store.Location{Origin: "github.com/org/tiny"}
	add := func(name string) store.User {
		t.Helper()
		u, err := st.AddUser(ctx, name, "h-"+name, time.Time{})
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	join := func(u store.User, name string, loc *store.Location) store.Session {
		t.Helper()
		sess, _, err := st.JoinSession(ctx, store.JoinParams{UserID: u.ID, Name: name, Location: loc})
		if err != nil {
			t.Fatal(err)
		}
		return sess
	}
	send := func(sender store.Session, scope, user, name, body string) int64 {
		t.Helper()
		sent, err := st.Send(ctx, sender.ID, scope, store.Target{User: user, Name: name}, body)
		if err != nil {
			t.Fatal(err)
		}
		return sent.MessageID
	}
	local, err := st.EnsureLocalUser(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ann := join(local, "ann", nil)
	join(local, "ben", nil)
	localID := send(ann, store.ScopeUser, store.LocalUser, "ben", "local-body")
	zed := add("Zed")
	zedDev := join(zed, "dev", nil)
	join(zed, "other", nil)
	zedID := send(zedDev, store.ScopeUser, "zed", "other", "zed-body")
	hyphen := add("-x")
	xa := join(hyphen, "a", nil)
	join(hyphen, "b", nil)
	hyphenID := send(xa, store.ScopeUser, "-x", "b", "hyphen-body")
	bobU, goneU := add("bob"), add("gone")
	bob, gone := join(bobU, "review", repo), join(goneU, "old", repo)
	fromGone := send(gone, store.ScopeRepository, "bob", "review", "from-gone")
	room, err := st.Send(ctx, bob.ID, store.ScopeRepository, store.Target{Room: store.LoungeRoom}, "room-with-gone")
	if err != nil {
		t.Fatal(err)
	}
	userBody, repoBody := "gone-user-memory", "gone-repo-memory"
	if _, err := st.CreateMemory(ctx, gone.ID, store.ScopeUser, store.MemoryInput{Body: &userBody}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateMemory(ctx, gone.ID, store.ScopeRepository, store.MemoryInput{Body: &repoBody}); err != nil {
		t.Fatal(err)
	}
	// The store stays open: the commands open the file themselves, which SQLite allows.
	one := func(args []string, id int64) {
		t.Helper()
		code, stdout, stderr := messageCmd(append(args, "--db", db)...)
		if code != 0 || stderr != "" || !slices.Equal(messageIDs(t, stdout), []int64{id}) {
			t.Errorf("%v: %d %q %s", args, code, stdout, stderr)
		}
	}
	one([]string{"list", "--user", "Local"}, localID)
	one([]string{"list", "--user", "Zed"}, zedID)
	one([]string{"list", "--user", "ＺＥＤ"}, zedID)
	one([]string{"list", "--user", "-x"}, hyphenID)
	one([]string{"list", "--user=-x"}, hyphenID)

	code, stdout, stderr := messageCmd("list", "--user", "gone", "--db", db)
	if code != 0 || stderr != "" || !slices.Equal(messageIDs(t, stdout), []int64{room.MessageID, fromGone}) {
		t.Fatalf("before removal: %d %q %s", code, stdout, stderr)
	}

	if _, err := st.RemoveUser(ctx, "gone"); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = messageCmd("list", "--user", "gone", "--db", db)
	if code != 0 || stderr != "" || stdout != messageHeader+"\n" || strings.Contains(stderr, "no such user") {
		t.Fatalf("removed user: %d %q %s", code, stdout, stderr)
	}
	if code, _, stderr := messageCmd("show", strconv.FormatInt(fromGone, 10), "--db", db); code != 1 || !strings.Contains(stderr, "no such message") {
		t.Errorf("removed sender's message: %d %s", code, stderr)
	}
	code, stdout, stderr = messageCmd("show", strconv.FormatInt(room.MessageID, 10), "--db", db)
	if code != 0 || !strings.Contains(stdout, "From: bob/review\n") || !strings.Contains(stdout, "Deliveries: none\n") || strings.Contains(stdout, "gone/") {
		t.Errorf("room after the recipient was removed: %d %q %s", code, stdout, stderr)
	}

	out := filepath.Join(t.TempDir(), "out")
	code, stdout, stderr = exportCmd("--user", "gone", "--out", out, "--db", db)
	if code != 0 || !strings.Contains(stdout, "exported 0 messages and 1 memory to 1 file ") || strings.Contains(stderr, "every user") || strings.Contains(stderr, "no such user") {
		t.Fatalf("export removed user: %d %q %s", code, stdout, stderr)
	}
	files := fileTexts(t, out)
	if len(files) != 1 {
		t.Fatalf("files: %v", files)
	}
	for name, text := range files {
		if !strings.Contains(text, "gone-repo-memory") || strings.Contains(text, "gone-user-memory") || strings.Contains(text, "from-gone") {
			t.Errorf("file %s:\n%s", name, text)
		}
	}
}

func TestMessageUsageErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.db")
	for _, tc := range []struct {
		args []string
		code int
		want string
	}{
		{nil, 2, "usage"},
		{[]string{"nope"}, 2, "unknown message command"},
		{[]string{"list", "x"}, 2, "no argument"},
		{[]string{"list", "--state", "read"}, 2, "unknown --state"},
		{[]string{"list", "--state", "queued", "--scope-id", "x"}, 2, "--scope-id needs --scope"},
		{[]string{"list", "--scope", "global"}, 2, "unknown --scope"},
		{[]string{"list", "--since", "yesterday"}, 2, "invalid --since"},
		{[]string{"list", "--until", "2026-11-23 08:00:00Z"}, 2, "invalid --until"},
		{[]string{"list", "--limit", "0"}, 2, "--limit must be at least 1"},
		{[]string{"list", "--before-id", "0"}, 2, "--before-id must be a positive message ID"},
		{[]string{"list", "--before-id", "-1"}, 2, "--before-id must be a positive message ID"},
		{[]string{"list", "--user", "bad name", "--db", missing}, 1, "invalid user name"},
		{[]string{"list", "--since", "yesterday", "--db", missing}, 2, "invalid --since"},
		{[]string{"show"}, 2, "one message ID"},
		{[]string{"show", "abc"}, 2, "one message ID"},
		{[]string{"show", "0"}, 2, "one message ID"},
		{[]string{"show", "--", "-1"}, 2, "one message ID"},
		{[]string{"show", "1", "2"}, 2, "one message ID"},
		{[]string{"show", "1", "--details"}, 2, "not defined"},
		{[]string{"list", "--db", missing}, 1, "does not exist"},
		{[]string{"show", "1", "--db", missing}, 1, "does not exist"},
	} {
		code, stdout, stderr := messageCmd(tc.args...)
		if code != tc.code || !strings.Contains(stderr, tc.want) || stdout != "" {
			t.Errorf("%v: %d %q %q, want %d and %q", tc.args, code, stdout, stderr, tc.code, tc.want)
		}
	}
	if code, stdout, stderr := messageCmd("help"); code != 0 || !strings.Contains(stdout, "message show") || stderr != "" {
		t.Errorf("help: %d %q %s", code, stdout, stderr)
	}
	if code, stdout, stderr := messageCmd("list", "--help"); code != 0 || stdout != "" || !strings.Contains(stderr, "--before-id") {
		t.Errorf("list --help: %d %q %s", code, stdout, stderr)
	}
	for _, args := range [][]string{
		{"--state", "queued", "--out", t.TempDir()},
		{"--details", "--out", t.TempDir()},
		{"--limit", "1", "--out", t.TempDir()},
		{"--before-id", "1", "--out", t.TempDir()},
	} {
		code, stdout, stderr := exportCmd(args...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, "not defined") {
			t.Errorf("export %v: %d %q %s", args, code, stdout, stderr)
		}
	}
	if code, _, stderr := exportCmd("--since", "yesterday", "--out", t.TempDir()); code != 2 || !strings.Contains(stderr, "invalid --since") {
		t.Errorf("export bad time: %d %s", code, stderr)
	}
	if code, _, stderr := exportCmd("--scope-id", "x", "--out", t.TempDir()); code != 2 || !strings.Contains(stderr, "--scope-id needs --scope") {
		t.Errorf("export scope-id: %d %s", code, stderr)
	}
	if code, _, stderr := exportCmd("--scope", "global", "--out", t.TempDir()); code != 2 || !strings.Contains(stderr, "unknown --scope") {
		t.Errorf("export scope: %d %s", code, stderr)
	}

	var usageText bytes.Buffer
	usage(&usageText)
	if !strings.Contains(usageText.String(), "message list|show") || !strings.Contains(usageText.String(), "export [filters]") {
		t.Errorf("top-level usage:\n%s", usageText.String())
	}
}

func TestExportFilters(t *testing.T) {
	db := seedMessages(t)
	since := db.tRoom.Format(time.RFC3339) // 2026-11-23T10:00:07Z: the room, the fraction and the updated memory

	fullSum, fullDir := exportTree(t, db.path, store.RecordFilter{})
	fullOut := filepath.Join(t.TempDir(), "full")
	code, stdout, stderr := exportCmd("--out", fullOut, "--db", db.path)
	if code != 0 || stdout != exportLine(fullSum, fullOut) || stderr != "The files contain the messages and memories of every user of the database: keep them where only the people who may read them can.\n" {
		t.Fatalf("unfiltered: %d %q %q", code, stdout, stderr)
	}
	if diff := diffTrees(t, fullDir, fullOut); diff != "" {
		t.Fatalf("unfiltered files differ from store.Export:\n%s", diff)
	}

	filtered := store.RecordFilter{Since: db.tRoom}
	filtSum, filtDir := exportTree(t, db.path, filtered)
	filtOut := filepath.Join(t.TempDir(), "filt")
	code, stdout, stderr = exportCmd("--since", since, "--out", filtOut, "--db", db.path)
	if code != 0 || stdout != exportLine(filtSum, filtOut) || stderr != "The files contain only the messages and memories that match the filter: keep them where only the people who may read them can.\n" {
		t.Fatalf("filtered: %d %q %q", code, stdout, stderr)
	}
	if diff := diffTrees(t, filtDir, filtOut); diff != "" {
		t.Fatalf("filtered files differ from ExportSelected:\n%s", diff)
	}
	text := joinedTexts(fileTexts(t, filtOut))
	for _, want := range []string{"room-body", "fraction-body", "repo-memory-updated"} {
		if !strings.Contains(text, want) {
			t.Errorf("filtered export missing %s\n%s", want, text)
		}
	}
	for _, absent := range []string{"queued-body", "expire-body", "delivered-body", "acked-body", "solo-body", "user-memory-body", "repo-memory-body"} {
		if strings.Contains(text, absent) {
			t.Errorf("filtered export contains %s", absent)
		}
	}
	if filtSum.Files >= fullSum.Files || filtSum.Messages != 2 || filtSum.Memories != 1 {
		t.Fatalf("filtered summary: %+v full %+v", filtSum, fullSum)
	}

	// A filtered export overwrites the files it writes and leaves every other file in place.
	mixed := filepath.Join(t.TempDir(), "mixed")
	if err := os.MkdirAll(filepath.Join(mixed, "chat", "old"), 0o755); err != nil {
		t.Fatal(err)
	}
	stray := filepath.Join(mixed, "stray.txt")
	previous := filepath.Join(mixed, "chat", "old", "room-lounge.md")
	if err := os.WriteFile(stray, []byte("keep-stray"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(previous, []byte("keep-previous"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, stderr = exportCmd("--user", "only", "--out", mixed, "--db", db.path)
	if code != 0 || strings.Contains(stderr, "every user") {
		t.Fatalf("export --user only: %d %s", code, stderr)
	}
	if b, err := os.ReadFile(stray); err != nil || string(b) != "keep-stray" {
		t.Errorf("stray file: %q %v", b, err)
	}
	if b, err := os.ReadFile(previous); err != nil || string(b) != "keep-previous" {
		t.Errorf("previous file: %q %v", b, err)
	}
	onlyText := joinedTexts(fileTexts(t, mixed))
	if !strings.Contains(onlyText, "solo-body") || strings.Contains(onlyText, "queued-body") || strings.Contains(onlyText, "room-body") {
		t.Errorf("export --user only wrote:\n%s", onlyText)
	}

	empty := filepath.Join(t.TempDir(), "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(empty, "stray.txt"), []byte("still"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ = exportCmd("--user", "nobody", "--out", empty, "--db", db.path)
	if code != 0 || !strings.Contains(stdout, "exported 0 messages and 0 memories to 0 files ") {
		t.Fatalf("export nobody: %d %q", code, stdout)
	}
	entries, err := os.ReadDir(empty)
	if err != nil || len(entries) != 1 || entries[0].Name() != "stray.txt" {
		t.Errorf("empty filter removed files: %v %v", entries, err)
	}
}

func exportLine(sum export.Summary, dir string) string {
	return fmt.Sprintf("exported %s and %s to %s in %s\n",
		plural(sum.Messages, "message", "messages"), plural(sum.Memories, "memory", "memories"), plural(sum.Files, "file", "files"), dir)
}

func exportTree(t *testing.T, db string, f store.RecordFilter) (export.Summary, string) {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	dir := t.TempDir()
	sum, err := export.WriteMarkdown(dir, func(onMessage func(store.ExportMessage) error, onMemory func(store.Memory) error) error {
		if f.IsZero() {
			return st.Export(ctx, onMessage, onMemory)
		}
		return st.ExportSelected(ctx, f, onMessage, onMemory)
	})
	if err != nil {
		t.Fatal(err)
	}
	return sum, dir
}

func fileTexts(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return out
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func joinedTexts(files map[string]string) string {
	var b strings.Builder
	for _, text := range files {
		b.WriteString(text)
		b.WriteByte('\n')
	}
	return b.String()
}

func diffTrees(t *testing.T, wantDir, gotDir string) string {
	t.Helper()
	want, got := fileTexts(t, wantDir), fileTexts(t, gotDir)
	var b strings.Builder
	for name, text := range want {
		if got[name] != text {
			fmt.Fprintf(&b, "file %s\n", name)
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			fmt.Fprintf(&b, "extra file %s\n", name)
		}
	}
	return b.String()
}

func TestMemoryListScopeID(t *testing.T) {
	db := filepath.Join(t.TempDir(), "lounge.db")
	userID, repoID := seedMemories(t, db, time.Now())
	code, stdout, stderr := memory("list", "--scope", "repository", "--scope-id", "github.com/org/a", "--db", db)
	if code != 0 || stderr != "" || strings.Contains(stdout, "\tuser\t") || !strings.Contains(stdout, fmt.Sprintf("%d\trepository\tgithub.com/org/a\t", repoID)) {
		t.Errorf("repository scope id: %d %q %s", code, stdout, stderr)
	}
	code, stdout, stderr = memory("list", "--scope", "user", "--scope-id", "local", "--db", db)
	if code != 0 || stderr != "" || strings.Contains(stdout, "repository") || !strings.Contains(stdout, fmt.Sprintf("%d\tuser\tlocal\t", userID)) {
		t.Errorf("user scope id: %d %q %s", code, stdout, stderr)
	}
	code, stdout, stderr = memory("list", "--scope", "repository", "--scope-id", "github.com/org/other", "--db", db)
	if code != 0 || stdout != "ID\tSCOPE\tSCOPE_ID\tSTATE\tRETENTION\tUPDATED\tCREATED_BY\tTITLE\n" {
		t.Errorf("other scope id: %d %q %s", code, stdout, stderr)
	}
	if code, stdout, stderr := memory("list", "--scope-id", "github.com/org/a", "--db", db); code != 2 || stdout != "" || !strings.Contains(stderr, "--scope-id needs --scope") {
		t.Errorf("scope-id without scope: %d %q %s", code, stdout, stderr)
	}
}
