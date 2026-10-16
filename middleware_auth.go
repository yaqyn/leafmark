package main

import (
	"database/sql"
	"errors"
	"github.com/yaqyn/leafmark/internal/database"
	"net/http"
)

type authedHandler func(http.ResponseWriter, *http.Request, database.User)

func (cfg *apiConfig) middlewareAuth(next authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := cfg.sessionUser(r)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) || errors.Is(err, http.ErrNoCookie) {
				respondWithError(w, 401, "Sign in to continue", nil)
			} else {
				respondWithError(w, 503, "Account unavailable; try again", nil)
			}
			return
		}
		next(w, r, user)
	}
}
