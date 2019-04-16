package db

import (
	"database/sql"
	"fmt"

	"github.com/sauerbraten/maitred/pkg/auth"
)

type User struct {
	Name      string         `json:"name"`
	PublicKey auth.PublicKey `json:"public_key"`
}

func (db *Database) UserExists(name string) (bool, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	var _name string
	err := db.Get(&_name, "select `name` from `users` where `name` = ?", name)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("db: error looking up user name '%s' in database: %v", name, err)
	}
	return _name == name, nil
}

func (db *Database) AddUser(name, pubkey string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	_, err := db.Exec("insert into `users` (`name`, `pubkey`) values (?, ?)", name, pubkey)
	if err != nil {
		return fmt.Errorf("db: error inserting '%s' (%s) into database: %v", name, pubkey, err)
	}
	return nil
}

func (db *Database) GetPublicKey(name string) (pubkey auth.PublicKey, err error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	var _pubkey string
	err = db.Get(&_pubkey, "select `pubkey` from `users` where `name` = ?", name)
	if err != nil {
		err = fmt.Errorf("db: error retrieving public key of '%s': %v", name, err)
		return
	}

	pubkey, err = auth.ParsePublicKey(_pubkey)
	if err != nil {
		err = fmt.Errorf("db: error parsing public key '%s': %v", _pubkey, err)
	}
	return
}

func (db *Database) UpdateUserLastAuthed(name string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	_, err := db.Exec("update `users` set `last_authed_at` = strftime('%s', 'now') where `name` = ?", name)
	if err != nil {
		return fmt.Errorf("db: error updating 'last_authed_at' field of user '%s': %v", name, err)
	}
	return nil
}

func (db *Database) DelUser(name string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	_, err := db.Exec("delete from `users` where `name` = ?", name)
	if err != nil {
		return fmt.Errorf("db: error deleting '%s': %v", name, err)
	}
	return nil
}
