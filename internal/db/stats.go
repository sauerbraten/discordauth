package db

import (
	"fmt"

	"github.com/sauerbraten/waiter/pkg/definitions/gamemode"
)

func (db *Database) AddStats(gameID int64, user string, frags, deaths, damage, potential, flags int64) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	_, err := db.Exec(
		"insert into `stats` (`game`, `user`, `frags`, `deaths`, `damage`, `potential`, `flags`) values (?, ?, ?, ?, ?, ?, ?)",
		gameID, user, frags, deaths, damage, potential, flags,
	)
	if err != nil {
		return fmt.Errorf("db: error inserting stats of user '%s' in game %d into database: %v", user, gameID, err)
	}
	return nil
}

type Game struct {
	ID   int    `json:"id"`
	Mode string `json:"mode"`
	Map  string `json:"map"`
}

type Stats struct {
	Game      Game `json:"game"`
	Frags     int  `json:"frags"`
	Deaths    int  `json:"deaths"`
	Damage    int  `json:"damage"`
	Potential int  `json:"potential"`
	Flags     int  `json:"flags"`
}

func (db *Database) GetStats(user string) ([]Stats, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	rows, err := db.Query("select `id`, `mode`, `map`, `frags`, `deaths`, `damage`, `potential`, `flags` from `stats`, `games` on `stats`.`game` = `games`.`id` where `user` = ?", user)
	if err != nil {
		return nil, fmt.Errorf("db: error getting stats of user %s: %v", user, err)
	}
	defer rows.Close()

	stats := []Stats{}

	for rows.Next() {
		s := Stats{}
		var _mode gamemode.ID
		rows.Scan(&s.Game.ID, &_mode, &s.Game.Map, &s.Frags, &s.Deaths, &s.Damage, &s.Potential, &s.Flags)
		s.Game.Mode = _mode.String()
		stats = append(stats, s)
	}

	return stats, rows.Err()
}
