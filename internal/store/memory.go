package store

import (
	"cmp"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"modernc.org/sqlite"
)

// Memories are short-lived pieces of knowledge that sessions share within a scope, as messages
// are shared: the memories of scope user are shared by the sessions of one user, those of
// scope repository by the sessions that declare one repository unit, and those of scope branch
// by the sessions that declare one repository unit and one branch, whatever their user.
// Every session that can use a scope can read, update, pin and delete all of its memories.
// A memory of a scope the session cannot use is reported as missing, so that a session does
// not learn whether it exists.
//
// Memories fade and are deleted by the forgetting model (retention.go). Reading a memory by ID
// and finding it near the top of a search count as references, which slow its fading; see
// countReference for the rules.

// init registers the SQL function fold_case, which returns its text argument with every letter
// in lower case (strings.ToLower), for the search of terms shorter than a trigram: the trigram
// index ignores the case of all letters, not only of ASCII ones, and so must the scan that
// stands in for it. Functions registered with the driver reach every connection it opens.
func init() {
	sqlite.MustRegisterDeterministicScalarFunction("fold_case", 1, func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		switch v := args[0].(type) {
		case string:
			return strings.ToLower(v), nil
		case []byte:
			return strings.ToLower(string(v)), nil
		default:
			return v, nil
		}
	})
}

// Bounds of memories and searches. The body limit is Limits.MaxMemoryBytes.
const (
	MaxMemoryTitleRunes = 200
	MaxMemoryTags       = 10
	MaxMemoryTagRunes   = 32
	MaxQueryTerms       = 10
	MaxQueryRunes       = 200
	// SearchCountedResults is how many of the first results of a search with terms count as
	// references.
	SearchCountedResults = 3
	// ReferenceWindow is how long a counted reference of a session to a memory keeps further
	// references of that session to it from being counted.
	ReferenceWindow = 24 * time.Hour
	// minTrigramRunes is the shortest term the trigram index can find (see SearchMemories).
	minTrigramRunes = 3
)

// Errors of the memory functions.
var (
	// ErrNoSuchMemory is returned for an ID that names no memory of the scopes the session
	// can use: a missing memory and a memory of another scope give the same error.
	ErrNoSuchMemory = errors.New("no such memory")
	// ErrMemoryTooLarge is returned for a body over Limits.MaxMemoryBytes.
	ErrMemoryTooLarge = errors.New("the memory body is too large")
	// ErrTooManyMemories is returned for creating a memory in a scope that holds
	// Limits.MaxMemoriesPerScope memories.
	ErrTooManyMemories = errors.New("the scope has too many memories")
	// ErrInvalidMemory is wrapped by the errors for a malformed body, title, tag or query.
	ErrInvalidMemory = errors.New("invalid memory")
)

// Author is who wrote a memory: the values the session had when it wrote, kept as they were
// because a memory outlives the session.
type Author struct {
	User, Session, Agent string
}

// Memory is a stored memory.
type Memory struct {
	ID          int64
	Scope       Scope
	Title       string
	Body        string
	Tags        []string
	Pinned      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	AccessedAt  time.Time // zero when never referenced
	AccessCount int
	CreatedBy   Author
	UpdatedBy   Author
	// Retention is the retention score and Faded whether the memory has faded, both at the
	// time the memory was read, under the store's RetentionPolicy.
	Retention float64
	Faded     bool

	forgettingCredit time.Duration
}

func (m Memory) state() retentionState {
	return retentionState{
		pinned: m.Pinned, updatedAt: m.UpdatedAt, accessedAt: m.AccessedAt,
		accessCount: m.AccessCount, credit: m.forgettingCredit,
	}
}

// assessWith fills in Retention and Faded at now under act.
func (p RetentionPolicy) assessWith(m *Memory, act scopeActivity, now time.Time) {
	st := m.state()
	m.Retention = p.scoreAt(st, act, now)
	m.Faded = p.fadedAt(st, act, now)
}

// MemoryInput holds the fields of a memory to write. A nil field is not given: creating a
// memory leaves it empty, updating one keeps its value.
type MemoryInput struct {
	Body  *string
	Title *string
	Tags  *[]string
}

// MemorySearch holds the arguments of SearchMemories.
type MemorySearch struct {
	// Query holds the search terms separated by white space. Empty lists the memories.
	Query string
	// Scope is the scope kind to search, or "" for every scope the session can use.
	Scope string
	// Tags keeps only the memories that have all of them.
	Tags []string
	// IncludeFaded includes faded memories.
	IncludeFaded bool
	// Limit is the most memories to return (at least 1).
	Limit int
}

// NormalizeTags checks tags and returns them as they are stored: lowercase, without
// duplicates, in the order given. A tag is 1 to MaxMemoryTagRunes characters of UTF-8 without
// white space, control characters or commas; a memory has at most MaxMemoryTags tags.
func NormalizeTags(tags []string) ([]string, error) {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		t := strings.ToLower(tag)
		if !utf8.ValidString(t) || t == "" || utf8.RuneCountInString(t) > MaxMemoryTagRunes ||
			strings.ContainsFunc(t, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) || r == ',' }) {
			return nil, fmt.Errorf("%w: tag %q: a tag is 1 to %d characters without spaces, control characters or commas",
				ErrInvalidMemory, tag, MaxMemoryTagRunes)
		}
		if !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	if len(out) > MaxMemoryTags {
		return nil, fmt.Errorf("%w: %d tags: a memory has at most %d tags", ErrInvalidMemory, len(out), MaxMemoryTags)
	}
	return out, nil
}

// normalizeTitle checks a title and returns it without surrounding white space.
func normalizeTitle(title string) (string, error) {
	t := strings.TrimSpace(title)
	if !utf8.ValidString(t) || strings.ContainsAny(t, "\r\n") || utf8.RuneCountInString(t) > MaxMemoryTitleRunes {
		return "", fmt.Errorf("%w: the title is one line of at most %d characters", ErrInvalidMemory, MaxMemoryTitleRunes)
	}
	return t, nil
}

// checkBody checks a memory body against the limit.
func (s *Store) checkBody(body string) error {
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("%w: the body is empty", ErrInvalidMemory)
	}
	if !utf8.ValidString(body) {
		return fmt.Errorf("%w: the body is not valid UTF-8", ErrInvalidMemory)
	}
	if len(body) > s.limits.MaxMemoryBytes {
		return fmt.Errorf("%w: %d bytes, at most %d", ErrMemoryTooLarge, len(body), s.limits.MaxMemoryBytes)
	}
	return nil
}

// memoryColumns selects a Memory from memories m; see scanMemory.
const memoryColumns = `m.id, m.scope, m.scope_id, m.title, m.body, m.pinned, m.created_at, m.updated_at, m.accessed_at,
	m.access_count, m.created_user, m.created_session, m.created_agent, m.updated_user, m.updated_session, m.updated_agent,
	m.forgetting_credit`

func scanMemory(r rowScanner) (Memory, error) {
	var m Memory
	var created, updated, accessed, credit int64
	err := r.Scan(&m.ID, &m.Scope.Kind, &m.Scope.ID, &m.Title, &m.Body, &m.Pinned, &created, &updated, &accessed,
		&m.AccessCount, &m.CreatedBy.User, &m.CreatedBy.Session, &m.CreatedBy.Agent,
		&m.UpdatedBy.User, &m.UpdatedBy.Session, &m.UpdatedBy.Agent, &credit)
	m.CreatedAt, m.UpdatedAt, m.AccessedAt = fromMillis(created), fromMillis(updated), optionalTime(accessed)
	m.forgettingCredit = time.Duration(credit) * time.Millisecond
	return m, err
}

// loadScopeActivity reads the activity row of sc, returning a zero scopeActivity when absent.
func loadScopeActivity(ctx context.Context, tx *sql.Tx, sc Scope) (scopeActivity, error) {
	var lastMs, discMs int64
	err := tx.QueryRowContext(ctx,
		"SELECT last_access, discount FROM memory_scope_activity WHERE scope = ? AND scope_id = ?",
		sc.Kind, sc.ID).Scan(&lastMs, &discMs)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return scopeActivity{}, nil
	case err != nil:
		return scopeActivity{}, err
	}
	return scopeActivity{lastAccess: fromMillis(lastMs), discount: time.Duration(discMs) * time.Millisecond}, nil
}

// touchScopeActivity closes any open quiet gap for sc at now and records now as its last
// access, returning the updated scopeActivity.
func (s *Store) touchScopeActivity(ctx context.Context, tx *sql.Tx, sc Scope, now time.Time) (scopeActivity, error) {
	nowMs := millis(now)
	nowT := fromMillis(nowMs)
	var lastMs, discMs int64
	err := tx.QueryRowContext(ctx,
		"SELECT last_access, discount FROM memory_scope_activity WHERE scope = ? AND scope_id = ?",
		sc.Kind, sc.ID).Scan(&lastMs, &discMs)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO memory_scope_activity (scope, scope_id, last_access, discount) VALUES (?, ?, ?, 0)",
			sc.Kind, sc.ID, nowMs); err != nil {
			return scopeActivity{}, err
		}
		return scopeActivity{lastAccess: nowT}, nil
	case err != nil:
		return scopeActivity{}, err
	}
	cur := scopeActivity{lastAccess: fromMillis(lastMs), discount: time.Duration(discMs) * time.Millisecond}
	openMs := s.retention.openDiscount(cur, nowT).Milliseconds()
	newDiscMs := discMs + openMs
	newLastMs := max(lastMs, nowMs)
	if _, err := tx.ExecContext(ctx,
		"UPDATE memory_scope_activity SET last_access = ?, discount = ? WHERE scope = ? AND scope_id = ?",
		newLastMs, newDiscMs, sc.Kind, sc.ID); err != nil {
		return scopeActivity{}, err
	}
	return scopeActivity{lastAccess: fromMillis(newLastMs), discount: time.Duration(newDiscMs) * time.Millisecond}, nil
}

// loadMemories returns the memories with the given IDs, with their tags, by ID. Missing IDs
// are left out.
func loadMemories(ctx context.Context, tx *sql.Tx, ids []int64) (map[int64]Memory, error) {
	out := make(map[int64]Memory, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	in, args := idList(ids)
	rows, err := tx.QueryContext(ctx, "SELECT "+memoryColumns+" FROM memories m WHERE m.id IN ("+in+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			return nil, err
		}
		m.Tags = []string{}
		out[m.ID] = m
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	trows, err := tx.QueryContext(ctx, "SELECT memory_id, tag FROM memory_tags WHERE memory_id IN ("+in+") ORDER BY memory_id, pos", args...)
	if err != nil {
		return nil, err
	}
	defer trows.Close()
	for trows.Next() {
		var id int64
		var tag string
		if err := trows.Scan(&id, &tag); err != nil {
			return nil, err
		}
		m := out[id]
		m.Tags = append(m.Tags, tag)
		out[id] = m
	}
	return out, trows.Err()
}

// idList returns a list of placeholders for ids and the arguments.
func idList(ids []int64) (string, []any) {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return strings.TrimSuffix(strings.Repeat("?,", len(ids)), ","), args
}

// loadMemory returns one memory with its tags, or ErrNoSuchMemory.
func loadMemory(ctx context.Context, tx *sql.Tx, id int64) (Memory, error) {
	ms, err := loadMemories(ctx, tx, []int64{id})
	if err != nil {
		return Memory{}, err
	}
	m, ok := ms[id]
	if !ok {
		return Memory{}, ErrNoSuchMemory
	}
	return m, nil
}

// accessibleMemory returns the memory id if the session can use its scope, and ErrNoSuchMemory
// otherwise.
func accessibleMemory(ctx context.Context, tx *sql.Tx, sess Session, id int64) (Memory, error) {
	m, err := loadMemory(ctx, tx, id)
	if err != nil {
		return Memory{}, err
	}
	if !slices.Contains(ScopesOf(sess), m.Scope) {
		return Memory{}, ErrNoSuchMemory
	}
	return m, nil
}

// writeMemoryIndex replaces the tags and the full-text index row of a memory.
func writeMemoryIndex(ctx context.Context, tx *sql.Tx, m Memory) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM memory_tags WHERE memory_id = ?", m.ID); err != nil {
		return err
	}
	for i, tag := range m.Tags {
		if _, err := tx.ExecContext(ctx, "INSERT INTO memory_tags (memory_id, pos, tag) VALUES (?, ?, ?)", m.ID, i, tag); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM memories_fts WHERE rowid = ?", m.ID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO memories_fts (rowid, title, body, tags) VALUES (?, ?, ?, ?)",
		m.ID, m.Title, m.Body, strings.Join(m.Tags, " "))
	return err
}

// deleteMemories deletes memories by ID with their index rows. Tags and references go with
// them (ON DELETE CASCADE); the full-text index is a virtual table without foreign keys.
func deleteMemories(ctx context.Context, tx *sql.Tx, ids []int64) error {
	for chunk := range slices.Chunk(ids, 500) {
		in, args := idList(chunk)
		if _, err := tx.ExecContext(ctx, "DELETE FROM memories_fts WHERE rowid IN ("+in+")", args...); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM memories WHERE id IN ("+in+")", args...); err != nil {
			return err
		}
	}
	return nil
}

func authorOf(sess Session) Author {
	return Author{User: sess.UserName, Session: sess.Name, Agent: sess.Agent}
}

// CreateMemory creates a memory of the given scope kind ("" for user) written by the session.
// in.Body is required. It fails with ErrTooManyMemories when the scope holds
// Limits.MaxMemoriesPerScope memories.
func (s *Store) CreateMemory(ctx context.Context, sessionID int64, kind string, in MemoryInput) (Memory, error) {
	kind = cmp.Or(kind, ScopeUser)
	rule, ok := scopeRules[kind]
	if !ok {
		return Memory{}, ErrInvalidScope
	}
	if in.Body == nil {
		return Memory{}, fmt.Errorf("%w: the body is empty", ErrInvalidMemory)
	}
	m := Memory{Body: *in.Body, Tags: []string{}}
	if err := s.checkBody(m.Body); err != nil {
		return Memory{}, err
	}
	var err error
	if in.Title != nil {
		if m.Title, err = normalizeTitle(*in.Title); err != nil {
			return Memory{}, err
		}
	}
	if in.Tags != nil {
		if m.Tags, err = NormalizeTags(*in.Tags); err != nil {
			return Memory{}, err
		}
	}
	err = s.tx(ctx, func(tx *sql.Tx) error {
		sess, err := loadSession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		id, ok := rule.id(sess)
		if !ok {
			return rule.unavailable
		}
		m.Scope = Scope{Kind: kind, ID: id}
		var n int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM memories WHERE scope = ? AND scope_id = ?", kind, id).Scan(&n); err != nil {
			return err
		}
		if n >= s.limits.MaxMemoriesPerScope {
			return fmt.Errorf("%w: %d memories, at most %d", ErrTooManyMemories, n, s.limits.MaxMemoriesPerScope)
		}
		var userID any
		if kind == ScopeUser {
			userID = sess.UserID
		}
		now := s.now()
		act, err := s.touchScopeActivity(ctx, tx, m.Scope, now)
		if err != nil {
			return err
		}
		m.CreatedAt, m.UpdatedAt = fromMillis(millis(now)), fromMillis(millis(now))
		m.CreatedBy, m.UpdatedBy = authorOf(sess), authorOf(sess)
		m.forgettingCredit = act.discount
		err = tx.QueryRowContext(ctx, `
			INSERT INTO memories (scope, scope_id, user_id, title, body, created_at, updated_at,
				created_user, created_session, created_agent, updated_user, updated_session, updated_agent, forgetting_credit)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`,
			kind, id, userID, m.Title, m.Body, millis(now), millis(now),
			m.CreatedBy.User, m.CreatedBy.Session, m.CreatedBy.Agent, m.UpdatedBy.User, m.UpdatedBy.Session, m.UpdatedBy.Agent,
			act.discount.Milliseconds()).
			Scan(&m.ID)
		if err != nil {
			return err
		}
		s.retention.assessWith(&m, act, now)
		return writeMemoryIndex(ctx, tx, m)
	})
	if err != nil {
		return Memory{}, err
	}
	return m, nil
}

// UpdateMemory replaces the given fields of a memory of the session's scopes and records the
// session as its last updater. At least one field must be given. It does not count as a
// reference, but a faded memory comes back, since its update time changes.
func (s *Store) UpdateMemory(ctx context.Context, sessionID, id int64, in MemoryInput) (Memory, error) {
	if in.Body == nil && in.Title == nil && in.Tags == nil {
		return Memory{}, fmt.Errorf("%w: give at least one of body, title and tags to update", ErrInvalidMemory)
	}
	var body, title string
	var tags []string
	var err error
	if in.Body != nil {
		body = *in.Body
		if err := s.checkBody(body); err != nil {
			return Memory{}, err
		}
	}
	if in.Title != nil {
		if title, err = normalizeTitle(*in.Title); err != nil {
			return Memory{}, err
		}
	}
	if in.Tags != nil {
		if tags, err = NormalizeTags(*in.Tags); err != nil {
			return Memory{}, err
		}
	}
	var m Memory
	err = s.tx(ctx, func(tx *sql.Tx) error {
		sess, err := loadSession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if m, err = accessibleMemory(ctx, tx, sess, id); err != nil {
			return err
		}
		if in.Body != nil {
			m.Body = body
		}
		if in.Title != nil {
			m.Title = title
		}
		if in.Tags != nil {
			m.Tags = tags
		}
		now := s.now()
		act, err := s.touchScopeActivity(ctx, tx, m.Scope, now)
		if err != nil {
			return err
		}
		m.UpdatedAt, m.UpdatedBy, m.forgettingCredit = fromMillis(millis(now)), authorOf(sess), act.discount
		if _, err := tx.ExecContext(ctx, `
			UPDATE memories SET title = ?, body = ?, updated_at = ?, updated_user = ?, updated_session = ?, updated_agent = ?, forgetting_credit = ?
			WHERE id = ?`, m.Title, m.Body, millis(now), m.UpdatedBy.User, m.UpdatedBy.Session, m.UpdatedBy.Agent, act.discount.Milliseconds(), m.ID); err != nil {
			return err
		}
		s.retention.assessWith(&m, act, now)
		return writeMemoryIndex(ctx, tx, m)
	})
	if err != nil {
		return Memory{}, err
	}
	return m, nil
}

// GetMemory returns a memory of the session's scopes and counts the read as a reference. A
// faded memory is always counted, so that reading it brings it back; the result is the memory
// after counting.
func (s *Store) GetMemory(ctx context.Context, sessionID, id int64) (Memory, error) {
	var m Memory
	err := s.tx(ctx, func(tx *sql.Tx) error {
		sess, err := loadSession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if m, err = accessibleMemory(ctx, tx, sess, id); err != nil {
			return err
		}
		now := s.now()
		act, err := s.touchScopeActivity(ctx, tx, m.Scope, now)
		if err != nil {
			return err
		}
		counted, err := s.countReference(ctx, tx, m.ID, sess.ID, now, act.discount, s.retention.fadedAt(m.state(), act, now))
		if err != nil {
			return err
		}
		if counted {
			m.AccessCount++
			m.AccessedAt = fromMillis(millis(now))
			m.forgettingCredit = act.discount
		}
		s.retention.assessWith(&m, act, now)
		return nil
	})
	if err != nil {
		return Memory{}, err
	}
	return m, nil
}

// countReference counts a reference of a session to a memory: its reference count grows by one,
// its last reference becomes now, and its forgetting_credit becomes credit. A session's
// references to one memory are counted at most once in ReferenceWindow, so that a session that
// reads or searches again and again does not keep a memory alive by itself. force counts the
// reference regardless (reading a faded memory by ID). It reports whether the reference was
// counted.
func (s *Store) countReference(ctx context.Context, tx *sql.Tx, memoryID, sessionID int64, now time.Time, credit time.Duration, force bool) (bool, error) {
	if !force {
		var at int64
		err := tx.QueryRowContext(ctx, "SELECT counted_at FROM memory_refs WHERE memory_id = ? AND session_id = ?", memoryID, sessionID).Scan(&at)
		switch {
		case errors.Is(err, sql.ErrNoRows):
		case err != nil:
			return false, err
		case at > millis(now.Add(-ReferenceWindow)):
			return false, nil
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO memory_refs (memory_id, session_id, counted_at) VALUES (?, ?, ?)
		ON CONFLICT (memory_id, session_id) DO UPDATE SET counted_at = excluded.counted_at`, memoryID, sessionID, millis(now)); err != nil {
		return false, err
	}
	_, err := tx.ExecContext(ctx, "UPDATE memories SET access_count = access_count + 1, accessed_at = ?, forgetting_credit = ? WHERE id = ?",
		millis(now), credit.Milliseconds(), memoryID)
	return err == nil, err
}

// PinMemory pins or unpins a memory of the session's scopes. A pinned memory neither fades nor
// is deleted. Unpinning sets its last reference to now and updates forgetting_credit to the
// scope's current discount, so that a memory pinned for a long time does not fade, or get
// deleted, right after it is unpinned; its reference count is unchanged. Setting the state it
// already has changes nothing to the memory (the scope access is still recorded).
func (s *Store) PinMemory(ctx context.Context, sessionID, id int64, pinned bool) (Memory, error) {
	var m Memory
	err := s.tx(ctx, func(tx *sql.Tx) error {
		sess, err := loadSession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if m, err = accessibleMemory(ctx, tx, sess, id); err != nil {
			return err
		}
		now := s.now()
		act, err := s.touchScopeActivity(ctx, tx, m.Scope, now)
		if err != nil {
			return err
		}
		if m.Pinned != pinned {
			m.Pinned = pinned
			if pinned {
				_, err = tx.ExecContext(ctx, "UPDATE memories SET pinned = 1 WHERE id = ?", m.ID)
			} else {
				m.AccessedAt = fromMillis(millis(now))
				m.forgettingCredit = act.discount
				_, err = tx.ExecContext(ctx, "UPDATE memories SET pinned = 0, accessed_at = ?, forgetting_credit = ? WHERE id = ?",
					millis(now), act.discount.Milliseconds(), m.ID)
			}
			if err != nil {
				return err
			}
		}
		s.retention.assessWith(&m, act, now)
		return nil
	})
	if err != nil {
		return Memory{}, err
	}
	return m, nil
}

// ForgetMemory deletes a memory of the session's scopes at once.
func (s *Store) ForgetMemory(ctx context.Context, sessionID, id int64) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		sess, err := loadSession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		m, err := accessibleMemory(ctx, tx, sess, id)
		if err != nil {
			return err
		}
		if _, err := s.touchScopeActivity(ctx, tx, m.Scope, s.now()); err != nil {
			return err
		}
		return deleteMemories(ctx, tx, []int64{id})
	})
}

// searchTerms splits a query into its terms.
func searchTerms(query string) ([]string, error) {
	if !utf8.ValidString(query) || utf8.RuneCountInString(query) > MaxQueryRunes {
		return nil, fmt.Errorf("%w: the query is at most %d characters", ErrInvalidMemory, MaxQueryRunes)
	}
	terms := strings.Fields(query)
	if len(terms) > MaxQueryTerms {
		return nil, fmt.Errorf("%w: the query has %d terms, at most %d", ErrInvalidMemory, len(terms), MaxQueryTerms)
	}
	return terms, nil
}

// ftsPhrase quotes a term as an FTS5 string, so that it is searched as the characters it has:
// no character of a term is FTS5 query syntax.
func ftsPhrase(term string) string {
	return `"` + strings.ReplaceAll(term, `"`, `""`) + `"`
}

// searchCandidate is a memory that matched a search, with what ordering needs.
type searchCandidate struct {
	id        int64
	state     retentionState
	bm25      float64
	relevance float64
	score     float64
	retention float64
	faded     bool
}

// SearchMemories finds the memories of the session's scopes that contain every term of the
// query, or lists them when the query has no term, and returns at most q.Limit of them, best
// first, and whether more matched.
//
// A term matches a memory whose title, body or tags contain it, ignoring the case of letters;
// every character of the term is taken as it is. The trigram index finds only terms of three
// or more characters (it indexes runs of three), so a shorter term, common in Japanese (such
// as 認証), is searched by scanning the text of the memories that the other conditions leave.
//
// Results with a term of three or more characters are ordered by relevance times retention
// score, where relevance is the bm25 rank scaled so that the best match of the search is 1.
// Other results are ordered by retention score. Ties go to the later update, then the larger
// ID. Faded memories are left out unless q.IncludeFaded.
//
// When the query has terms, the first SearchCountedResults results count as references
// (countReference), except those that have faded. The order and the values returned are those before counting.
func (s *Store) SearchMemories(ctx context.Context, sessionID int64, q MemorySearch) (memories []Memory, more bool, err error) {
	terms, err := searchTerms(q.Query)
	if err != nil {
		return nil, false, err
	}
	var tags []string
	if len(q.Tags) > 0 {
		if tags, err = NormalizeTags(q.Tags); err != nil {
			return nil, false, err
		}
	}
	if q.Scope != "" && !ValidScope(q.Scope) {
		return nil, false, ErrInvalidScope
	}
	limit := max(q.Limit, 1)
	err = s.tx(ctx, func(tx *sql.Tx) error {
		sess, err := loadSession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		var scopes []Scope
		if q.Scope != "" {
			rule := scopeRules[q.Scope]
			id, ok := rule.id(sess)
			if !ok {
				return rule.unavailable
			}
			scopes = []Scope{{Kind: q.Scope, ID: id}}
		} else {
			scopes = ScopesOf(sess)
		}

		now := s.now()
		acts := make(map[Scope]scopeActivity, len(scopes))
		for _, sc := range scopes {
			act, err := s.touchScopeActivity(ctx, tx, sc, now)
			if err != nil {
				return err
			}
			acts[sc] = act
		}

		var where []string
		var args []any
		var scopeConds []string
		for _, sc := range scopes {
			scopeConds = append(scopeConds, "(m.scope = ? AND m.scope_id = ?)")
			args = append(args, sc.Kind, sc.ID)
		}
		where = append(where, "("+strings.Join(scopeConds, " OR ")+")")
		var phrases, short []string
		for _, t := range terms {
			if utf8.RuneCountInString(t) >= minTrigramRunes {
				phrases = append(phrases, ftsPhrase(t))
			} else {
				short = append(short, t)
			}
		}
		from := "memories m"
		rank := "0"
		if len(terms) > 0 {
			from += " JOIN memories_fts ON memories_fts.rowid = m.id"
		}
		if len(phrases) > 0 {
			where = append(where, "memories_fts MATCH ?")
			args = append(args, strings.Join(phrases, " "))
			rank = "bm25(memories_fts)"
		}
		for _, t := range short {
			// SQLite's lower() folds ASCII letters only; fold_case folds every letter, as the
			// trigram index does (see init).
			where = append(where, "instr(fold_case(memories_fts.title || char(10) || memories_fts.body || char(10) || memories_fts.tags), fold_case(?)) > 0")
			args = append(args, t)
		}
		if len(tags) > 0 {
			in := strings.TrimSuffix(strings.Repeat("?,", len(tags)), ",")
			where = append(where, "(SELECT count(*) FROM memory_tags t WHERE t.memory_id = m.id AND t.tag IN ("+in+")) = ?")
			for _, t := range tags {
				args = append(args, t)
			}
			args = append(args, len(tags))
		}
		rows, err := tx.QueryContext(ctx, "SELECT m.id, m.scope, m.scope_id, m.pinned, m.updated_at, m.accessed_at, m.access_count, m.forgetting_credit, "+rank+
			" FROM "+from+" WHERE "+strings.Join(where, " AND "), args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		var cands []searchCandidate
		best := 0.0
		for rows.Next() {
			var c searchCandidate
			var sc Scope
			var updated, accessed, credit int64
			if err := rows.Scan(&c.id, &sc.Kind, &sc.ID, &c.state.pinned, &updated, &accessed, &c.state.accessCount, &credit, &c.bm25); err != nil {
				return err
			}
			c.state.updatedAt, c.state.accessedAt = fromMillis(updated), optionalTime(accessed)
			c.state.credit = time.Duration(credit) * time.Millisecond
			act := acts[sc]
			c.faded = s.retention.fadedAt(c.state, act, now)
			if c.faded && !q.IncludeFaded {
				continue
			}
			c.retention = s.retention.scoreAt(c.state, act, now)
			best = min(best, c.bm25)
			cands = append(cands, c)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
		for i := range cands {
			c := &cands[i]
			c.relevance = 1
			if len(phrases) > 0 && best < 0 {
				// bm25 is negative, lower for a better match: scale it to (0, 1].
				c.relevance = max(c.bm25/best, 1e-9)
			}
			c.score = c.relevance * c.retention
		}
		slices.SortFunc(cands, func(a, b searchCandidate) int {
			if a.score != b.score {
				return cmp.Compare(b.score, a.score)
			}
			if !a.state.updatedAt.Equal(b.state.updatedAt) {
				return b.state.updatedAt.Compare(a.state.updatedAt)
			}
			return cmp.Compare(b.id, a.id)
		})
		if len(cands) > limit {
			cands, more = cands[:limit], true
		}
		ids := make([]int64, len(cands))
		for i, c := range cands {
			ids[i] = c.id
		}
		loaded, err := loadMemories(ctx, tx, ids)
		if err != nil {
			return err
		}
		memories = make([]Memory, 0, len(cands))
		for _, c := range cands {
			m := loaded[c.id]
			s.retention.assessWith(&m, acts[m.Scope], now)
			memories = append(memories, m)
		}
		if len(terms) == 0 {
			return nil
		}
		for _, m := range memories[:min(len(memories), SearchCountedResults)] {
			if m.Faded {
				continue
			}
			if _, err := s.countReference(ctx, tx, m.ID, sess.ID, now, acts[m.Scope].discount, false); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return memories, more, nil
}

// scopeActivityIdle is how long a memory_scope_activity row with no memory is kept after its
// last access. The period is fixed and independent of the retention options, including a
// half-life of 0 and a grace of 0: the table must not grow when forgetting is disabled, and
// deleting a row of a scope without memories does not change any retention score.
const scopeActivityIdle = 60 * 24 * time.Hour

// deleteIdleScopeActivity deletes memory_scope_activity rows that have no memory in the same
// (scope, scope_id) and whose last access is older than scopeActivityIdle. The absence check
// and the delete are one statement. Pinned and faded memories are rows of memories, so they
// keep the activity row.
func deleteIdleScopeActivity(ctx context.Context, tx *sql.Tx, now time.Time) error {
	_, err := tx.ExecContext(ctx, `
		DELETE FROM memory_scope_activity
		WHERE last_access < ?
		  AND NOT EXISTS (
			SELECT 1 FROM memories m
			WHERE m.scope = memory_scope_activity.scope AND m.scope_id = memory_scope_activity.scope_id)`,
		millis(now.Add(-scopeActivityIdle)))
	return err
}

// deleteForgottenMemories deletes the memories that faded more than the grace period ago, and
// the references older than ReferenceWindow, which no longer keep anything from being counted.
// It returns the number of memories it deleted.
func (s *Store) deleteForgottenMemories(ctx context.Context, tx *sql.Tx, now time.Time) (int, error) {
	if _, err := tx.ExecContext(ctx, "DELETE FROM memory_refs WHERE counted_at <= ?", millis(now.Add(-ReferenceWindow))); err != nil {
		return 0, err
	}
	p := s.retention
	if !p.forgets() || p.Grace <= 0 {
		return 0, nil
	}
	// A memory never referenced fades soonest: HalfLife × log2(1/Threshold) after it was last
	// touched. A memory touched more recently than that plus the grace period cannot be due.
	least := p.fadeAfter(0) + p.Grace
	if least < 0 {
		return 0, nil // the sum overflows: no memory can be due before the year 2262
	}
	cutoff := millis(now.Add(-least))
	rows, err := tx.QueryContext(ctx, `
		SELECT m.id, m.updated_at, m.accessed_at, m.access_count, m.forgetting_credit,
			coalesce(a.last_access, 0), coalesce(a.discount, 0)
		FROM memories m
		LEFT JOIN memory_scope_activity a ON a.scope = m.scope AND a.scope_id = m.scope_id
		WHERE m.pinned = 0 AND max(m.updated_at, m.accessed_at) <= ?`, cutoff)
	if err != nil {
		return 0, err
	}
	var due []int64
	for rows.Next() {
		var id, updated, accessed, credit, lastMs, discMs int64
		var st retentionState
		if err := rows.Scan(&id, &updated, &accessed, &st.accessCount, &credit, &lastMs, &discMs); err != nil {
			rows.Close()
			return 0, err
		}
		st.updatedAt, st.accessedAt = fromMillis(updated), optionalTime(accessed)
		st.credit = time.Duration(credit) * time.Millisecond
		act := scopeActivity{lastAccess: optionalTime(lastMs), discount: time.Duration(discMs) * time.Millisecond}
		if p.expiredAt(st, act, now) {
			due = append(due, id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return len(due), deleteMemories(ctx, tx, due)
}

// ListMemories returns every memory, or those of one scope kind, in the order scope, scope
// identifier, ID, with the retention under the store's RetentionPolicy. It is for the
// administration commands, which see every user's memories.
func (s *Store) ListMemories(ctx context.Context, kind string) ([]Memory, error) {
	if kind != "" && !ValidScope(kind) {
		return nil, ErrInvalidScope
	}
	var out []Memory
	err := s.readTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, "SELECT m.id FROM memories m WHERE ? = '' OR m.scope = ? ORDER BY m.scope, m.scope_id, m.id", kind, kind)
		if err != nil {
			return err
		}
		var ids []int64
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		loaded := map[int64]Memory{}
		for chunk := range slices.Chunk(ids, 500) {
			ms, err := loadMemories(ctx, tx, chunk)
			if err != nil {
				return err
			}
			for id, m := range ms {
				loaded[id] = m
			}
		}
		acts := map[Scope]scopeActivity{}
		now := s.now()
		out = make([]Memory, 0, len(ids))
		for _, id := range ids {
			m := loaded[id]
			act, ok := acts[m.Scope]
			if !ok {
				act, err = loadScopeActivity(ctx, tx, m.Scope)
				if err != nil {
					return err
				}
				acts[m.Scope] = act
			}
			s.retention.assessWith(&m, act, now)
			out = append(out, m)
		}
		return nil
	})
	return out, err
}

// MemoryByID returns any memory by ID, for the administration commands. It does not count as
// a reference.
func (s *Store) MemoryByID(ctx context.Context, id int64) (Memory, error) {
	var m Memory
	var act scopeActivity
	err := s.readTx(ctx, func(tx *sql.Tx) error {
		var err error
		if m, err = loadMemory(ctx, tx, id); err != nil {
			return err
		}
		act, err = loadScopeActivity(ctx, tx, m.Scope)
		return err
	})
	if err != nil {
		return Memory{}, err
	}
	s.retention.assessWith(&m, act, s.now())
	return m, nil
}

// RemoveMemory deletes any memory by ID, for the administration commands, and returns it.
func (s *Store) RemoveMemory(ctx context.Context, id int64) (Memory, error) {
	var m Memory
	err := s.tx(ctx, func(tx *sql.Tx) error {
		var err error
		if m, err = loadMemory(ctx, tx, id); err != nil {
			return err
		}
		return deleteMemories(ctx, tx, []int64{id})
	})
	return m, err
}
