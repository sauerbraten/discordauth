package db

import (
	"fmt"
)

func (db *Database) AddGame(serverID, mode int64, mapname string) (int64, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	res, err := db.Exec("insert into `games` (`server`, `mode`, `map`) values (?, ?, ?)", serverID, mode, mapname)
	if err != nil {
		return -1, fmt.Errorf("db: error inserting game on server with ID %d (mode %d, map %s) into database: %v", serverID, mode, mapname, err)
	}
	return res.LastInsertId()
}
