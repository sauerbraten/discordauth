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

	return a
}

func (a *API) games(resp http.ResponseWriter, req *http.Request) {
	user, mode, mapname := req.FormValue("user"), req.FormValue("mode"), req.FormValue("map")

	games, err := a.db.GetAllGames(user, gamemode.Parse(mode), mapname)
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

func (a *API) stats(resp http.ResponseWriter, req *http.Request) {
	user, _game, mode, mapname := req.FormValue("user"), req.FormValue("game"), req.FormValue("mode"), req.FormValue("map")

	game := int64(-1)
	if _game != "" {
		var err error
		game, err = strconv.ParseInt(_game, 10, 64)
		if err != nil {
			respondWithError(resp, http.StatusBadRequest, err)
			return
		}
	}

	stats, err := a.db.GetStats(user, game, gamemode.Parse(mode), mapname)
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
