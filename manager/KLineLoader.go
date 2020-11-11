package manager

import (
	"database/sql"
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/database"
	"github.com/NERON/tran/providers"
	"log"
)

func GetLastKlineForSymbol(symbol string, timeframe string) (uint64, error) {

	timestamp := uint64(0)

	err := database.DatabaseManager.QueryRow(fmt.Sprintf(`SELECT "openTime" FROM public.tran_candles_%s WHERE symbol ='$1' ORDER BY "openTime" DESC LIMIT 1;`, timeframe), symbol).Scan(&timestamp)

	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	return timestamp, nil
}
func GetFirstKlineForSymbol(symbol string, timeframe string) (uint64, error) {

	timestamp := uint64(0)

	err := database.DatabaseManager.QueryRow(fmt.Sprintf(`SELECT "openTime" FROM public.tran_candles_%s WHERE symbol ='$1' ORDER BY "openTime" ASC LIMIT 1;`, timeframe), symbol).Scan(&timestamp)

	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	return timestamp, nil
}

func LoadKlinesToDatabase(symbol string, interval candlescommon.Interval, up bool) {

	timestamp := uint64(0)

	var err error

	//get first or latest save kline in database
	if up {

		timestamp, err = GetLastKlineForSymbol(symbol, fmt.Sprintf("%d%s", interval.Duration, interval.Letter))

	} else {

		timestamp, err = GetFirstKlineForSymbol(symbol, fmt.Sprintf("%d%s", interval.Duration, interval.Letter))
	}

	if err != nil {
		//TODO: handle issue
		return
	}

	//detect more suitable interval
	timeframes := providers.GetSupportedTimeframes()
	optimalTimeFrame := uint(0)

	for _, val := range timeframes[interval.Letter] {

		if interval.Duration%val == 0 {
			optimalTimeFrame = val
		}
	}

	if optimalTimeFrame == 0 {
		log.Fatal("Not supported")
	}

	log.Println("Choosed interval:", optimalTimeFrame, timestamp)

}
