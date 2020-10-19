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

func InitializeDatabase(Intervals []string) {

	for _, interval := range Intervals {

		DatabaseManager.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS public.tran_candles_%s 
(
    symbol character varying COLLATE pg_catalog."default" NOT NULL,
    "openTime" bigint NOT NULL,
    "closeTime" bigint NOT NULL,
    "prevCandle" bigint,
    "openPrice" double precision,
    "closePrice" double precision,
    "lowPrice" double precision,
    "highPrice" double precision,
    volume double precision,
    "quoteVolume" double precision,
    "takerVolume" double precision,
    "takerQuoteVolume" double precision,
    CONSTRAINT primary_%s PRIMARY KEY (symbol, "openTime")
)`, interval, interval))
	}
}
