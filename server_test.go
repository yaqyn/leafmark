package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bootdotdev/learn-cicd-starter/internal/database"
)

func testApp(t *testing.T) (*apiConfig, http.Handler) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err = migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	cfg := &apiConfig{DB: database.New(db), conn: db, allowSignup: true, limiter: newRateLimiter()}
	return cfg, cfg.routes()
}
func request(h http.Handler, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func signup(t *testing.T, h http.Handler, name string) *http.Cookie {
	t.Helper()
	w := request(h, "POST", "/v1/auth/register", `{"name":"Test","username":"`+name+`","password":"correct horse battery staple"}`, nil)
	if w.Code != 201 {
		t.Fatalf("signup: %d %s", w.Code, w.Body)
	}
	return w.Result().Cookies()[0]
}
func TestOwnershipConflictAndTrash(t *testing.T) {
	_, h := testApp(t)
	alice := signup(t, h, "alice")
	bob := signup(t, h, "bob")
	w := request(h, "POST", "/v1/notes", `{"title":"Private","note":"original"}`, alice)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body)
	}
	var n leafNote
	if err := json.Unmarshal(w.Body.Bytes(), &n); err != nil {
		t.Fatal(err)
	}
	w = request(h, "PUT", "/v1/notes/"+n.ID, `{"title":"Stolen","version":1}`, bob)
	if w.Code != 404 {
		t.Fatal("cross-owner write", w.Code)
	}
	w = request(h, "PUT", "/v1/notes/"+n.ID, `{"title":"Saved","note":"edited","version":1}`, alice)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w = request(h, "PUT", "/v1/notes/"+n.ID, `{"title":"Stale","version":1}`, alice)
	if w.Code != 409 {
		t.Fatal("stale write", w.Code)
	}
	w = request(h, "PUT", "/v1/notes/"+n.ID, `{"title":"Saved","note":"edited","trashed":true,"version":2}`, alice)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w = request(h, "GET", "/v1/notes", ``, alice)
	if !strings.Contains(w.Body.String(), `"total":0`) {
		t.Fatal(w.Body)
	}
	w = request(h, "GET", "/v1/notes?view=trash", ``, alice)
	if !strings.Contains(w.Body.String(), `"total":1`) {
		t.Fatal(w.Body)
	}
	w = request(h, "PUT", "/v1/notes/"+n.ID, `{"title":"Saved","note":"edited","trashed":false,"version":3}`, alice)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w = request(h, "GET", "/v1/notes?q=edited", ``, alice)
	if !strings.Contains(w.Body.String(), `"total":1`) {
		t.Fatal(w.Body)
	}
	w = request(h, "GET", "/v1/export", ``, bob)
	if strings.Contains(w.Body.String(), "edited") {
		t.Fatal("export leaked another user's note")
	}
}
func TestSessionsValidationAndCSRF(t *testing.T) {
	cfg, h := testApp(t)
	cookie := signup(t, h, "tester")
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("unsafe session cookie")
	}
	for _, body := range []string{`{`, `{"title":""}`, `{"title":"ok","unknown":1}`, `{"title":"ok"} {}`} {
		if w := request(h, "POST", "/v1/notes", body, cookie); w.Code != 400 {
			t.Fatal(body, w.Code)
		}
	}
	r := httptest.NewRequest("POST", "/v1/notes", strings.NewReader(`{"title":"attack"}`))
	r.Header.Set("Origin", "https://attacker.example")
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("CSRF accepted", w.Code)
	}
	w = request(h, "POST", "/v1/auth/login", `{"username":"tester","password":"wrong"}`, nil)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
	w = request(h, "POST", "/v1/auth/login", `{"username":"tester","password":"correct horse battery staple"}`, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w = request(h, "POST", "/v1/auth/logout", ``, cookie)
	if w.Code != 204 {
		t.Fatal(w.Code)
	}
	w = request(h, "GET", "/v1/users", ``, cookie)
	if w.Code != 401 {
		t.Fatal("revoked session accepted", w.Code)
	}
	cfg.allowSignup = false
	w = request(h, "POST", "/v1/auth/register", `{}`, nil)
	if w.Code != 403 {
		t.Fatal("registration switch ignored")
	}
}
func TestMigrationPreservesCourseData(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, name := range []string{"001_users.sql", "002_notes.sql"} {
		data, _ := schemaFiles.ReadFile("sql/schema/" + name)
		up := strings.Split(strings.TrimPrefix(string(data), "-- +goose Up"), "-- +goose Down")[0]
		if _, err = db.Exec(up); err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.Exec(`INSERT INTO users VALUES('old','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','Old user','legacy-secret'); INSERT INTO notes VALUES('old-note','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','Keep me','old')`)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err = migrate(context.Background(), db); err != nil {
			t.Fatal(err)
		}
	}
	cfg := &apiConfig{DB: database.New(db), conn: db, allowSignup: true, limiter: newRateLimiter()}
	h := cfg.routes()
	w := request(h, "POST", "/v1/auth/claim", `{"username":"old-user","password":"correct horse battery staple","legacy_key":"legacy-secret"}`, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w = request(h, "GET", "/v1/notes", ``, w.Result().Cookies()[0])
	if !strings.Contains(w.Body.String(), "Keep me") {
		t.Fatal("legacy data lost", w.Body)
	}
	w = request(h, "POST", "/v1/auth/claim", `{"username":"other-user","password":"correct horse battery staple","legacy_key":"legacy-secret"}`, nil)
	if w.Code != 401 {
		t.Fatal("legacy key still usable", w.Code)
	}
}
func TestAuthRateLimit(t *testing.T) {
	_, h := testApp(t)
	for range 20 {
		request(h, "POST", "/v1/auth/login", `{}`, nil)
	}
	w := request(h, "POST", "/v1/auth/login", `{}`, nil)
	if w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatal(w.Code)
	}
}
