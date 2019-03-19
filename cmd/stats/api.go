package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/sauerbraten/waiter/pkg/definitions/gamemode"

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
	user, _mode, mapname := chi.URLParam(req, "user"), chi.URLParam(req, "mode"), chi.URLParam(req, "map")

	var (
		mode = int64(-1)
		err  error
	)
	if _mode != "" {
		mode, err = strconv.ParseInt(_mode, 10, 64)
		if err != nil {
			respondWithError(resp, http.StatusBadRequest, err)
			return
		}
	}

	games, err := a.db.GetAllGames(user, gamemode.ID(mode), mapname)
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
	name, _game, _mode, mapname := chi.URLParam(req, "name"), chi.URLParam(req, "game"), chi.URLParam(req, "mode"), chi.URLParam(req, "map")

	game, err := strconv.ParseInt(_game, 10, 64)
	if err != nil {
		respondWithError(resp, http.StatusBadRequest, err)
		return
	}

	mode := int64(-1)
	if _mode != "" {
		mode, err = strconv.ParseInt(_mode, 10, 64)
		if err != nil {
			respondWithError(resp, http.StatusBadRequest, err)
			return
		}
	}

	stats, err := a.db.GetStats(name, game, gamemode.ID(mode), mapname)
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
