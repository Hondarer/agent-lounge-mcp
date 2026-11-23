package store

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// messageFixture is the chapter 5 read model: several users, direct and room messages,
// every delivery state, a message with no recipient, and a user who is removed at the end.
type messageFixture struct {
	s                                                         *Store
	t0, tSolo, tQueued, tHalf, tAck, tRoom                    time.Time
	tDave, tCarol, tEve, tLate, tOrphan, expiredAt            time.Time
	expire, solo, goneDirect, tinyRoom                        Sent
	queued, delivered, acked, room, late, orphan              Sent
	memStay, memEarly, memRepoAlice, memRepoGone, memUserGone Memory
}

func (f messageFixture) list(t *testing.T, q MessageQuery) []AdminMessage {
	t.Helper()
	if q.Limit == 0 {
		q.Limit = 100
	}
	got, err := f.s.ListMessages(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func (f messageFixture) ids(t *testing.T, filter RecordFilter, state string) []int64 {
	t.Helper()
	return adminIDs(f.list(t, MessageQuery{Filter: filter, State: state, Limit: 100, Details: true}))
}

func adminIDs(ms []AdminMessage) []int64 {
	out := make([]int64, len(ms))
	for i, m := range ms {
		out[i] = m.ID
	}
	return out
}

func sameIDs(got, want []int64) bool {
	got, want = slices.Clone(got), slices.Clone(want)
	slices.Sort(got)
	slices.Sort(want)
	return slices.Equal(got, want)
}

func findAdmin(t *testing.T, ms []AdminMessage, id int64) AdminMessage {
	t.Helper()
	for _, m := range ms {
		if m.ID == id {
			return m
		}
	}
	t.Fatalf("message %d is not in the list", id)
	return AdminMessage{}
}

func sameRecips(got, want []AdminRecipient) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i].User != want[i].User || got[i].Name != want[i].Name || got[i].State != want[i].State || !got[i].At.Equal(want[i].At) {
			return false
		}
	}
	return true
}

func exportSig(m ExportMessage) string {
	b := m.Scope.Kind + "|" + m.Scope.ID + "|" + m.Room + "|" + m.FromUser + "/" + m.From + "|" + m.ToUser + "/" + m.ToName + "|" + m.Body
	for _, d := range m.Deliveries {
		b += ";" + d.User + "/" + d.Name + ":" + d.State
	}
	return b
}

func collectExport(t *testing.T, s *Store, f RecordFilter) (msgs []ExportMessage, mems []Memory) {
	t.Helper()
	read := func(onMessage func(ExportMessage) error, onMemory func(Memory) error) error {
		if f.IsZero() {
			return s.Export(context.Background(), onMessage, onMemory)
		}
		return s.ExportSelected(context.Background(), f, onMessage, onMemory)
	}
	err := read(func(m ExportMessage) error {
		msgs = append(msgs, m)
		return nil
	}, func(m Memory) error {
		mems = append(mems, m)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return msgs, mems
}

func signatures(msgs []ExportMessage) []string {
	out := make([]string, len(msgs))
	for i, m := range msgs {
		out[i] = exportSig(m)
	}
	return out
}

// isSubseq reports whether sub appears inside all in order, not necessarily contiguously.
func isSubseq(all, sub []string) bool {
	i := 0
	for _, s := range all {
		if i < len(sub) && s == sub[i] {
			i++
		}
	}
	return i == len(sub)
}

func buildMessageFixture(t *testing.T) messageFixture {
	t.Helper()
	s, _ := openLocal(t)
	clock := useClock(s)
	ctx := context.Background()
	var f messageFixture
	f.s = s

	aliceU, bobU := addUser(t, s, "alice"), addUser(t, s, "bob")
	carolU, daveU := addUser(t, s, "carol"), addUser(t, s, "dave")
	eveU, onlyU, goneU := addUser(t, s, "eve"), addUser(t, s, "only"), addUser(t, s, "gone")
	repoA, repoTiny := &Location{Origin: "github.com/org/a"}, &Location{Origin: "github.com/org/tiny"}
	mainBr := "main"
	joinAt := func(user User, name string, loc *Location, branch *string) Session {
		t.Helper()
		return joinWith(t, s, JoinParams{UserID: user.ID, Name: name, Location: loc, Branch: branch})
	}
	alice := joinAt(aliceU, "dev", repoA, &mainBr)
	aliceExtra := joinAt(aliceU, "extra", nil, nil)
	bob := joinAt(bobU, "review", repoA, &mainBr)
	bobTiny := joinAt(bobU, "tiny", repoTiny, nil)
	carol := joinAt(carolU, "ops", repoA, &mainBr)
	dave := joinAt(daveU, "qa", repoA, &mainBr)
	eve := joinAt(eveU, "wait", repoA, &mainBr)
	solo := joinAt(onlyU, "solo", nil, nil)
	gone := joinAt(goneU, "old", repoTiny, nil)

	mustTo := func(sender Session, scope, user, name, body string) Sent {
		t.Helper()
		sent, err := sendTo(s, sender, scope, user, name, body)
		if err != nil {
			t.Fatal(err)
		}
		return sent
	}
	mustRoom := func(sender Session, scope, body string) Sent {
		t.Helper()
		sent, err := sendRoom(s, sender, scope, LoungeRoom, body)
		if err != nil {
			t.Fatal(err)
		}
		return sent
	}

	// One queued delivery, then expire it before any other message is queued, so Cleanup
	// does not expire the deliveries the later cases need to stay queued.
	f.t0 = clock.now()
	f.expire = mustTo(bob, ScopeRepository, "alice", "dev", "expire-me")
	clock.advance(2 * time.Hour)
	f.expiredAt = clock.now()
	cleaned, err := s.Cleanup(ctx, CleanupPolicy{ExpireAfter: time.Hour})
	if err != nil || cleaned.Expired != 1 {
		t.Fatalf("expire: %+v %v", cleaned, err)
	}

	f.tSolo = clock.now()
	f.solo = mustRoom(solo, ScopeUser, "nobody-home")
	if len(f.solo.Recipients) != 0 {
		t.Fatalf("solo recipients: %+v", f.solo.Recipients)
	}
	f.goneDirect = mustTo(gone, ScopeRepository, "bob", "tiny", "gone-direct")
	f.tinyRoom = mustRoom(bobTiny, ScopeRepository, "tiny-room")
	if len(f.tinyRoom.Recipients) != 1 || f.tinyRoom.Recipients[0].User != "gone" {
		t.Fatalf("tiny room recipients: %+v", f.tinyRoom.Recipients)
	}

	clock.advance(time.Second)
	f.tQueued = clock.now()
	f.queued = mustTo(alice, ScopeUser, "alice", "extra", "direct-queued")
	f.memStay = createMemory(t, s, alice, ScopeUser, "stay-body")
	f.memEarly = createMemory(t, s, alice, ScopeUser, "early-body")
	f.memRepoAlice = createMemory(t, s, alice, ScopeRepository, "repo-alice")
	f.memRepoGone = createMemory(t, s, gone, ScopeRepository, "repo-gone")
	f.memUserGone = createMemory(t, s, gone, ScopeUser, "user-gone")

	clock.advance(500 * time.Millisecond)
	f.tHalf = clock.now()
	f.delivered = mustTo(aliceExtra, ScopeUser, "alice", "dev", "direct-delivered")

	clock.advance(500 * time.Millisecond)
	f.tAck = clock.now()
	f.acked = mustTo(alice, ScopeRepository, "bob", "review", "repo-acked")
	if got := fetch(t, s, bob); len(got) != 1 || got[0].MessageID != f.acked.MessageID {
		t.Fatalf("bob fetch: %+v", got)
	}
	if ackedIDs, missing := ack(t, s, bob, f.acked.MessageID); len(missing) != 0 || !slices.Equal(ackedIDs, []int64{f.acked.MessageID}) {
		t.Fatalf("ack bob: %v %v", ackedIDs, missing)
	}

	clock.advance(time.Second)
	f.tRoom = clock.now()
	f.room = mustRoom(alice, ScopeBranch, "branch-room")
	var gotRecips []string
	for _, r := range f.room.Recipients {
		gotRecips = append(gotRecips, r.User+"/"+r.Name)
	}
	slices.Sort(gotRecips)
	if !slices.Equal(gotRecips, []string{"bob/review", "carol/ops", "dave/qa", "eve/wait"}) {
		t.Fatalf("branch recipients: %v", gotRecips)
	}

	clock.advance(time.Second)
	f.tDave = clock.now()
	if got := fetch(t, s, dave); len(got) != 1 || got[0].MessageID != f.room.MessageID {
		t.Fatalf("dave fetch: %+v", got)
	}
	clock.advance(time.Second)
	f.tCarol = clock.now()
	if got := fetch(t, s, carol); len(got) != 1 || got[0].MessageID != f.room.MessageID {
		t.Fatalf("carol fetch: %+v", got)
	}
	if ackedIDs, missing := ack(t, s, carol, f.room.MessageID); len(missing) != 0 || !slices.Equal(ackedIDs, []int64{f.room.MessageID}) {
		t.Fatalf("ack carol: %v %v", ackedIDs, missing)
	}
	clock.advance(time.Second)
	f.tEve = clock.now()
	res, err := s.db.ExecContext(ctx,
		`UPDATE deliveries SET state = 'expired', expired_at = ?
		 WHERE message_id = ? AND recipient_session_id = ? AND state = 'queued'`,
		millis(f.tEve), f.room.MessageID, eve.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		t.Fatalf("expired rows = %d", n)
	}

	// Fetching moves direct-delivered to delivered at tLate without changing its sent time.
	// The memory update and the late message share that instant, so a bound at tLate keeps
	// them and not the earlier message.
	clock.advance(time.Second)
	f.tLate = clock.now()
	if got := fetch(t, s, alice); len(got) != 1 || got[0].MessageID != f.delivered.MessageID {
		t.Fatalf("alice fetch: %+v", got)
	}
	body := "early-updated"
	if _, err := s.UpdateMemory(ctx, alice.ID, f.memEarly.ID, MemoryInput{Body: &body}); err != nil {
		t.Fatal(err)
	}
	f.late = mustTo(aliceExtra, ScopeUser, "alice", "dev", "late-message")

	clock.advance(time.Second)
	f.tOrphan = clock.now()
	f.orphan = mustTo(carol, ScopeRepository, "dave", "qa", "orphan-direct")
	return f
}

func TestMessageListShowAndFilters(t *testing.T) {
	ctx := context.Background()
	f := buildMessageFixture(t)
	s := f.s

	all := f.list(t, MessageQuery{Limit: 100, Details: true})
	ids := adminIDs(all)
	if len(all) != 10 {
		t.Fatalf("list = %d messages, want 10: %v", len(all), ids)
	}
	for i := 1; i < len(ids); i++ {
		if ids[i-1] <= ids[i] {
			t.Fatalf("newest-first order: %v", ids)
		}
	}
	if ids[0] != f.orphan.MessageID || ids[len(ids)-1] != f.expire.MessageID {
		t.Fatalf("ends: %v", ids)
	}
	for _, m := range all {
		if m.Body != "" {
			t.Errorf("list included the body of %d", m.ID)
		}
	}
	plain := f.list(t, MessageQuery{Limit: 100})
	if plain[0].Deliveries != nil {
		t.Error("list without --details loaded recipients")
	}

	expire := findAdmin(t, all, f.expire.MessageID)
	if expire.FromUser != "bob" || expire.FromName != "review" || expire.Room != "" || expire.ToUser != "alice" || expire.ToName != "dev" ||
		expire.Scope != (Scope{ScopeRepository, "github.com/org/a"}) || !expire.SentAt.Equal(f.t0) ||
		expire.Queued != 0 || expire.Delivered != 0 || expire.Acked != 0 || expire.Expired != 1 ||
		!sameRecips(expire.Deliveries, []AdminRecipient{{User: "alice", Name: "dev", State: StateExpired, At: f.expiredAt}}) {
		t.Errorf("expired direct: %+v", expire)
	}
	solo := findAdmin(t, all, f.solo.MessageID)
	if solo.Room != LoungeRoom || solo.ToUser != "" || solo.Scope != (Scope{ScopeUser, "only"}) ||
		solo.Queued != 0 || solo.Delivered != 0 || solo.Acked != 0 || solo.Expired != 0 ||
		solo.Deliveries == nil || len(solo.Deliveries) != 0 {
		t.Errorf("message with no recipient: %+v", solo)
	}
	room := findAdmin(t, all, f.room.MessageID)
	if room.Room != LoungeRoom || room.Scope != (Scope{ScopeBranch, "github.com/org/a:main"}) ||
		room.Queued != 1 || room.Delivered != 1 || room.Acked != 1 || room.Expired != 1 ||
		!sameRecips(room.Deliveries, []AdminRecipient{
			{User: "bob", Name: "review", State: StateQueued, At: f.tRoom},
			{User: "carol", Name: "ops", State: StateAcked, At: f.tCarol},
			{User: "dave", Name: "qa", State: StateDelivered, At: f.tDave},
			{User: "eve", Name: "wait", State: StateExpired, At: f.tEve},
		}) {
		t.Errorf("branch room: %+v deliveries %+v", room, room.Deliveries)
	}
	if got := findAdmin(t, all, f.delivered.MessageID); !got.SentAt.Equal(f.tHalf) || got.Delivered != 1 ||
		!sameRecips(got.Deliveries, []AdminRecipient{{User: "alice", Name: "dev", State: StateDelivered, At: f.tLate}}) {
		t.Errorf("delivered direct: sent %s deliveries %+v", got.SentAt, got.Deliveries)
	}
	if got := findAdmin(t, all, f.acked.MessageID); got.Acked != 1 || got.Scope.ID != "github.com/org/a" ||
		!sameRecips(got.Deliveries, []AdminRecipient{{User: "bob", Name: "review", State: StateAcked, At: f.tAck}}) {
		t.Errorf("acked direct: %+v", got)
	}

	shown, err := s.MessageAdmin(ctx, f.room.MessageID)
	if err != nil || shown.Body != "branch-room" || !sameRecips(shown.Deliveries, room.Deliveries) || shown.Queued != 1 {
		t.Errorf("show: %+v %v", shown, err)
	}
	if _, err := s.MessageAdmin(ctx, 999999); !errors.Is(err, ErrMessageNotFound) || !strings.Contains(err.Error(), "no such message") {
		t.Errorf("missing message: %v", err)
	}

	page := f.list(t, MessageQuery{Limit: 2, BeforeID: ids[0]})
	if !slices.Equal(adminIDs(page), ids[1:3]) {
		t.Errorf("limit/before-id: got %v want %v", adminIDs(page), ids[1:3])
	}
	if got := f.list(t, MessageQuery{Limit: 1}); len(got) != 1 || got[0].ID != ids[0] {
		t.Errorf("limit 1: %v", adminIDs(got))
	}

	if got := f.ids(t, RecordFilter{}, StateQueued); !sameIDs(got, []int64{f.queued.MessageID, f.goneDirect.MessageID, f.tinyRoom.MessageID, f.room.MessageID, f.late.MessageID, f.orphan.MessageID}) {
		t.Errorf("state queued: %v", got)
	}
	if got := f.ids(t, RecordFilter{}, StateDelivered); !sameIDs(got, []int64{f.delivered.MessageID, f.room.MessageID}) {
		t.Errorf("state delivered: %v", got)
	}
	if got := f.ids(t, RecordFilter{}, StateAcked); !sameIDs(got, []int64{f.acked.MessageID, f.room.MessageID}) {
		t.Errorf("state acked: %v", got)
	}
	if got := f.ids(t, RecordFilter{}, StateExpired); !sameIDs(got, []int64{f.expire.MessageID, f.room.MessageID}) {
		t.Errorf("state expired: %v", got)
	}
	for _, state := range DeliveryStates {
		if slices.Contains(f.ids(t, RecordFilter{}, state), f.solo.MessageID) {
			t.Errorf("a message with no recipient matched --state %s", state)
		}
	}

	if got := f.ids(t, RecordFilter{User: "only"}, ""); !sameIDs(got, []int64{f.solo.MessageID}) {
		t.Errorf("user only: %v", got)
	}
	if got := f.ids(t, RecordFilter{User: "eve"}, ""); !sameIDs(got, []int64{f.room.MessageID}) {
		t.Errorf("user eve (delivery recipient): %v", got)
	}
	if got := f.ids(t, RecordFilter{User: "alice"}, ""); !sameIDs(got, []int64{f.expire.MessageID, f.queued.MessageID, f.delivered.MessageID, f.acked.MessageID, f.room.MessageID, f.late.MessageID}) {
		t.Errorf("user alice: %v", got)
	}
	if got := f.ids(t, RecordFilter{User: "bob"}, ""); !sameIDs(got, []int64{f.expire.MessageID, f.acked.MessageID, f.room.MessageID, f.goneDirect.MessageID, f.tinyRoom.MessageID}) {
		t.Errorf("user bob: %v", got)
	}
	if got := f.ids(t, RecordFilter{User: "gone"}, ""); !sameIDs(got, []int64{f.goneDirect.MessageID, f.tinyRoom.MessageID}) {
		t.Errorf("user gone: %v", got)
	}
	if got := f.ids(t, RecordFilter{User: "nobody"}, ""); len(got) != 0 {
		t.Errorf("unknown user: %v", got)
	}
	if got := f.ids(t, RecordFilter{User: "bob", Scope: ScopeRepository, HasScopeID: true, ScopeID: "github.com/org/a"}, ""); !sameIDs(got, []int64{f.expire.MessageID, f.acked.MessageID}) {
		t.Errorf("user and scope together: %v", got)
	}
	if got := f.ids(t, RecordFilter{Scope: ScopeBranch, HasScopeID: true, ScopeID: "github.com/org/a:main"}, ""); !sameIDs(got, []int64{f.room.MessageID}) {
		t.Errorf("branch scope id: %v", got)
	}
	if got := f.ids(t, RecordFilter{Scope: ScopeRepository, HasScopeID: true, ScopeID: "github.com/org/a"}, ""); !sameIDs(got, []int64{f.expire.MessageID, f.acked.MessageID, f.orphan.MessageID}) {
		t.Errorf("repository scope id: %v", got)
	}
	if got := f.ids(t, RecordFilter{Scope: ScopeUser, HasScopeID: true, ScopeID: "only"}, ""); !sameIDs(got, []int64{f.solo.MessageID}) {
		t.Errorf("user scope id: %v", got)
	}
	if got := f.ids(t, RecordFilter{Scope: ScopeRepository, HasScopeID: true, ScopeID: "github.com/org/missing"}, ""); len(got) != 0 {
		t.Errorf("other scope id: %v", got)
	}
	if got := f.ids(t, RecordFilter{Scope: ScopeUser, HasScopeID: true, ScopeID: ""}, ""); len(got) != 0 {
		t.Errorf("empty scope id: %v", got)
	}

	// since is inclusive and until is exclusive, on the sent time. Delivering direct-delivered
	// at tLate does not bring it into a window that starts at tLate.
	if got := f.ids(t, RecordFilter{Since: f.tHalf, Until: f.tHalf}, ""); len(got) != 0 {
		t.Errorf("until equal to since: %v", got)
	}
	if got := f.ids(t, RecordFilter{Since: f.tHalf, Until: f.tHalf.Add(time.Millisecond)}, ""); !sameIDs(got, []int64{f.delivered.MessageID}) {
		t.Errorf("half-open millisecond window: %v", got)
	}
	if got := f.ids(t, RecordFilter{Since: f.tQueued, Until: f.tHalf}, ""); !slices.Contains(got, f.queued.MessageID) || slices.Contains(got, f.delivered.MessageID) {
		t.Errorf("since inclusive, until exclusive: %v", got)
	}
	if got := f.ids(t, RecordFilter{Since: f.tLate, Until: f.tOrphan}, ""); !sameIDs(got, []int64{f.late.MessageID}) {
		t.Errorf("sent time, not delivery time: %v", got)
	}
	jst := f.tLate.In(time.FixedZone("JST", 9*3600))
	if got := f.ids(t, RecordFilter{Since: jst, Until: f.tOrphan}, ""); !sameIDs(got, []int64{f.late.MessageID}) {
		t.Errorf("offset equal to the same instant: %v", got)
	}
	if got := f.ids(t, RecordFilter{Since: f.tLate, Until: f.t0}, ""); len(got) != 0 {
		t.Errorf("since after until: %v", got)
	}

	_, memStayWindow := collectExport(t, s, RecordFilter{Since: f.tLate, Until: f.tOrphan})
	if !slices.Equal(memoryIDs(memStayWindow), []int64{f.memEarly.ID}) || memStayWindow[0].Body != "early-updated" || !memStayWindow[0].UpdatedAt.Equal(f.tLate) || !memStayWindow[0].CreatedAt.Equal(f.tQueued) {
		t.Errorf("memory window uses updated time: %+v", memStayWindow)
	}
	_, memUntil := collectExport(t, s, RecordFilter{Until: f.tLate})
	if slices.Contains(memoryIDs(memUntil), f.memEarly.ID) || !slices.Contains(memoryIDs(memUntil), f.memStay.ID) {
		t.Errorf("until excludes the memory updated at that instant: %v", memoryIDs(memUntil))
	}

	if _, err := s.ListMessages(ctx, MessageQuery{Limit: 0}); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Errorf("limit 0: %v", err)
	}
	if _, err := s.ListMessages(ctx, MessageQuery{Limit: 1, State: "read"}); err == nil || !strings.Contains(err.Error(), "unknown delivery state") {
		t.Errorf("bad state: %v", err)
	}
	if _, err := s.ListMessages(ctx, MessageQuery{Limit: 1, Filter: RecordFilter{Scope: "global"}}); !errors.Is(err, ErrInvalidScope) {
		t.Errorf("bad scope: %v", err)
	}
	if err := s.ExportSelected(ctx, RecordFilter{Scope: "nope"}, func(ExportMessage) error { return nil }, nil); !errors.Is(err, ErrInvalidScope) {
		t.Errorf("export bad scope: %v", err)
	}

	exported, exportedMem := collectExport(t, s, RecordFilter{})
	var direct []ExportMessage
	var directMem []Memory
	if err := s.ExportSelected(ctx, RecordFilter{}, func(m ExportMessage) error { direct = append(direct, m); return nil }, func(m Memory) error { directMem = append(directMem, m); return nil }); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(signatures(exported), signatures(direct)) || !slices.Equal(memoryIDs(exportedMem), memoryIDs(directMem)) {
		t.Fatalf("ExportSelected with no filter:\n%v\nwant\n%v", signatures(direct), signatures(exported))
	}
	for _, m := range exported {
		if m.Body == "nobody-home" && len(m.Deliveries) != 0 {
			t.Errorf("exported message with no recipient: %+v", m)
		}
	}
	eveMsgs, _ := collectExport(t, s, RecordFilter{User: "eve"})
	if len(eveMsgs) != 1 || eveMsgs[0].ID != f.room.MessageID || len(eveMsgs[0].Deliveries) != 4 || !isSubseq(signatures(exported), signatures(eveMsgs)) {
		t.Errorf("filtered export dropped deliveries or order: %+v", eveMsgs)
	}
	repoMsgs, repoMem := collectExport(t, s, RecordFilter{Scope: ScopeRepository, HasScopeID: true, ScopeID: "github.com/org/a"})
	if !isSubseq(signatures(exported), signatures(repoMsgs)) {
		t.Errorf("filtered message order: %v", signatures(repoMsgs))
	}
	if _, aliceMem := collectExport(t, s, RecordFilter{User: "alice"}); !isSubseq(signaturesOfMem(exportedMem), signaturesOfMem(aliceMem)) ||
		!sameIDs(memoryIDs(aliceMem), []int64{f.memStay.ID, f.memEarly.ID, f.memRepoAlice.ID}) {
		t.Errorf("memories for alice: %v", memoryIDs(aliceMem))
	}
	if !sameIDs(memoryIDs(repoMem), []int64{f.memRepoAlice.ID}) {
		t.Errorf("memories for the repository: %v", memoryIDs(repoMem))
	}

	// A direct recipient still matches after the delivery row is gone. A delivery recipient
	// does not: that row was the only stored name tying the message to the user.
	if _, err := s.db.ExecContext(ctx, "DELETE FROM deliveries WHERE message_id = ?", f.orphan.MessageID); err != nil {
		t.Fatal(err)
	}
	if got := f.ids(t, RecordFilter{User: "dave"}, ""); !sameIDs(got, []int64{f.room.MessageID, f.orphan.MessageID}) {
		t.Errorf("direct recipient after the delivery was deleted: %v", got)
	}
	orphan := findAdmin(t, f.list(t, MessageQuery{Limit: 100, Details: true}), f.orphan.MessageID)
	if orphan.Queued != 0 || orphan.ToUser != "dave" || orphan.ToName != "qa" || len(orphan.Deliveries) != 0 ||
		slices.Contains(f.ids(t, RecordFilter{}, StateQueued), f.orphan.MessageID) {
		t.Errorf("orphan after deleting its delivery: %+v", orphan)
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM deliveries WHERE message_id = ? AND recipient_session_id = (SELECT id FROM sessions WHERE name = 'wait')", f.room.MessageID); err != nil {
		t.Fatal(err)
	}
	if got := f.ids(t, RecordFilter{User: "eve"}, ""); len(got) != 0 {
		t.Errorf("delivery recipient after the delivery was deleted: %v", got)
	}

	// Removing the user deletes the messages that user sent. A room message sent by someone
	// else stays, without the removed user's delivery, so the stored name no longer matches.
	// A repository memory keeps the creator's name; the user's own memory does not.
	if _, err := s.RemoveUser(ctx, "gone"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.MessageAdmin(ctx, f.goneDirect.MessageID); !errors.Is(err, ErrMessageNotFound) {
		t.Errorf("message sent by the removed user: %v", err)
	}
	if got := f.ids(t, RecordFilter{User: "gone"}, ""); len(got) != 0 {
		t.Errorf("removed user still matched a message: %v", got)
	}
	tiny := findAdmin(t, f.list(t, MessageQuery{Limit: 100, Details: true}), f.tinyRoom.MessageID)
	if tiny.FromUser != "bob" || tiny.FromName != "tiny" || tiny.Queued != 0 || len(tiny.Deliveries) != 0 || tiny.Scope.ID != "github.com/org/tiny" {
		t.Errorf("room message after the only recipient was removed: %+v", tiny)
	}
	goneMsgs, goneMem := collectExport(t, s, RecordFilter{User: "gone"})
	if len(goneMsgs) != 0 || !slices.Equal(memoryIDs(goneMem), []int64{f.memRepoGone.ID}) || goneMem[0].CreatedBy.User != "gone" || goneMem[0].Body != "repo-gone" {
		t.Errorf("export for the removed user: messages %v memories %+v", signatures(goneMsgs), goneMem)
	}
	if _, err := s.MemoryByID(ctx, f.memUserGone.ID); !errors.Is(err, ErrNoSuchMemory) {
		t.Errorf("user memory of the removed user: %v", err)
	}
}

func signaturesOfMem(ms []Memory) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Scope.Kind + "|" + m.Scope.ID + "|" + m.Body
	}
	return out
}

// The default page is 100 messages, newest first, and --before-id continues from there.
func TestListMessagesPagesByID(t *testing.T) {
	s, local := openLocal(t)
	ann, ben := join(t, s, local, "ann"), join(t, s, local, "ben")
	ids := make([]int64, 101)
	for i := range ids {
		sent, err := sendTo(s, ann, ScopeUser, LocalUser, "ben", "n")
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = sent.MessageID
	}
	if unread(t, s, ben) != 101 {
		t.Fatalf("unread = %d", unread(t, s, ben))
	}
	page, err := s.ListMessages(context.Background(), MessageQuery{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	want := slices.Clone(ids[1:])
	slices.Reverse(want)
	if !slices.Equal(adminIDs(page), want) {
		t.Fatalf("first page = %v, want %v", adminIDs(page), want)
	}
	rest, err := s.ListMessages(context.Background(), MessageQuery{Limit: 100, BeforeID: page[len(page)-1].ID})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(adminIDs(rest), []int64{ids[0]}) {
		t.Errorf("next page = %v, want %v", adminIDs(rest), ids[0])
	}
}
