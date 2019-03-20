package db

import (
	"fmt"
	"strings"

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

type Stats struct {
	User      string `json:"user"`
	Games     int    `json:"games"`
	Frags     int    `json:"frags"`
	Deaths    int    `json:"deaths"`
	Damage    int    `json:"damage"`
	Potential int    `json:"potential"`
	Flags     int    `json:"flags"`
}

func (db *Database) GetStats(user string, game int64, mode gamemode.ID, mapname string) ([]Stats, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	args := []interface{}{}

	from := "from `stats`"

	wheres := []string{}
	if game > -1 {
		wheres = append(wheres, "`game` = ?")
		args = append(args, game)
	} else if mode > -1 || mapname != "" {
		innerWheres := []string{}
		if mode > -1 {
			innerWheres = append(innerWheres, "`mode` = ?")
			args = append(args, mode)
		}
		if mapname != "" {
			innerWheres = append(innerWheres, "`map` = ?")
			args = append(args, mapname)
		}
		wheres = append(wheres, "`game` in (select `id` from `games` where "+strings.Join(innerWheres, " and ")+")")
	}
	if user != "" {
		wheres = append(wheres, "`user` = ?")
		args = append(args, user)
	}

	where := ""
	if len(wheres) > 0 {
		where = "where " + strings.Join(wheres, " and ")
	}

	rows, err := db.Query(fmt.Sprintln("select `user`, count(`game`), total(`frags`), total(`deaths`), total(`damage`), total(`potential`), total(`flags`)", from, where, "group by `user`"), args...)
	if err != nil {
		return nil, fmt.Errorf("db: error getting all users' stats: %v", err)
	}
	defer rows.Close()

	stats := []Stats{}

	for rows.Next() {
		s := Stats{}
		err = rows.Scan(&s.User, &s.Games, &s.Frags, &s.Deaths, &s.Damage, &s.Potential, &s.Flags)
		if err != nil {
			return nil, fmt.Errorf("db: error scanning row from 'stats' table: %v", err)
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}
