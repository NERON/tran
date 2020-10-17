package database

import (
	"database/sql"
	"fmt"
)

var DatabaseManager *sql.DB

func OpenDatabaseConnection() error {

	var err error
	DatabaseManager, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@127.0.0.1:5432/%s", "neron", "neronium196531abc", "tran"))

	return err
}
