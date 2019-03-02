package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi"
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

	a.HandleFunc("/player/{name}", a.player)

	return a
}

func (a *API) player(resp http.ResponseWriter, req *http.Request) {
	name := chi.URLParam(req, "name")

	stats, err := a.db.GetStats(name)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(err.Error()))
		return
	}

	err = json.NewEncoder(resp).Encode(stats)
	if err != nil {
		log.Println(err)
	}
}
