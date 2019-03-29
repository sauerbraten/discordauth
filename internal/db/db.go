package db

import (
	"errors"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	_ "github.com/mattn/go-sqlite3" // driver
)

type Database struct {
	mutex sync.Mutex // not embedded so that access to Mutex.Lock() and Mutex.Unlock() is not exported
	*sqlx.DB
}

func New(path string) (*Database, error) {
	db, err := sqlx.Open("sqlite3", path)
	if err != nil {
		return nil, errors.New("db: could not open database: " + err.Error())
	}

	db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower) // use json struct tags

	_, err = db.Exec("pragma foreign_keys = on")
	if err != nil {
		return nil, errors.New("db: could not enable foreign keys: " + err.Error())
	}

	return &Database{
		mutex: sync.Mutex{},
		DB:    db,
	}, nil
}
