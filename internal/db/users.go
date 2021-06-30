package db

import (
	"database/sql"
	"fmt"
)

type User struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

type UserExistsError User

func (e UserExistsError) Error() string {
	return fmt.Sprintf("db: user %s already exists (with public key: %s)", e.Name, e.PublicKey)
}

func (db *Database) AddUser(name string, pubkey string, override bool) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	if !override {
		err := db.Get(&pubkey, "select `pubkey` from `users` where `name` = ?", name)
		if err == nil {
			return UserExistsError(User{name, pubkey})
		}
	}

	insert := "insert"
	if override {
		insert = "insert or replace"
	}

	_, err := db.Exec(fmt.Sprintf("%s into `users` (`name`, `pubkey`) values (?, ?)", insert), name, pubkey)
	if err != nil {
		return fmt.Errorf("db: inserting ('%s', '%s') into users table: %w", name, pubkey, err)
	}

	return nil
}

type UserNotFoundError string

func (e UserNotFoundError) Error() string {
	return fmt.Sprintf("db: no user named %s", string(e))
}

func (db *Database) GetPublicKey(name string) (pubkey string, err error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	err = db.Get(&pubkey, "select `pubkey` from `users` where `name` = ?", name)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", UserNotFoundError(name)
		}
		err = fmt.Errorf("db: retrieving public key of '%s': %v", name, err)
		return
	}

	return
}

func (db *Database) UpdateUserLastAuthed(name string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	_, err := db.Exec("update `users` set `last_authed_at` = strftime('%s', 'now') where `name` = ?", name)
	if err != nil {
		return fmt.Errorf("db: updating 'last_authed_at' field of user '%s': %v", name, err)
	}
	return nil
}

func (db *Database) DelUser(name string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	_, err := db.Exec("delete from `users` where `name` = ?", name)
	if err != nil {
		return fmt.Errorf("db: deleting '%s' from users table: %v", name, err)
	}
	return nil
}
