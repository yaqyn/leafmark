package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yaqyn/leafmark/internal/database"
	"golang.org/x/crypto/argon2"
)

const sessionCookie = "leafmark_session"
const sessionLifetime = 7 * 24 * time.Hour

type credentials struct {
	Name      string `json:"name"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	LegacyKey string `json:"legacy_key"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
		respondWithError(w, 415, "Send application/json", nil)
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		respondWithError(w, 400, "Invalid JSON or request too large", nil)
		return false
	}
	if err := d.Decode(new(any)); err != io.EOF {
		respondWithError(w, 400, "Send one JSON object", nil)
		return false
	}
	return true
}
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("secure random source unavailable")
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func passwordHash(password string) string {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		panic("secure random source unavailable")
	}
	key := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
	return base64.RawStdEncoding.EncodeToString(salt) + "." + base64.RawStdEncoding.EncodeToString(key)
}
func checkPassword(encoded, password string) bool {
	parts := strings.Split(encoded, ".")
	if len(parts) != 2 {
		return false
	}
	salt, e1 := base64.RawStdEncoding.DecodeString(parts[0])
	key, e2 := base64.RawStdEncoding.DecodeString(parts[1])
	if e1 != nil || e2 != nil || len(salt) != 16 || len(key) != 32 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
	return subtle.ConstantTimeCompare(actual, key) == 1
}
func validCredentials(w http.ResponseWriter, c *credentials) bool {
	c.Username = strings.ToLower(strings.TrimSpace(c.Username))
	c.Name = strings.TrimSpace(c.Name)
	if len(c.Username) < 3 || len(c.Username) > 40 || strings.IndexFunc(c.Username, func(r rune) bool { return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-') }) >= 0 {
		respondWithError(w, 400, "Username needs 3–40 letters, numbers, dashes or underscores", nil)
		return false
	}
	if len(c.Password) < 12 || len(c.Password) > 256 {
		respondWithError(w, 400, "Use a password between 12 and 256 bytes", nil)
		return false
	}
	return true
}
func (cfg *apiConfig) register(w http.ResponseWriter, r *http.Request) {
	if !cfg.allowSignup {
		respondWithError(w, 403, "Registration is closed on this server", nil)
		return
	}
	var c credentials
	if !decodeJSON(w, r, &c) || !validCredentials(w, &c) {
		return
	}
	if c.Name == "" || len(c.Name) > 80 {
		respondWithError(w, 400, "Enter a name of 1–80 bytes", nil)
		return
	}
	hash := passwordHash(c.Password)
	tx, err := cfg.conn.BeginTx(r.Context(), nil)
	if err != nil {
		respondWithError(w, 503, "Could not create account; try again", nil)
		return
	}
	defer tx.Rollback()
	id := uuid.NewString()
	now := timestamp()
	_, err = tx.ExecContext(r.Context(), `INSERT INTO users(id,created_at,updated_at,name,api_key) VALUES(?,?,?,?,?)`, id, now, now, c.Name, hashToken(randomToken()))
	if err == nil {
		_, err = tx.ExecContext(r.Context(), `INSERT INTO accounts(user_id,username,password_hash) VALUES(?,?,?)`, id, c.Username, hash)
	}
	if err != nil {
		respondWithError(w, 409, "Could not create account; try another username", nil)
		return
	}
	if err = tx.Commit(); err != nil {
		respondWithError(w, 503, "Could not finish registration", nil)
		return
	}
	cfg.startSession(w, r, id, 201)
}
func (cfg *apiConfig) login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if !decodeJSON(w, r, &c) {
		return
	}
	c.Username = strings.ToLower(strings.TrimSpace(c.Username))
	if len(c.Password) > 256 || len(c.Username) > 40 {
		respondWithError(w, 401, "Username or password is incorrect", nil)
		return
	}
	var id, hash string
	err := cfg.conn.QueryRowContext(r.Context(), `SELECT user_id,password_hash FROM accounts WHERE username=?`, c.Username).Scan(&id, &hash)
	// Equal-cost verification also for unknown usernames.
	if errors.Is(err, sql.ErrNoRows) {
		hash = "AAAAAAAAAAAAAAAAAAAAAA.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	} else if err != nil {
		respondWithError(w, 503, "Sign-in unavailable; try again", nil)
		return
	}
	valid := checkPassword(hash, c.Password)
	if !valid || err != nil {
		respondWithError(w, 401, "Username or password is incorrect", nil)
		return
	}
	cfg.startSession(w, r, id, 200)
}
func (cfg *apiConfig) claimLegacy(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if !decodeJSON(w, r, &c) || !validCredentials(w, &c) {
		return
	}
	user, err := cfg.DB.GetUser(r.Context(), c.LegacyKey)
	if err != nil {
		respondWithError(w, 401, "Notely key is invalid or already migrated", nil)
		return
	}
	hash := passwordHash(c.Password)
	tx, err := cfg.conn.BeginTx(r.Context(), nil)
	if err != nil {
		respondWithError(w, 503, "Migration unavailable", nil)
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(), `INSERT INTO accounts(user_id,username,password_hash) VALUES(?,?,?)`, user.ID, c.Username, hash)
	if err != nil {
		respondWithError(w, 409, "Account already migrated or username unavailable", nil)
		return
	}
	_, err = tx.ExecContext(r.Context(), `UPDATE users SET api_key=? WHERE id=?`, hashToken(randomToken()), user.ID)
	if err != nil || tx.Commit() != nil {
		respondWithError(w, 503, "Could not finish migration", nil)
		return
	}
	cfg.startSession(w, r, user.ID, 200)
}
func (cfg *apiConfig) startSession(w http.ResponseWriter, r *http.Request, id string, status int) {
	token := randomToken()
	expiry := time.Now().UTC().Add(sessionLifetime)
	_, err := cfg.conn.ExecContext(r.Context(), `DELETE FROM sessions WHERE expires_at < ?`, time.Now().UTC().Format(time.RFC3339))
	if err == nil {
		_, err = cfg.conn.ExecContext(r.Context(), `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, hashToken(token), id, expiry.Format(time.RFC3339))
	}
	if err != nil {
		respondWithError(w, 503, "Could not start session; sign in again", nil)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: cfg.secureCookies, SameSite: http.SameSiteStrictMode, MaxAge: int(sessionLifetime.Seconds()), Expires: expiry})
	respondWithJSON(w, status, map[string]string{"status": "signed_in"})
}
func (cfg *apiConfig) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		if _, err = cfg.conn.ExecContext(r.Context(), `DELETE FROM sessions WHERE token_hash=?`, hashToken(cookie.Value)); err != nil {
			respondWithError(w, 503, "Could not sign out; try again", nil)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", HttpOnly: true, Secure: cfg.secureCookies, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	w.WriteHeader(204)
}

type attemptWindow struct {
	start time.Time
	count int
}
type rateLimiter struct {
	mu       sync.Mutex
	attempts map[string]attemptWindow
	active   chan struct{}
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{attempts: make(map[string]attemptWindow), active: make(chan struct{}, 2)}
}
func (cfg *apiConfig) limitAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		now := time.Now()
		l := cfg.limiter
		l.mu.Lock()
		for k, v := range l.attempts {
			if now.Sub(v.start) > 10*time.Minute {
				delete(l.attempts, k)
			}
		}
		a, known := l.attempts[host]
		if !known && len(l.attempts) >= 10000 {
			l.mu.Unlock()
			w.Header().Set("Retry-After", "600")
			respondWithError(w, 429, "Too many attempts; try again later", nil)
			return
		}
		if a.start.IsZero() {
			a.start = now
		}
		a.count++
		allowed := a.count <= 20
		l.attempts[host] = a
		l.mu.Unlock()
		if !allowed {
			w.Header().Set("Retry-After", "600")
			respondWithError(w, 429, "Too many attempts; try again in 10 minutes", nil)
			return
		}
		select {
		case l.active <- struct{}{}:
			defer func() { <-l.active }()
			next(w, r)
		default:
			w.Header().Set("Retry-After", "2")
			respondWithError(w, 429, "Sign-in is busy; try again shortly", nil)
		}
	}
}

func (cfg *apiConfig) sessionUser(r *http.Request) (database.User, error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return database.User{}, err
	}
	var u database.User
	err = cfg.conn.QueryRowContext(r.Context(), `SELECT u.id,u.created_at,u.updated_at,u.name FROM users u JOIN sessions s ON s.user_id=u.id WHERE s.token_hash=? AND s.expires_at>?`, hashToken(cookie.Value), time.Now().UTC().Format(time.RFC3339)).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt, &u.Name)
	return u, err
}
