package db

import (
	"fmt"
)

type Ban struct {
	Name string `json:"name"`
}

func (db *Database) AddBan(name string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	_, err := db.Exec("insert into `bans` (`name`) values (?)", name)
	if err != nil {
		return fmt.Errorf("db: inserting ('%s') into bans table: %w", name, err)
	}

	return nil
}

func (db *Database) IsBanned(name string) (bool, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	count := 0
	err := db.Get(&count, "select count(*) from `bans` where `name` = ?", name)
	if err != nil {
		return false, fmt.Errorf("db: checking if '%s' is in bans table: %v", name, err)
	}
	return count == 1, nil
}

func (db *Database) DelBan(name string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	_, err := db.Exec("delete from `bans` where `name` = ?", name)
	if err != nil {
		return fmt.Errorf("db: deleting '%s' from bans table: %v", name, err)
	}
	return nil
}
