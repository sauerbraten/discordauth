package db

import (
	"database/sql"
	"errors"
	"strings"
	"sync"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/file"
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
		return nil, errors.New("db: opening database: " + err.Error())
	}

	db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower) // use json struct tags

	_, err = db.Exec("pragma foreign_keys = on")
	if err != nil {
		db.Close()
		return nil, errors.New("db: enabling foreign keys: " + err.Error())
	}

	err = migrateUp(db.DB)
	if err != nil {
		db.Close()
		return nil, err
	}

	_, err = db.Exec("pragma journal_mode = wal")
	if err != nil {
		db.Close()
		return nil, errors.New("db: enabling WAL mode: " + err.Error())
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
		return errors.New("db: opening migration source files: " + err.Error())
	}
	defer src.Close()

	mDB, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return errors.New("db: using database for migrations: " + err.Error())
	}

	m, err := migrate.NewWithInstance("./migrations/", src, "users.sqlite", mDB)
	if err != nil {
		return errors.New("db: setting up migrations: " + err.Error())
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return errors.New("db: migrating database: " + err.Error())
	}

	return nil
}
