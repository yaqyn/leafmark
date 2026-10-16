package main

import (
	"database/sql"
	"errors"
	"github.com/bootdotdev/learn-cicd-starter/internal/database"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

type leafNote struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Note      string `json:"note"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Pinned    bool   `json:"pinned"`
	Archived  bool   `json:"archived"`
	Trashed   bool   `json:"trashed"`
	Version   int    `json:"version"`
}

const noteColumns = "id,title,note,created_at,updated_at,pinned,archived,trashed,version"

func scanNote(row interface{ Scan(...any) error }) (n leafNote, err error) {
	err = row.Scan(&n.ID, &n.Title, &n.Note, &n.CreatedAt, &n.UpdatedAt, &n.Pinned, &n.Archived, &n.Trashed, &n.Version)
	return
}
func validateNote(w http.ResponseWriter, n *leafNote) bool {
	n.Title = strings.TrimSpace(n.Title)
	if !utf8.ValidString(n.Title) || !utf8.ValidString(n.Note) || utf8.RuneCountInString(n.Title) > 160 || len(n.Note) > 100000 {
		respondWithError(w, 400, "Use a title under 160 characters and a note under 100 KB", nil)
		return false
	}
	if n.Title == "" && strings.TrimSpace(n.Note) == "" {
		respondWithError(w, 400, "Write a title or note before saving", nil)
		return false
	}
	return true
}
func (cfg *apiConfig) handlerNotesGet(w http.ResponseWriter, r *http.Request, u database.User) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		respondWithError(w, 400, "Page is out of range", nil)
		return
	}
	q := r.URL.Query().Get("q")
	if len(q) > 200 {
		respondWithError(w, 400, "Search is too long", nil)
		return
	}
	view := r.URL.Query().Get("view")
	filter := "trashed=0 AND archived=0"
	switch view {
	case "archive":
		filter = "trashed=0 AND archived=1"
	case "trash":
		filter = "trashed=1"
	case "pinned":
		filter = "trashed=0 AND archived=0 AND pinned=1"
	case "", "all":
	default:
		respondWithError(w, 400, "Unknown notebook view", nil)
		return
	}
	where := ` FROM notes WHERE user_id=? AND ` + filter + ` AND (?='' OR instr(lower(title || char(10) || note),lower(?))>0)`
	var total int
	if err := cfg.conn.QueryRowContext(r.Context(), "SELECT count(*)"+where, u.ID, q, q).Scan(&total); err != nil {
		respondWithError(w, 503, "Could not load notes", nil)
		return
	}
	rows, err := cfg.conn.QueryContext(r.Context(), "SELECT "+noteColumns+where+" ORDER BY pinned DESC, updated_at DESC, id LIMIT 30 OFFSET ?", u.ID, q, q, (page-1)*30)
	if err != nil {
		respondWithError(w, 503, "Could not load notes", nil)
		return
	}
	defer rows.Close()
	notes := []leafNote{}
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			respondWithError(w, 503, "Could not read notes", nil)
			return
		}
		notes = append(notes, n)
	}
	if rows.Err() != nil {
		respondWithError(w, 503, "Could not read notes", nil)
		return
	}
	respondWithJSON(w, 200, map[string]any{"notes": notes, "total": total, "page": page, "page_size": 30})
}
func (cfg *apiConfig) handlerNotesCreate(w http.ResponseWriter, r *http.Request, u database.User) {
	var n leafNote
	if !decodeJSON(w, r, &n) || !validateNote(w, &n) {
		return
	}
	n.ID = uuid.NewString()
	n.CreatedAt = timestamp()
	n.UpdatedAt = n.CreatedAt
	n.Version = 1
	_, err := cfg.conn.ExecContext(r.Context(), `INSERT INTO notes(id,title,note,created_at,updated_at,user_id,pinned,archived,trashed,version) VALUES(?,?,?,?,?,?,?,0,0,1)`, n.ID, n.Title, n.Note, n.CreatedAt, n.UpdatedAt, u.ID, n.Pinned)
	if err != nil {
		respondWithError(w, 503, "Could not save note; try again", nil)
		return
	}
	n.Archived = false
	n.Trashed = false
	respondWithJSON(w, 201, n)
}
func (cfg *apiConfig) handlerNotesUpdate(w http.ResponseWriter, r *http.Request, u database.User) {
	var n leafNote
	if !decodeJSON(w, r, &n) || !validateNote(w, &n) {
		return
	}
	id := chi.URLParam(r, "id")
	now := timestamp()
	// Ownership and version are checked by the same atomic write.
	result, err := cfg.conn.ExecContext(r.Context(), `UPDATE notes SET title=?,note=?,pinned=?,archived=?,trashed=?,updated_at=?,version=version+1 WHERE id=? AND user_id=? AND version=?`, n.Title, n.Note, n.Pinned, n.Archived, n.Trashed, now, id, u.ID, n.Version)
	if err != nil {
		respondWithError(w, 503, "Could not save note; try again", nil)
		return
	}
	count, err := result.RowsAffected()
	if err != nil {
		respondWithError(w, 503, "Could not verify save", nil)
		return
	}
	if count == 0 {
		var version int
		err = cfg.conn.QueryRowContext(r.Context(), `SELECT version FROM notes WHERE id=? AND user_id=?`, id, u.ID).Scan(&version)
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 404, "Note not found", nil)
		} else if err != nil {
			respondWithError(w, 503, "Could not verify note", nil)
		} else {
			respondWithError(w, 409, "This note changed in another tab. Export your draft, then reload notes before editing again.", nil)
		}
		return
	}
	n.ID = id
	n.UpdatedAt = now
	n.Version++
	respondWithJSON(w, 200, n)
}
func (cfg *apiConfig) exportNotes(w http.ResponseWriter, r *http.Request, u database.User) {
	rows, err := cfg.conn.QueryContext(r.Context(), "SELECT "+noteColumns+" FROM notes WHERE user_id=? ORDER BY created_at,id", u.ID)
	if err != nil {
		respondWithError(w, 503, "Could not export notes", nil)
		return
	}
	defer rows.Close()
	notes := []leafNote{}
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			respondWithError(w, 503, "Could not read export", nil)
			return
		}
		notes = append(notes, n)
	}
	if rows.Err() != nil {
		respondWithError(w, 503, "Could not finish export", nil)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="leafmark-notes.json"`)
	respondWithJSON(w, 200, map[string]any{"format": "leafmark-v1", "exported_at": timestamp(), "notes": notes})
}
