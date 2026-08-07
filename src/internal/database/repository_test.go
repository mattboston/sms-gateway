package database

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	smsgateway "github.com/mattboston/sms-gateway"
	"github.com/mattboston/sms-gateway/internal/models"
	_ "modernc.org/sqlite"
)

// setupTestDB builds a test database by running the real migrations rather than
// a hand-copied schema. Tests then exercise the same tables *and indexes* that
// production runs, so query plans — and therefore row ordering — match.
func setupTestDB(t *testing.T) *Repository {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := RunMigrations(db, "sqlite", smsgateway.MigrationsFS); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	return NewRepository(db)
}

func TestCreateUser(t *testing.T) {
	repo := setupTestDB(t)

	user, err := repo.CreateUser("testuser", "hash123", false, false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("user.Username = %q, want %q", user.Username, "testuser")
	}
	if user.IsAdmin {
		t.Error("user.IsAdmin should be false")
	}
	if user.ID == "" {
		t.Error("user.ID should not be empty")
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	repo := setupTestDB(t)

	_, err := repo.CreateUser("dup", "hash1", false, false)
	if err != nil {
		t.Fatalf("first CreateUser() error = %v", err)
	}

	_, err = repo.CreateUser("dup", "hash2", false, false)
	if err == nil {
		t.Error("second CreateUser() should fail for duplicate username")
	}
}

func TestGetUserByUsername(t *testing.T) {
	repo := setupTestDB(t)

	_, err := repo.CreateUser("findme", "hash", true, false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	user, err := repo.GetUserByUsername("findme")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if user.Username != "findme" {
		t.Errorf("user.Username = %q, want %q", user.Username, "findme")
	}
	if !user.IsAdmin {
		t.Error("user.IsAdmin should be true")
	}
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	repo := setupTestDB(t)

	_, err := repo.GetUserByUsername("nonexistent")
	if err == nil {
		t.Error("GetUserByUsername() should return error for missing user")
	}
}

func TestSeedDefaultAdmin(t *testing.T) {
	repo := setupTestDB(t)

	seeded, err := repo.SeedDefaultAdmin("adminhash")
	if err != nil {
		t.Fatalf("SeedDefaultAdmin() error = %v", err)
	}
	if !seeded {
		t.Error("SeedDefaultAdmin() should return true on first call")
	}

	seeded, err = repo.SeedDefaultAdmin("adminhash")
	if err != nil {
		t.Fatalf("SeedDefaultAdmin() second call error = %v", err)
	}
	if seeded {
		t.Error("SeedDefaultAdmin() should return false when users exist")
	}
}

func TestUpdatePassword(t *testing.T) {
	repo := setupTestDB(t)

	user, _ := repo.CreateUser("pwuser", "oldhash", false, true)
	if !user.MustChangePassword {
		t.Fatal("user.MustChangePassword should be true initially")
	}

	err := repo.UpdatePassword(user.ID, "newhash")
	if err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}

	updated, _ := repo.GetUserByID(user.ID)
	if updated.MustChangePassword {
		t.Error("MustChangePassword should be false after UpdatePassword")
	}
}

func TestListUsers(t *testing.T) {
	repo := setupTestDB(t)

	_, _ = repo.CreateUser("user1", "hash1", false, false)
	_, _ = repo.CreateUser("user2", "hash2", true, false)

	users, err := repo.ListUsers(ListOptions{})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 2 {
		t.Errorf("ListUsers() returned %d users, want 2", len(users))
	}
}

func TestCreateAPIKey(t *testing.T) {
	repo := setupTestDB(t)

	user, _ := repo.CreateUser("keyuser", "hash", false, false)
	key, err := repo.CreateAPIKey("testapikey123", "test label", user.ID)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}
	if key.Key != "testapikey123" {
		t.Errorf("key.Key = %q, want %q", key.Key, "testapikey123")
	}
	if key.Label != "test label" {
		t.Errorf("key.Label = %q, want %q", key.Label, "test label")
	}
	if !key.IsActive {
		t.Error("key.IsActive should be true")
	}
}

func TestGetAPIKeyByKey(t *testing.T) {
	repo := setupTestDB(t)

	user, _ := repo.CreateUser("keyuser2", "hash", false, false)
	_, _ = repo.CreateAPIKey("findthiskey", "label", user.ID)

	found, err := repo.GetAPIKeyByKey("findthiskey")
	if err != nil {
		t.Fatalf("GetAPIKeyByKey() error = %v", err)
	}
	if found.Key != "findthiskey" {
		t.Errorf("found.Key = %q, want %q", found.Key, "findthiskey")
	}
}

func TestDeactivateAPIKey(t *testing.T) {
	repo := setupTestDB(t)

	user, _ := repo.CreateUser("keyuser3", "hash", false, false)
	key, _ := repo.CreateAPIKey("deactivateme", "label", user.ID)

	err := repo.DeactivateAPIKey(key.ID)
	if err != nil {
		t.Fatalf("DeactivateAPIKey() error = %v", err)
	}

	// Should no longer be findable by key (only active keys returned).
	_, err = repo.GetAPIKeyByKey("deactivateme")
	if err == nil {
		t.Error("GetAPIKeyByKey() should fail for deactivated key")
	}
}

func TestListAPIKeys(t *testing.T) {
	repo := setupTestDB(t)

	user, _ := repo.CreateUser("keyuser4", "hash", false, false)
	_, _ = repo.CreateAPIKey("key1", "label1", user.ID)
	_, _ = repo.CreateAPIKey("key2", "label2", user.ID)

	keys, err := repo.ListAPIKeys(ListOptions{})
	if err != nil {
		t.Fatalf("ListAPIKeys() error = %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("ListAPIKeys() returned %d keys, want 2", len(keys))
	}
}

func TestCreateMessage(t *testing.T) {
	repo := setupTestDB(t)

	msg, err := repo.CreateMessage(models.DirectionOutbound, "+15551234567", "Hello", models.StatusPending, nil)
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}
	if msg.PhoneNumber != "+15551234567" {
		t.Errorf("msg.PhoneNumber = %q, want %q", msg.PhoneNumber, "+15551234567")
	}
	if msg.Direction != models.DirectionOutbound {
		t.Errorf("msg.Direction = %q, want %q", msg.Direction, models.DirectionOutbound)
	}
	if msg.Status != models.StatusPending {
		t.Errorf("msg.Status = %q, want %q", msg.Status, models.StatusPending)
	}
}

func TestGetMessage(t *testing.T) {
	repo := setupTestDB(t)

	created, _ := repo.CreateMessage(models.DirectionInbound, "+15559876543", "Hi there", models.StatusReceived, nil)

	found, err := repo.GetMessage(created.ID)
	if err != nil {
		t.Fatalf("GetMessage() error = %v", err)
	}
	if found.Body != "Hi there" {
		t.Errorf("found.Body = %q, want %q", found.Body, "Hi there")
	}
}

func TestListMessages(t *testing.T) {
	repo := setupTestDB(t)

	_, _ = repo.CreateMessage(models.DirectionInbound, "+1111", "in1", models.StatusReceived, nil)
	_, _ = repo.CreateMessage(models.DirectionInbound, "+2222", "in2", models.StatusReceived, nil)
	_, _ = repo.CreateMessage(models.DirectionOutbound, "+3333", "out1", models.StatusSent, nil)

	inbound, err := repo.ListMessages(models.DirectionInbound, nil, ListOptions{})
	if err != nil {
		t.Fatalf("ListMessages(inbound) error = %v", err)
	}
	if len(inbound) != 2 {
		t.Errorf("inbound messages = %d, want 2", len(inbound))
	}

	outbound, err := repo.ListMessages(models.DirectionOutbound, nil, ListOptions{})
	if err != nil {
		t.Fatalf("ListMessages(outbound) error = %v", err)
	}
	if len(outbound) != 1 {
		t.Errorf("outbound messages = %d, want 1", len(outbound))
	}
}

func TestListMessages_FilterByStatus(t *testing.T) {
	repo := setupTestDB(t)

	_, _ = repo.CreateMessage(models.DirectionOutbound, "+1111", "msg1", models.StatusSent, nil)
	_, _ = repo.CreateMessage(models.DirectionOutbound, "+2222", "msg2", models.StatusFailed, nil)
	_, _ = repo.CreateMessage(models.DirectionOutbound, "+3333", "msg3", models.StatusSent, nil)

	sent := models.StatusSent
	filtered, err := repo.ListMessages(models.DirectionOutbound, &sent, ListOptions{})
	if err != nil {
		t.Fatalf("ListMessages(outbound, sent) error = %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("filtered messages = %d, want 2", len(filtered))
	}
}

// TestListMessages_PaginationCoversEveryRowExactlyOnce walks the whole table one
// page at a time and requires every id to come back exactly once.
//
// This is what makes offset paging trustworthy: all 25 rows here share the same
// created_at (it is stamped with second granularity), so any instability in how
// ties are ordered would surface as a row on two consecutive pages and another
// missing entirely. It also covers plain limit/offset arithmetic mistakes.
//
// Honest caveat: this test does not currently fail if the `, id DESC` tiebreaker
// is removed from ListMessages. On SQLite the query is answered by the covering
// index (direction, created_at DESC, id DESC), whose own third column already
// orders the ties. The explicit tiebreaker is kept so the guarantee comes from
// the query rather than from the incidental shape of an index that a future
// migration could change, and so it holds on PostgreSQL too.
func TestListMessages_PaginationCoversEveryRowExactlyOnce(t *testing.T) {
	repo := setupTestDB(t)

	const total = 25
	want := make(map[string]bool, total)
	for i := 0; i < total; i++ {
		m, err := repo.CreateMessage(models.DirectionInbound, "+1111", "body", models.StatusReceived, nil)
		if err != nil {
			t.Fatalf("CreateMessage(%d) error = %v", i, err)
		}
		want[m.ID] = true
	}

	const pageSize = 7
	seen := make(map[string]int)
	for offset := 0; offset < total; offset += pageSize {
		page, err := repo.ListMessages(models.DirectionInbound, nil, ListOptions{Limit: pageSize, Offset: offset})
		if err != nil {
			t.Fatalf("ListMessages(offset=%d) error = %v", offset, err)
		}

		wantLen := pageSize
		if remaining := total - offset; remaining < pageSize {
			wantLen = remaining
		}
		if len(page) != wantLen {
			t.Errorf("page at offset %d returned %d messages, want %d", offset, len(page), wantLen)
		}

		for _, m := range page {
			seen[m.ID]++
		}
	}

	if len(seen) != total {
		t.Errorf("paging saw %d distinct messages, want %d", len(seen), total)
	}
	for id, count := range seen {
		if count != 1 {
			t.Errorf("message %s appeared on %d pages, want exactly 1", id, count)
		}
		if !want[id] {
			t.Errorf("paging returned unknown message %s", id)
		}
	}
}

func TestListMessages_LimitAndOffset(t *testing.T) {
	repo := setupTestDB(t)

	for i := 0; i < 5; i++ {
		_, _ = repo.CreateMessage(models.DirectionOutbound, "+1111", "body", models.StatusSent, nil)
	}

	all, err := repo.ListMessages(models.DirectionOutbound, nil, ListOptions{})
	if err != nil {
		t.Fatalf("ListMessages(no opts) error = %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("zero ListOptions returned %d messages, want all 5", len(all))
	}

	tests := []struct {
		name string
		opts ListOptions
		want int
	}{
		{"limit under total", ListOptions{Limit: 2}, 2},
		{"limit over total", ListOptions{Limit: 50}, 5},
		{"offset into last page", ListOptions{Limit: 2, Offset: 4}, 1},
		{"offset past end", ListOptions{Limit: 2, Offset: 99}, 0},
		{"offset without limit is ignored", ListOptions{Offset: 3}, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.ListMessages(models.DirectionOutbound, nil, tt.opts)
			if err != nil {
				t.Fatalf("ListMessages(%+v) error = %v", tt.opts, err)
			}
			if len(got) != tt.want {
				t.Errorf("ListMessages(%+v) returned %d messages, want %d", tt.opts, len(got), tt.want)
			}
		})
	}

	// A page must be a prefix of the unpaginated listing, in the same order.
	page, err := repo.ListMessages(models.DirectionOutbound, nil, ListOptions{Limit: 3})
	if err != nil {
		t.Fatalf("ListMessages(limit=3) error = %v", err)
	}
	for i, m := range page {
		if m.ID != all[i].ID {
			t.Errorf("page[%d] = %s, want %s (page must match unpaginated order)", i, m.ID, all[i].ID)
		}
	}
}

func TestCountMessages(t *testing.T) {
	repo := setupTestDB(t)

	_, _ = repo.CreateMessage(models.DirectionInbound, "+1111", "in1", models.StatusReceived, nil)
	_, _ = repo.CreateMessage(models.DirectionInbound, "+2222", "in2", models.StatusRead, nil)
	_, _ = repo.CreateMessage(models.DirectionOutbound, "+3333", "out1", models.StatusSent, nil)

	got, err := repo.CountMessages(models.DirectionInbound, nil)
	if err != nil {
		t.Fatalf("CountMessages(inbound) error = %v", err)
	}
	if got != 2 {
		t.Errorf("CountMessages(inbound) = %d, want 2", got)
	}

	received := models.StatusReceived
	got, err = repo.CountMessages(models.DirectionInbound, &received)
	if err != nil {
		t.Fatalf("CountMessages(inbound, received) error = %v", err)
	}
	if got != 1 {
		t.Errorf("CountMessages(inbound, received) = %d, want 1", got)
	}

	// The count must ignore the pagination window entirely: it reports how many
	// rows match, not how many a page returned.
	page, _ := repo.ListMessages(models.DirectionInbound, nil, ListOptions{Limit: 1})
	total, _ := repo.CountMessages(models.DirectionInbound, nil)
	if len(page) != 1 || total != 2 {
		t.Errorf("page=%d total=%d, want page=1 total=2", len(page), total)
	}
}

func TestMessageStats(t *testing.T) {
	repo := setupTestDB(t)

	_, _ = repo.CreateMessage(models.DirectionInbound, "+1111", "unread", models.StatusReceived, nil)
	_, _ = repo.CreateMessage(models.DirectionInbound, "+2222", "read", models.StatusRead, nil)
	_, _ = repo.CreateMessage(models.DirectionOutbound, "+3333", "sent", models.StatusSent, nil)
	_, _ = repo.CreateMessage(models.DirectionOutbound, "+4444", "pending", models.StatusPending, nil)
	_, _ = repo.CreateMessage(models.DirectionOutbound, "+5555", "sending", models.StatusSending, nil)
	_, _ = repo.CreateMessage(models.DirectionOutbound, "+6666", "failed", models.StatusFailed, nil)

	stats, err := repo.MessageStats()
	if err != nil {
		t.Fatalf("MessageStats() error = %v", err)
	}

	checks := []struct {
		name string
		got  int
		want int
	}{
		{"Total", stats.Total, 6},
		{"Inbound", stats.Inbound, 2},
		{"Outbound", stats.Outbound, 4},
		{"Unread", stats.Unread, 1},
		{"Sent", stats.Sent, 1},
		{"Pending", stats.Pending, 2}, // pending + sending
		{"Failed", stats.Failed, 1},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("stats.%s = %d, want %d", c.name, c.got, c.want)
		}
	}

	if got := stats.ByStatus["outbound.sent"]; got != 1 {
		t.Errorf(`stats.ByStatus["outbound.sent"] = %d, want 1`, got)
	}
	if _, ok := stats.ByStatus["inbound.failed"]; ok {
		t.Error(`stats.ByStatus should omit combinations with no rows`)
	}
}

func TestMessageStats_EmptyTable(t *testing.T) {
	repo := setupTestDB(t)

	stats, err := repo.MessageStats()
	if err != nil {
		t.Fatalf("MessageStats() error = %v", err)
	}
	if stats.Total != 0 {
		t.Errorf("stats.Total = %d, want 0", stats.Total)
	}
	// ByStatus must serialize as {} rather than null so clients can index it.
	if stats.ByStatus == nil {
		t.Error("stats.ByStatus = nil, want an empty map")
	}
}

func TestUpdateMessageStatus(t *testing.T) {
	repo := setupTestDB(t)

	msg, _ := repo.CreateMessage(models.DirectionOutbound, "+1111", "test", models.StatusPending, nil)

	err := repo.UpdateMessageStatus(msg.ID, models.StatusSent, nil, nil)
	if err != nil {
		t.Fatalf("UpdateMessageStatus() error = %v", err)
	}

	updated, _ := repo.GetMessage(msg.ID)
	if updated.Status != models.StatusSent {
		t.Errorf("updated.Status = %q, want %q", updated.Status, models.StatusSent)
	}
}

func TestUpdateMessageStatus_WithError(t *testing.T) {
	repo := setupTestDB(t)

	msg, _ := repo.CreateMessage(models.DirectionOutbound, "+1111", "test", models.StatusPending, nil)

	errMsg := "modem timeout"
	err := repo.UpdateMessageStatus(msg.ID, models.StatusFailed, nil, &errMsg)
	if err != nil {
		t.Fatalf("UpdateMessageStatus() error = %v", err)
	}

	updated, _ := repo.GetMessage(msg.ID)
	if updated.Status != models.StatusFailed {
		t.Errorf("updated.Status = %q, want %q", updated.Status, models.StatusFailed)
	}
	if updated.ErrorMessage == nil || *updated.ErrorMessage != "modem timeout" {
		t.Errorf("updated.ErrorMessage = %v, want %q", updated.ErrorMessage, "modem timeout")
	}
}

func TestMarkMessageRead(t *testing.T) {
	repo := setupTestDB(t)

	msg, _ := repo.CreateMessage(models.DirectionInbound, "+1111", "hello", models.StatusReceived, nil)

	err := repo.MarkMessageRead(msg.ID)
	if err != nil {
		t.Fatalf("MarkMessageRead() error = %v", err)
	}

	updated, _ := repo.GetMessage(msg.ID)
	if updated.Status != models.StatusRead {
		t.Errorf("status = %q, want %q", updated.Status, models.StatusRead)
	}
}

func TestMarkMessageRead_AlreadyRead(t *testing.T) {
	repo := setupTestDB(t)

	msg, _ := repo.CreateMessage(models.DirectionInbound, "+1111", "hello", models.StatusReceived, nil)
	_ = repo.MarkMessageRead(msg.ID)

	err := repo.MarkMessageRead(msg.ID)
	if err == nil {
		t.Error("MarkMessageRead() should fail for already-read message")
	}
}

func TestMarkMessageRead_OutboundMessage(t *testing.T) {
	repo := setupTestDB(t)

	msg, _ := repo.CreateMessage(models.DirectionOutbound, "+1111", "hello", models.StatusSent, nil)

	err := repo.MarkMessageRead(msg.ID)
	if err == nil {
		t.Error("MarkMessageRead() should fail for outbound message")
	}
}

func TestPing(t *testing.T) {
	repo := setupTestDB(t)
	if err := repo.Ping(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}


func TestConsolidateConnectedInboundMessages(t *testing.T) {
	repo := setupTestDB(t)

	part1 := strings.Repeat("ا", 67)
	part2 := strings.Repeat("ب", 67)
	part3 := "پایان"

	baseTime := time.Date(2026, 8, 3, 8, 47, 0, 0, time.UTC)
	ids := make([]string, 0, 3)
	for i, body := range []string{part1, part2, part3} {
		msg, err := repo.CreateMessage(models.DirectionInbound, "9008", body, models.StatusReceived, nil)
		if err != nil {
			t.Fatalf("CreateMessage part %d: %v", i, err)
		}
		ts := baseTime.Add(time.Duration(i) * time.Second).Format(time.RFC3339)
		if _, err := repo.db.Exec(`UPDATE messages SET created_at = ?, updated_at = ? WHERE id = ?`, ts, ts, msg.ID); err != nil {
			t.Fatalf("touch timestamps: %v", err)
		}
		ids = append(ids, msg.ID)
	}

	_, err := repo.CreateMessage(models.DirectionInbound, "9008", "fd", models.StatusReceived, nil)
	if err != nil {
		t.Fatalf("CreateMessage short: %v", err)
	}

	deleted, err := repo.ConsolidateConnectedInboundMessages()
	if err != nil {
		t.Fatalf("ConsolidateConnectedInboundMessages: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("deleted = %d, want 2", deleted)
	}

	merged, err := repo.GetMessage(ids[0])
	if err != nil {
		t.Fatalf("GetMessage merged: %v", err)
	}
	want := part1 + part2 + part3
	if merged.Body != want {
		t.Fatalf("merged body = %q, want %q", merged.Body, want)
	}

	inbox, err := repo.ListMessages(models.DirectionInbound, nil, ListOptions{})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(inbox) != 2 {
		t.Fatalf("inbox len = %d, want 2 (merged + short)", len(inbox))
	}
}

func TestCreateOrMergeInboundMessage(t *testing.T) {
	repo := setupTestDB(t)

	part1 := strings.Repeat("ا", 67)
	msg1, err := repo.CreateOrMergeInboundMessage("9008", part1)
	if err != nil {
		t.Fatalf("part1: %v", err)
	}

	part2 := strings.Repeat("ب", 67)
	msg2, err := repo.CreateOrMergeInboundMessage("9008", part2)
	if err != nil {
		t.Fatalf("part2: %v", err)
	}
	if msg2.ID != msg1.ID {
		t.Fatalf("part2 id = %s, want %s", msg2.ID, msg1.ID)
	}

	part3 := "پایان"
	msg3, err := repo.CreateOrMergeInboundMessage("9008", part3)
	if err != nil {
		t.Fatalf("part3: %v", err)
	}
	if msg3.ID != msg1.ID {
		t.Fatalf("part3 should merge into same message")
	}

	// 68-unit OTP-sized bodies must not merge with each other.
	otp1 := strings.Repeat("ک", 68)
	otpMsg, err := repo.CreateOrMergeInboundMessage("500044", otp1)
	if err != nil {
		t.Fatalf("otp1: %v", err)
	}
	otp2 := strings.Repeat("د", 68)
	otpMsg2, err := repo.CreateOrMergeInboundMessage("500044", otp2)
	if err != nil {
		t.Fatalf("otp2: %v", err)
	}
	if otpMsg2.ID == otpMsg.ID {
		t.Fatal("OTP-sized messages should not merge")
	}

	short, err := repo.CreateOrMergeInboundMessage("9008", "fd")
	if err != nil {
		t.Fatalf("short: %v", err)
	}
	if short.ID == msg1.ID {
		t.Fatalf("short message should not merge after final short segment")
	}
}
