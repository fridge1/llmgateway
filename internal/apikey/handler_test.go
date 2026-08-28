package apikey

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/store"
)

// mockStore implements the subset of store.Store used by Handler.
type mockStore struct {
	store.Store
	keys      []store.APIKey
	createErr error
	listErr   error
	deleteErr error
}

func (m *mockStore) ListAPIKeysByUser(userID string) ([]store.APIKey, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []store.APIKey
	for _, k := range m.keys {
		if k.UserID == userID {
			result = append(result, k)
		}
	}
	return result, nil
}

func (m *mockStore) CreateAPIKey(userID, keyHash, keyPrefix, name string, planID *int) (*store.APIKey, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	k := store.APIKey{ID: "key-1", UserID: userID, KeyHash: keyHash, KeyPrefix: keyPrefix, Name: name, Status: "active", PlanID: planID}
	m.keys = append(m.keys, k)
	return &k, nil
}

func (m *mockStore) DeleteAPIKey(id, userID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func TestGenerateAPIKey(t *testing.T) {
	plainKey, hash, prefix, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}
	if !strings.HasPrefix(plainKey, "sk-") {
		t.Errorf("key should start with sk-, got %q", plainKey[:5])
	}
	if len(plainKey) != 3+64 { // "sk-" + 32 bytes hex
		t.Errorf("key length = %d, want %d", len(plainKey), 67)
	}
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}
	if _, err := hex.DecodeString(hash); err != nil {
		t.Errorf("hash is not valid hex: %v", err)
	}
	if prefix != plainKey[:10] {
		t.Errorf("prefix = %q, want %q", prefix, plainKey[:10])
	}
}

func TestHashAPIKey_Deterministic(t *testing.T) {
	key := "sk-abc123"
	h1 := HashAPIKey(key)
	h2 := HashAPIKey(key)
	if h1 != h2 {
		t.Errorf("HashAPIKey not deterministic: %q != %q", h1, h2)
	}
	if len(h1) != 64 {
		t.Errorf("hash length = %d, want 64", len(h1))
	}
}

func TestHandleList_Unauthorized(t *testing.T) {
	h := NewHandler(&mockStore{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/keys", nil)
	w := httptest.NewRecorder()
	h.HandleList(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleList_Success(t *testing.T) {
	ms := &mockStore{keys: []store.APIKey{
		{ID: "k1", UserID: "user-1", Name: "test"},
	}}
	h := NewHandler(ms, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/keys", nil)
	ctx := context.WithValue(req.Context(), admin.CtxUserIDKey, "user-1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.HandleList(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp map[string][]store.APIKey
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp["keys"]) != 1 {
		t.Errorf("got %d keys, want 1", len(resp["keys"]))
	}
}

func TestHandleCreate_Success(t *testing.T) {
	h := NewHandler(&mockStore{}, nil)
	body := strings.NewReader(`{"name":"my-key"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/keys", body)
	ctx := context.WithValue(req.Context(), admin.CtxUserIDKey, "user-1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.HandleCreate(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	var resp createKeyResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.HasPrefix(resp.Key, "sk-") {
		t.Errorf("response key should start with sk-, got %q", resp.Key[:5])
	}
	if resp.APIKey.Name != "my-key" {
		t.Errorf("name = %q, want %q", resp.APIKey.Name, "my-key")
	}
}

func TestHandleCreate_Unauthorized(t *testing.T) {
	h := NewHandler(&mockStore{}, nil)
	body := strings.NewReader(`{"name":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/keys", body)
	w := httptest.NewRecorder()
	h.HandleCreate(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleRevoke_NotFound(t *testing.T) {
	ms := &mockStore{deleteErr: errors.New("not found")}
	h := NewHandler(ms, nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/keys/nonexistent", nil)
	ctx := context.WithValue(req.Context(), admin.CtxUserIDKey, "user-1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.HandleRevoke(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
