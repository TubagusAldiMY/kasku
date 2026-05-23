package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TubagusAldiMY/kasku/notification-service/internal/infrastructure/persistence"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// fakeRepo adalah in-memory PreferenceRepository untuk handler test.
type fakeRepo struct {
	store     map[string]persistence.NotificationPreference
	getErr    error
	upsertErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{store: make(map[string]persistence.NotificationPreference)}
}

func (r *fakeRepo) Get(_ context.Context, userID string) (*persistence.NotificationPreference, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	p, ok := r.store[userID]
	if !ok {
		return nil, nil
	}
	return &p, nil
}

func (r *fakeRepo) Upsert(_ context.Context, userID string, pref persistence.NotificationPreference) (*persistence.NotificationPreference, error) {
	if r.upsertErr != nil {
		return nil, r.upsertErr
	}
	r.store[userID] = pref
	return &pref, nil
}

func setupRouter(repo persistence.PreferenceRepository) *gin.Engine {
	r := gin.New()
	h := NewPreferenceHandler(repo)
	r.GET("/notifications/preferences", h.Get)
	r.PUT("/notifications/preferences", h.Update)
	return r
}

func TestPreferenceGet_RejectsMissingUserHeader(t *testing.T) {
	t.Parallel()
	r := setupRouter(newFakeRepo())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/preferences", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestPreferenceGet_ReturnsDefaultsWhenNoRowExists(t *testing.T) {
	t.Parallel()
	r := setupRouter(newFakeRepo())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/preferences", nil)
	req.Header.Set("X-User-ID", "u1")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Success bool                               `json:"success"`
		Data    persistence.NotificationPreference `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if !body.Data.EmailEnabled || !body.Data.PaymentAlertsEnabled || !body.Data.ExpiryAlertsEnabled {
		t.Fatalf("default should enable all toggles, got %+v", body.Data)
	}
}

func TestPreferenceGet_ReturnsStoredRow(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	repo.store["u1"] = persistence.NotificationPreference{
		EmailEnabled: false, PaymentAlertsEnabled: true, ExpiryAlertsEnabled: false,
	}
	r := setupRouter(repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/preferences", nil)
	req.Header.Set("X-User-ID", "u1")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body struct {
		Data persistence.NotificationPreference `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Data.EmailEnabled {
		t.Fatalf("expected stored EmailEnabled=false, got true")
	}
}

func TestPreferenceGet_RepoErrorReturns500(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	repo.getErr = errors.New("db down")
	r := setupRouter(repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/preferences", nil)
	req.Header.Set("X-User-ID", "u1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestPreferenceUpdate_PersistsValueAndReturnsIt(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	r := setupRouter(repo)

	payload := persistence.NotificationPreference{
		EmailEnabled: true, PaymentAlertsEnabled: false, ExpiryAlertsEnabled: true,
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("X-User-ID", "u2")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	stored, ok := repo.store["u2"]
	if !ok {
		t.Fatal("repo missing upserted row")
	}
	if stored != payload {
		t.Fatalf("stored mismatch: got %+v", stored)
	}
}

func TestPreferenceUpdate_RejectsInvalidJSON(t *testing.T) {
	t.Parallel()
	r := setupRouter(newFakeRepo())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/notifications/preferences", bytes.NewReader([]byte("{not-json")))
	req.Header.Set("X-User-ID", "u1")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
