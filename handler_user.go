package main

import (
	"github.com/yaqyn/leafmark/internal/database"
	"net/http"
)

func (cfg *apiConfig) handlerUsersGet(w http.ResponseWriter, r *http.Request, user database.User) {
	respondWithJSON(w, 200, map[string]string{"id": user.ID, "name": user.Name})
}
