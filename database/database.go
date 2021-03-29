package database

import (
	"database/sql"
	"fmt"
	"log"
)

var DatabaseManager *sql.DB

func OpenDatabaseConnection() error {

	var err error
	log.Println("establish db")
	DatabaseManager, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@127.0.0.1:5432/%s", "neron", "neronium196531abc", "tran"))

	return err
}

func InitializeDatabase() {

	timeframes := GetDatabaseSupportedTimeframes()

	for letter := range timeframes {

		for _, value := range timeframes[letter] {
			DatabaseManager.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS public.tran_candles_%d%s 
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
    CONSTRAINT primary_%d%s PRIMARY KEY (symbol, "openTime")
)`, value, letter, value, letter))

		}
	}

}
func GetDatabaseSupportedTimeframes() map[string][]uint {

	return map[string][]uint{
		"m": {1, 2, 3, 4, 5, 21, 72},
		"h": {1, 4, 6},
		"d": {1, 3},
		"w": {1},
		"M": {1},
	}
}
