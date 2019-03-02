package main

import (
	"errors"
	"os"

	"github.com/sauerbraten/jsonfile"

	"github.com/sauerbraten/maitred/internal/db"
)

type config struct {
	db                  *db.Database
	webInterfaceAddress string
}

var conf config

func init() {
	configFilePath := "api_config.json"
	if len(os.Args) >= 2 {
		configFilePath = os.Args[1]
	}

	_conf := struct {
		DatabaseFilePath    string `json:"db_file_path"`
		WebInterfaceAddress string `json:"web_interface_address"`
	}{}

	err := jsonfile.ParseFile(configFilePath, &_conf)
	if err != nil {
		panic(err)
	}

	_db, err := db.New(_conf.DatabaseFilePath)
	if err != nil {
		panic(errors.New("database initialization failed: " + err.Error()))
	}

	conf = config{
		db:                  _db,
		webInterfaceAddress: _conf.WebInterfaceAddress,
	}
}
