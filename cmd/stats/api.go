package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"

	"github.com/sauerbraten/maitred/internal/db"
)

type API struct {
	chi.Router
	db *db.Database
}

func NewAPI(db *db.Database) *API {
	a := &API{
		Router: chi.NewRouter(),
		db:     db,
	}

	a.Use(middleware.SetHeader("Content-Type", "application/json; charset=utf-8"))

	a.HandleFunc("/games", a.games)
	a.HandleFunc("/game/{id}", a.game)
	a.HandleFunc("/stats", a.stats)
	a.HandleFunc("/stats/{name}", a.userStats)

	return a
}

func (a *API) games(resp http.ResponseWriter, req *http.Request) {
	// todo: ?user ?mode ?map

	games, err := a.db.GetAllGames()
	if err != nil {
		respondWithError(resp, http.StatusInternalServerError, err)
		return
	}

	err = json.NewEncoder(resp).Encode(games)
	if err != nil {
		log.Println(err)
	}
}

func (a *API) game(resp http.ResponseWriter, req *http.Request) {
	_id := chi.URLParam(req, "id")
	id, err := strconv.ParseInt(_id, 10, 64)
	if err != nil {
		respondWithError(resp, http.StatusBadRequest, err)
		return
	}

	game, err := a.db.GetGame(id)
	if err != nil {
		respondWithError(resp, http.StatusInternalServerError, err)
		return
	}

	err = json.NewEncoder(resp).Encode(game)
	if err != nil {
		log.Println(err)
	}
}

func (a *API) userStats(resp http.ResponseWriter, req *http.Request) {
	// todo: ?game ?mode ?map

	name := chi.URLParam(req, "name")

	stats, err := a.db.GetStats(name)
	if err != nil {
		respondWithError(resp, http.StatusInternalServerError, err)
		return
	}

	err = json.NewEncoder(resp).Encode(stats)
	if err != nil {
		log.Println(err)
	}
}

func (a *API) stats(resp http.ResponseWriter, req *http.Request) {
	// todo: ?game ?mode ?map (?sortBy)

	stats, err := a.db.GetAllStats()
	if err != nil {
		respondWithError(resp, http.StatusInternalServerError, err)
		return
	}

	err = json.NewEncoder(resp).Encode(stats)
	if err != nil {
		log.Println(err)
	}
}

func respondWithError(resp http.ResponseWriter, statusCode int, err error) {
	resp.WriteHeader(statusCode)
	err = json.NewEncoder(resp).Encode(struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	})
	if err != nil {
		log.Println(err)
	}
}
