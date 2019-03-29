package db

import (
	"fmt"
	"strings"

	"github.com/sauerbraten/waiter/pkg/protocol/gamemode"
)

type Game struct {
	ID      int64    `json:"id"`
	Server  int64    `json:"server"`
	Mode    gmode    `json:"mode"`
	Map     string   `json:"map"`
	EndedAt int64    `json:"ended_at"`
	Players []string `json:"players,omitempty"`
}

type gmode string

func (m *gmode) Scan(v interface{}) error {
	_mode, ok := v.(int64)
	if !ok {
		return fmt.Errorf("db: can't scan %v (type %T) into game mode field", v, v)
	}
	mode := gamemode.ID(_mode)
	*m = gmode(mode.String())
	return nil
}

func (db *Database) AddGame(serverID, mode int64, mapname string) (int64, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	res, err := db.Exec("insert into `games` (`server`, `mode`, `map`) values (?, ?, ?)", serverID, mode, mapname)
	if err != nil {
		return -1, fmt.Errorf("db: error inserting game on server with ID %d (mode %d, map %s): %v", serverID, mode, mapname, err)
	}
	return res.LastInsertId()
}

func (db *Database) GetAllGames(user string, mode gamemode.ID, mapname string) ([]Game, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	args := []interface{}{}

	wheres := []string{}
	if mode > -1 {
		wheres = append(wheres, "`mode` = ?")
		args = append(args, mode)
	}
	if mapname != "" {
		wheres = append(wheres, "`map` = ?")
		args = append(args, mapname)
	}
	if user != "" {
		wheres = append(wheres, "`id` in (select `game` from `stats` where `user` = ?)")
		args = append(args, user)
	}

	where := ""
	if len(wheres) > 0 {
		where = "where " + strings.Join(wheres, " and ")
	}

	games := []Game{}

	err := db.Select(&games, "select `id`, `server`, `mode`, `map`, `ended_at` from `games` "+where, args...)
	if err != nil {
		return nil, fmt.Errorf("db: error retrieving games: %v", err)
	}
	return games, nil
}

func (db *Database) GetGame(id int64) (Game, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	g := Game{}
	err := db.Get(&g, "select `id`, `server`, `mode`, `map`, `ended_at` from `games` where `id` = ?", id)
	if err != nil {
		return g, fmt.Errorf("db: error retrieving game with ID %d: %v", id, err)
	}

	err = db.Select(&g.Players, "select `user` from `stats` where `game` = ?", id)
	if err != nil {
		return g, fmt.Errorf("db: error retrieving all players of game with ID %d: %v", id, err)
	}

	return g, nil
}
