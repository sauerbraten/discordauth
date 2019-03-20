package db

import (
	"fmt"
	"strings"

	"github.com/sauerbraten/waiter/pkg/definitions/gamemode"
)

type Game struct {
	ID      int64    `json:"id"`
	Server  int64    `json:"server_id"`
	Mode    string   `json:"mode"`
	Map     string   `json:"map"`
	EndedAt int64    `json:"ended_at"`
	Players []string `json:"players,omitempty"`
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

	rows, err := db.Query("select `id`, `server`, `mode`, `map`, `ended_at` from `games` "+where, args...)
	if err != nil {
		return nil, fmt.Errorf("db: error retrieving all games: %v", err)
	}
	defer rows.Close()

	games := []Game{}

	for rows.Next() {
		g := Game{}
		var _mode gamemode.ID
		err = rows.Scan(&g.ID, &g.Server, &_mode, &g.Map, &g.EndedAt)
		if err != nil {
			return nil, fmt.Errorf("db: error scanning row from 'games' table: %v", err)
		}
		g.Mode = _mode.String()
		games = append(games, g)
	}

	return games, rows.Err()
}

func (db *Database) GetGame(id int64) (Game, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	g := Game{}
	var _mode gamemode.ID
	err := db.QueryRow("select `id`, `server`, `mode`, `map`, `ended_at` from `games` where `id` = ?", id).
		Scan(&g.ID, &g.Server, &_mode, &g.Map, &g.EndedAt)
	if err != nil {
		return g, fmt.Errorf("db: error retrieving game with ID %d: %v", id, err)
	}
	g.Mode = _mode.String()

	rows, err := db.Query("select `user` from `stats` where `game` = ?", id)
	if err != nil {
		return g, fmt.Errorf("db: error retrieving all players of game with ID %d: %v", id, err)
	}
	defer rows.Close()

	for rows.Next() {
		var p string
		err = rows.Scan(&p)
		if err != nil {
			return g, fmt.Errorf("db: error scanning `user` column from 'stats' table: %v", err)
		}
		g.Players = append(g.Players, p)
	}

	return g, rows.Err()
}
