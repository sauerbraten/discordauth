package db

import (
	"fmt"
)

func (db *Database) AddStats(gameID int64, user string, frags, deaths, damage, shotDamage, flags int64) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	_, err := db.Exec(
		"insert into `stats` (`game`, `user`, `frags`, `deaths`, `damage`, `shotdamage`, `flags`) values (?, ?, ?, ?, ?, ?, ?)",
		gameID, user, frags, deaths, damage, shotDamage, flags,
	)
	if err != nil {
		return fmt.Errorf("db: error inserting stats of user '%s' in game %d into database: %v", user, gameID, err)
	}
	return nil
}
