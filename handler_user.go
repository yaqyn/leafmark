package main

import (
	"github.com/bootdotdev/learn-cicd-starter/internal/database"
	"net/http"
)

func (cfg *apiConfig) handlerUsersGet(w http.ResponseWriter, r *http.Request, user database.User) {
	respondWithJSON(w, 200, map[string]string{"id": user.ID, "name": user.Name})
}
