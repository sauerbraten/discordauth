package db

import (
	"fmt"
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

type Stats struct {
	Game      int64 `json:"game_id"`
	Frags     int   `json:"frags"`
	Deaths    int   `json:"deaths"`
	Damage    int   `json:"damage"`
	Potential int   `json:"potential"`
	Flags     int   `json:"flags"`
}

func (db *Database) GetStats(user string) ([]Stats, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	rows, err := db.Query("select `game`, `frags`, `deaths`, `damage`, `potential`, `flags` from `stats` where `user` = ?", user)
	if err != nil {
		return nil, fmt.Errorf("db: error getting stats of user %s: %v", user, err)
	}
	defer rows.Close()

	stats := []Stats{}

	for rows.Next() {
		s := Stats{}
		err = rows.Scan(&s.Game, &s.Frags, &s.Deaths, &s.Damage, &s.Potential, &s.Flags)
		if err != nil {
			return nil, fmt.Errorf("db: error scanning row from 'stats' table: %v", err)
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}
