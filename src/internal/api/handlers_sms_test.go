package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	smsgateway "github.com/mattboston/sms-gateway"
	"github.com/mattboston/sms-gateway/internal/database"
	"github.com/mattboston/sms-gateway/internal/models"
	"github.com/mattboston/sms-gateway/internal/modem"
	_ "modernc.org/sqlite"
)

// newSMSTestHandler returns an SMSHandler backed by an in-memory database built
// from the real migrations, plus the repository so tests can seed messages.
func newSMSTestHandler(t *testing.T) (*SMSHandler, *database.Repository) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := database.RunMigrations(db, "sqlite", smsgateway.MigrationsFS); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	repo := database.NewRepository(db)
	return NewSMSHandler(repo, modem.NewMockModem()), repo
}

func seedMessages(t *testing.T, repo *database.Repository, direction models.Direction, status models.MessageStatus, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if _, err := repo.CreateMessage(direction, "+1555000000", fmt.Sprintf("body %d", i), status, nil); err != nil {
			t.Fatalf("seeding message %d: %v", i, err)
		}
	}
}

func decodeMessages(t *testing.T, w *httptest.ResponseRecorder) []models.Message {
	t.Helper()
	var got []models.Message
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return got
}

// TestHandleGetInbox_DefaultsToUnpaginated pins the backward-compatibility
// contract: a request with no limit behaves exactly as it did before pagination
// existed, so existing API-key clients keep working untouched.
func TestHandleGetInbox_DefaultsToUnpaginated(t *testing.T) {
	handler, repo := newSMSTestHandler(t)
	seedMessages(t, repo, models.DirectionInbound, models.StatusReceived, 12)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sms/inbox?all=true", nil)
	w := httptest.NewRecorder()
	handler.HandleGetInbox(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := len(decodeMessages(t, w)); got != 12 {
		t.Errorf("returned %d messages, want all 12", got)
	}
	if got := w.Header().Get("X-Total-Count"); got != "12" {
		t.Errorf("X-Total-Count = %q, want %q", got, "12")
	}
}

// TestHandleGetInbox_ResponseIsBareArray guards the shape itself. Switching to
// an envelope would silently break every existing consumer, so the body must
// stay a JSON array.
func TestHandleGetInbox_ResponseIsBareArray(t *testing.T) {
	handler, repo := newSMSTestHandler(t)
	seedMessages(t, repo, models.DirectionInbound, models.StatusReceived, 1)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sms/inbox?all=true", nil)
	w := httptest.NewRecorder()
	handler.HandleGetInbox(w, req)

	if first := w.Body.Bytes()[0]; first != '[' {
		t.Errorf("response begins with %q, want a JSON array starting with '['", first)
	}
}

// TestHandleGetInbox_EmptyIsArrayNotNull ensures an empty mailbox serializes as
// [] rather than null, which would break clients that iterate the result.
func TestHandleGetInbox_EmptyIsArrayNotNull(t *testing.T) {
	handler, _ := newSMSTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sms/inbox?all=true", nil)
	w := httptest.NewRecorder()
	handler.HandleGetInbox(w, req)

	if body := w.Body.String(); body != "[]\n" {
		t.Errorf("body = %q, want %q", body, "[]\n")
	}
	if got := w.Header().Get("X-Total-Count"); got != "0" {
		t.Errorf("X-Total-Count = %q, want %q", got, "0")
	}
}

func TestHandleGetInbox_Pagination(t *testing.T) {
	handler, repo := newSMSTestHandler(t)
	seedMessages(t, repo, models.DirectionInbound, models.StatusReceived, 10)

	tests := []struct {
		name      string
		query     string
		wantLen   int
		wantTotal string
	}{
		{"first page", "?all=true&limit=4", 4, "10"},
		{"middle page", "?all=true&limit=4&offset=4", 4, "10"},
		{"partial last page", "?all=true&limit=4&offset=8", 2, "10"},
		{"offset past end", "?all=true&limit=4&offset=99", 0, "10"},
		{"limit above cap is clamped", "?all=true&limit=99999", 10, "10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/sms/inbox"+tt.query, nil)
			w := httptest.NewRecorder()
			handler.HandleGetInbox(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
			}
			if got := len(decodeMessages(t, w)); got != tt.wantLen {
				t.Errorf("returned %d messages, want %d", got, tt.wantLen)
			}
			// The total must describe the whole result set, never the page.
			if got := w.Header().Get("X-Total-Count"); got != tt.wantTotal {
				t.Errorf("X-Total-Count = %q, want %q", got, tt.wantTotal)
			}
		})
	}
}

// TestHandleGetInbox_TotalIgnoresPageButRespectsFilter is the subtle one: the
// count must match the filter actually applied, otherwise the UI computes the
// wrong number of pages. The inbox defaults to unread-only, so the total must
// count unread messages, not every inbound message.
func TestHandleGetInbox_TotalIgnoresPageButRespectsFilter(t *testing.T) {
	handler, repo := newSMSTestHandler(t)
	seedMessages(t, repo, models.DirectionInbound, models.StatusReceived, 3)
	seedMessages(t, repo, models.DirectionInbound, models.StatusRead, 7)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sms/inbox?limit=2", nil)
	w := httptest.NewRecorder()
	handler.HandleGetInbox(w, req)

	if got := len(decodeMessages(t, w)); got != 2 {
		t.Errorf("returned %d messages, want 2", got)
	}
	if got := w.Header().Get("X-Total-Count"); got != "3" {
		t.Errorf("X-Total-Count = %q, want %q (unread only, not all 10 inbound)", got, "3")
	}
}

func TestHandleGetInbox_InvalidPaginationParams(t *testing.T) {
	handler, repo := newSMSTestHandler(t)
	seedMessages(t, repo, models.DirectionInbound, models.StatusReceived, 3)

	for _, query := range []string{
		"?all=true&limit=abc",
		"?all=true&limit=-1",
		"?all=true&offset=-5",
		"?all=true&offset=xyz",
	} {
		t.Run(query, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/sms/inbox"+query, nil)
			w := httptest.NewRecorder()
			handler.HandleGetInbox(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleGetOutbox_Pagination(t *testing.T) {
	handler, repo := newSMSTestHandler(t)
	seedMessages(t, repo, models.DirectionOutbound, models.StatusSent, 9)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sms/outbox?limit=4&offset=8", nil)
	w := httptest.NewRecorder()
	handler.HandleGetOutbox(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := len(decodeMessages(t, w)); got != 1 {
		t.Errorf("returned %d messages, want 1", got)
	}
	if got := w.Header().Get("X-Total-Count"); got != "9" {
		t.Errorf("X-Total-Count = %q, want %q", got, "9")
	}
}

// TestHandleGetOutbox_PagesDoNotOverlap walks the outbox page by page through
// the HTTP layer and requires each message to appear exactly once.
func TestHandleGetOutbox_PagesDoNotOverlap(t *testing.T) {
	handler, repo := newSMSTestHandler(t)
	const total, pageSize = 17, 5
	seedMessages(t, repo, models.DirectionOutbound, models.StatusSent, total)

	seen := map[string]int{}
	for offset := 0; offset < total; offset += pageSize {
		url := fmt.Sprintf("/api/v1/sms/outbox?limit=%d&offset=%d", pageSize, offset)
		req := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		handler.HandleGetOutbox(w, req)

		for _, m := range decodeMessages(t, w) {
			seen[m.ID]++
		}
	}

	if len(seen) != total {
		t.Errorf("paging saw %d distinct messages, want %d", len(seen), total)
	}
	for id, count := range seen {
		if count != 1 {
			t.Errorf("message %s appeared %d times across pages, want 1", id, count)
		}
	}
}

func TestHandleMessageStats(t *testing.T) {
	handler, repo := newSMSTestHandler(t)
	seedMessages(t, repo, models.DirectionInbound, models.StatusReceived, 3)
	seedMessages(t, repo, models.DirectionInbound, models.StatusRead, 2)
	seedMessages(t, repo, models.DirectionOutbound, models.StatusSent, 4)
	seedMessages(t, repo, models.DirectionOutbound, models.StatusPending, 1)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sms/stats", nil)
	w := httptest.NewRecorder()
	handler.HandleMessageStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var stats models.MessageStats
	if err := json.NewDecoder(w.Body).Decode(&stats); err != nil {
		t.Fatalf("decoding stats: %v", err)
	}

	checks := []struct {
		name string
		got  int
		want int
	}{
		{"Total", stats.Total, 10},
		{"Inbound", stats.Inbound, 5},
		{"Outbound", stats.Outbound, 5},
		{"Unread", stats.Unread, 3},
		{"Sent", stats.Sent, 4},
		{"Pending", stats.Pending, 1},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("stats.%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}
