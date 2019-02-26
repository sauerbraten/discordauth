package db

import (
	"fmt"
)

func (db *Database) AddGame(serverID, mode int64) (int64, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	res, err := db.Exec("insert into `games` (`server`, `mode`) values (?, ?)", serverID, mode)
	if err != nil {
		return -1, fmt.Errorf("db: error inserting game on server with ID %d (mode %d) into database: %v", serverID, mode, err)
	}
	return res.LastInsertId()
}
