package main

import (
	"database/sql"
	"fmt"
)

func OpenDatabaseConnection() error {

	var err error
	DatabaseManager, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@157.230.174.164/%s", "neronru", "TESTSHIT", "trades"))

	return err
}
