package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	b := New("test-token", "123456")
	if b == nil {
		t.Fatal("New() returned nil")
	}
	if b.token != "test-token" {
		t.Errorf("token = %q, want %q", b.token, "test-token")
	}
	if b.chatID != "123456" {
		t.Errorf("chatID = %q, want %q", b.chatID, "123456")
	}
	if b.client == nil {
		t.Error("client is nil")
	}
	if b.client.Timeout != 30*time.Second {
		t.Errorf("client timeout = %v, want 30s", b.client.Timeout)
	}
	if b.OnApproval != nil {
		t.Error("OnApproval should be nil by default")
	}
}

func TestAPIURL(t *testing.T) {
	b := New("abc123", "chat1")
	url := b.apiURL("sendMessage")
	want := "https://api.telegram.org/botabc123/sendMessage"
	if url != want {
		t.Errorf("apiURL = %q, want %q", url, want)
	}

	url2 := b.apiURL("getUpdates")
	want2 := "https://api.telegram.org/botabc123/getUpdates"
	if url2 != want2 {
		t.Errorf("apiURL = %q, want %q", url2, want2)
	}
}

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello", "hello"},
		{"hello*world", "hello\\*world"},
		{"foo_bar", "foo\\_bar"},
		{"`code`", "\\`code\\`"},
		{"[link]", "\\[link]"},
		{"a*b_c`d[e", "a\\*b\\_c\\`d\\[e"},
	}
	for _, tc := range tests {
		got := escapeMarkdown(tc.input)
		if got != tc.want {
			t.Errorf("escapeMarkdown(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestCallbackParsing(t *testing.T) {
	tests := []struct {
		data    string
		action  string
		blockID string
		valid   bool
	}{
		{"approve:block-1", "approve", "block-1", true},
		{"reject:block-2", "reject", "block-2", true},
		{"invalid", "", "", false},
		{"approve:block:extra", "approve", "block:extra", true},
		{"unknown:block-3", "", "", false},
		{"", "", "", false},
		{"nocolon", "", "", false},
	}

	for _, tc := range tests {
		parts := strings.SplitN(tc.data, ":", 2)
		if len(parts) != 2 {
			if tc.valid {
				t.Errorf("data=%q: expected valid but SplitN returned %d parts", tc.data, len(parts))
			}
			continue
		}
		action, blockID := parts[0], parts[1]
		if action != "approve" && action != "reject" {
			if tc.valid {
				t.Errorf("data=%q: expected valid but action=%q not approve/reject", tc.data, action)
			}
			continue
		}
		if !tc.valid {
			t.Errorf("data=%q: expected invalid but parsed as action=%q blockID=%q", tc.data, action, blockID)
			continue
		}
		if action != tc.action || blockID != tc.blockID {
			t.Errorf("data=%q: got action=%q blockID=%q, want action=%q blockID=%q",
				tc.data, action, blockID, tc.action, tc.blockID)
		}
	}
}

func TestOnApprovalCallback(t *testing.T) {
	b := New("token", "chat1")

	var calledBlockID, calledDecision string
	b.OnApproval = func(blockID, decision string) {
		calledBlockID = blockID
		calledDecision = decision
	}

	b.OnApproval("block-42", "approve")
	if calledBlockID != "block-42" {
		t.Errorf("blockID = %q, want block-42", calledBlockID)
	}
	if calledDecision != "approve" {
		t.Errorf("decision = %q, want approve", calledDecision)
	}

	b.OnApproval("block-99", "reject")
	if calledBlockID != "block-99" {
		t.Errorf("blockID = %q, want block-99", calledBlockID)
	}
	if calledDecision != "reject" {
		t.Errorf("decision = %q, want reject", calledDecision)
	}
}

func TestStartPollingContextCancellation(t *testing.T) {
	b := New("token", "chat1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan bool, 1)
	go func() {
		b.StartPolling(ctx)
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StartPolling did not exit on cancelled context")
	}
}

// transportFunc is an http.RoundTripper implemented as a function.
type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// newBotWithServer creates a Bot whose HTTP client is redirected to the test server.
// The server URL is passed so the bot's apiURL can be mocked.
func newBotWithServer(srv *httptest.Server, token, chatID string) *Bot {
	b := New(token, chatID)
	b.client = srv.Client()
	// Override apiURL behavior by replacing the client's transport
	// to rewrite Telegram URLs to the test server URL.
	baseTransport := srv.Client().Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	b.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		// Rewrite the URL from Telegram API to the test server
		r.URL.Scheme = "http"
		r.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return baseTransport.RoundTrip(r)
	})
	return b
}

func TestSendMessageHTTPSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["chat_id"] != "chat123" {
			t.Errorf("chat_id = %v, want chat123", payload["chat_id"])
		}
		if payload["parse_mode"] != "Markdown" {
			t.Errorf("parse_mode = %v", payload["parse_mode"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	b := newBotWithServer(srv, "test-token", "chat123")
	err := b.SendMessage("Hello, World!")
	if err != nil {
		t.Fatalf("SendMessage() error: %v", err)
	}
}

func TestSendMessageAcceptAnyStatus(t *testing.T) {
	// SendMessage does not inspect the HTTP status code; it only fails on
	// transport-level errors. This test confirms that behavior.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	b := newBotWithServer(srv, "test-token", "chat123")
	err := b.SendMessage("test")
	if err != nil {
		t.Fatalf("SendMessage() returned error on transport success: %v", err)
	}
}

func TestSendWorkBlockApprovalActivate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)

		rm, _ := payload["reply_markup"].(map[string]any)
		kb, _ := rm["inline_keyboard"].([]any)
		row, _ := kb[0].([]any)
		btn1, _ := row[0].(map[string]any)
		if btn1["text"] != "Activate" {
			t.Errorf("button text = %q, want Activate", btn1["text"])
		}
		if btn1["callback_data"] != "approve:block-1" {
			t.Errorf("callback_data = %q", btn1["callback_data"])
		}
		btn2, _ := row[1].(map[string]any)
		if btn2["text"] != "Reject" {
			t.Errorf("reject text = %q", btn2["text"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	b := newBotWithServer(srv, "test-token", "chat1")
	err := b.SendWorkBlockApproval("block-1", "Test Title", "Test Goal", "activate")
	if err != nil {
		t.Fatalf("SendWorkBlockApproval() error: %v", err)
	}
}

func TestSendWorkBlockApprovalShip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		rm, _ := payload["reply_markup"].(map[string]any)
		kb, _ := rm["inline_keyboard"].([]any)
		row, _ := kb[0].([]any)
		btn1, _ := row[0].(map[string]any)
		if btn1["text"] != "Ship" {
			t.Errorf("button text = %q, want Ship", btn1["text"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	b := newBotWithServer(srv, "test-token", "chat1")
	err := b.SendWorkBlockApproval("block-2", "Test Title", "Test Goal", "ready_to_ship")
	if err != nil {
		t.Fatalf("SendWorkBlockApproval(ship) error: %v", err)
	}
}

func TestSendWorkBlockApprovalHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	b := newBotWithServer(srv, "test-token", "chat1")
	err := b.SendWorkBlockApproval("block-1", "Title", "Goal", "activate")
	if err == nil {
		t.Fatal("expected error from SendWorkBlockApproval with 400 status")
	}
}

func TestGetUpdates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"result": []map[string]any{
				{
					"update_id": 100,
					"callback_query": map[string]string{
						"id":   "cb-1",
						"data": "approve:block-42",
					},
				},
				{
					"update_id": 200,
					"callback_query": map[string]string{
						"id":   "cb-2",
						"data": "reject:block-99",
					},
				},
			},
		})
	}))
	defer srv.Close()

	b := newBotWithServer(srv, "test-token", "chat1")
	updates, newOffset, err := b.getUpdates(0)
	if err != nil {
		t.Fatalf("getUpdates() error: %v", err)
	}
	if len(updates) != 2 {
		t.Fatalf("got %d updates, want 2", len(updates))
	}
	if updates[0].CallbackQuery.Data != "approve:block-42" {
		t.Errorf("update[0] data = %q", updates[0].CallbackQuery.Data)
	}
	if updates[1].CallbackQuery.Data != "reject:block-99" {
		t.Errorf("update[1] data = %q", updates[1].CallbackQuery.Data)
	}
	if newOffset != 201 {
		t.Errorf("newOffset = %d, want 201 (max 200 + 1)", newOffset)
	}
}

func TestGetUpdatesOffset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify offset query parameter
		if r.URL.Query().Get("offset") != "42" {
			t.Errorf("offset = %q, want 42", r.URL.Query().Get("offset"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": []any{}})
	}))
	defer srv.Close()

	b := newBotWithServer(srv, "test-token", "chat1")
	_, newOffset, err := b.getUpdates(42)
	if err != nil {
		t.Fatalf("getUpdates() error: %v", err)
	}
	if newOffset != 42 {
		t.Errorf("newOffset = %d, want 42 (no updates to advance)", newOffset)
	}
}

func TestGetUpdatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	b := newBotWithServer(srv, "test-token", "chat1")
	_, _, err := b.getUpdates(0)
	if err == nil {
		t.Fatal("expected error from getUpdates with 500 status")
	}
}

func TestAnswerCallbackApprove(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/answerCallbackQuery") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["callback_query_id"] != "cb-1" {
			t.Errorf("callback_query_id = %v", payload["callback_query_id"])
		}
		if payload["text"] != "Approved" {
			t.Errorf("text = %v, want Approved", payload["text"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	b := newBotWithServer(srv, "test-token", "chat1")
	b.answerCallback("cb-1", "approve")
}

func TestAnswerCallbackReject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["text"] != "Rejected" {
			t.Errorf("text = %v, want Rejected", payload["text"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	b := newBotWithServer(srv, "test-token", "chat1")
	b.answerCallback("cb-2", "reject")
}

func TestStartPollingProcessCallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "getUpdates") {
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"result": []map[string]any{
					{
						"update_id": 1,
						"callback_query": map[string]string{
							"id":   "cb-1",
							"data": "approve:block-approved",
						},
					},
				},
			})
		} else {
			json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
	}))
	defer srv.Close()

	b := newBotWithServer(srv, "token", "chat1")

	var gotBlockID, gotDecision string
	gotCall := make(chan struct{}, 1)
	b.OnApproval = func(blockID, decision string) {
		gotBlockID = blockID
		gotDecision = decision
		gotCall <- struct{}{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start polling; it will process one update and then wait 30s for the next
	// (timeout=30). We cancel after receiving the callback.
	go b.StartPolling(ctx)

	select {
	case <-gotCall:
		if gotBlockID != "block-approved" {
			t.Errorf("blockID = %q, want block-approved", gotBlockID)
		}
		if gotDecision != "approve" {
			t.Errorf("decision = %q, want approve", gotDecision)
		}
		cancel()
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for callback processing")
	}
}
