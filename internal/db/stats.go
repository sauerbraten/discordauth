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
	User      string `json:"user"`
	Frags     int    `json:"frags"`
	Deaths    int    `json:"deaths"`
	Damage    int    `json:"damage"`
	Potential int    `json:"potential"`
	Flags     int    `json:"flags"`
}

func (db *Database) GetStats(user string) (Stats, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	s := Stats{
		User: user,
	}

	err := db.
		QueryRow("select total(`frags`), total(`deaths`), total(`damage`), total(`potential`), total(`flags`) from `stats` where `user` = ?", user).
		Scan(&s.Frags, &s.Deaths, &s.Damage, &s.Potential, &s.Flags)
	if err != nil {
		return s, fmt.Errorf("db: error retrieving stats of user %s: %v", user, err)
	}

	return s, nil
}

func (db *Database) GetAllStats() ([]Stats, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	rows, err := db.Query("select `user`, total(`frags`), total(`deaths`), total(`damage`), total(`potential`), total(`flags`) from `stats` group by `user`")
	if err != nil {
		return nil, fmt.Errorf("db: error getting all users' stats: %v", err)
	}
	defer rows.Close()

	stats := []Stats{}

	for rows.Next() {
		s := Stats{}
		err = rows.Scan(&s.User, &s.Frags, &s.Deaths, &s.Damage, &s.Potential, &s.Flags)
		if err != nil {
			return nil, fmt.Errorf("db: error scanning row from 'stats' table: %v", err)
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}
