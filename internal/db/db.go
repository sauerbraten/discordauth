package db

import (
	"database/sql"
	"errors"
	"strings"
	"sync"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/sqlite3"
	"github.com/golang-migrate/migrate/source/file"
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
		db.Close()
		return nil, errors.New("db: could not enable foreign keys: " + err.Error())
	}

	err = migrateUp(db.DB)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Database{
		mutex: sync.Mutex{},
		DB:    db,
	}, nil
}

func migrateUp(db *sql.DB) error {
	srcFiles := &file.File{}
	src, err := srcFiles.Open("file://migrations")
	if err != nil {
		return errors.New("db: could not open migration source files: " + err.Error())
	}
	defer src.Close()

	mDB, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return errors.New("db: could not use database for migrations: " + err.Error())
	}

	m, err := migrate.NewWithInstance("./migrations/", src, "maitred.sqlite", mDB)
	if err != nil {
		return errors.New("db: could not set up migrations: " + err.Error())
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return errors.New("db: failed to migrate database: " + err.Error())
	}

	return nil
}
