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

	err := database.DatabaseManager.QueryRow(fmt.Sprintf(`SELECT "openTime" FROM public.tran_candles_%s WHERE symbol =$1 ORDER BY "openTime" DESC LIMIT 1;`, timeframe), symbol).Scan(&timestamp)

	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	return timestamp, nil
}
func GetFirstKlineForSymbol(symbol string, timeframe string) (uint64, error) {

	timestamp := uint64(0)

	err := database.DatabaseManager.QueryRow(fmt.Sprintf(`SELECT "openTime" FROM public.tran_candles_%s WHERE symbol =$1 ORDER BY "openTime" ASC LIMIT 1;`, timeframe), symbol).Scan(&timestamp)

	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	return timestamp, nil
}

func GetOptimalLoadTimeframe(interval candlescommon.Interval) uint {

	//detect more suitable interval
	timeframes := providers.GetSupportedTimeframes()
	optimalTimeFrame := uint(0)

	for _, val := range timeframes[interval.Letter] {

		if interval.Duration%val == 0 {
			optimalTimeFrame = val
		}
	}

	return optimalTimeFrame
}

func LoadKlinesToDatabase(symbol string, interval candlescommon.Interval, up bool, limit uint) {

	startTimestamp := uint64(0)
	endTimestamp := uint64(0)

	var err error

	//get first or latest save kline in database
	if up {

		startTimestamp, err = GetLastKlineForSymbol(symbol, fmt.Sprintf("%d%s", interval.Duration, interval.Letter))

	} else {

		endTimestamp, err = GetFirstKlineForSymbol(symbol, fmt.Sprintf("%d%s", interval.Duration, interval.Letter))
	}

	if err != nil {
		log.Fatal("error", err.Error())
		return
	}

	optimalTimeFrame := GetOptimalLoadTimeframe(interval)

	if optimalTimeFrame == 0 {
		return
	}

	klines, err := providers.GetKlinesNew(symbol, fmt.Sprintf("%d%s", optimalTimeFrame, interval.Letter), providers.GetKlineRange{})

	if len(klines) == 0 {
		log.Println("Klines nil")
		return
	}

	log.Println(startTimestamp, endTimestamp)

	klines = candlescommon.MinutesGroupKlineAsc(klines, uint64(optimalTimeFrame), uint64(interval.Duration))

	log.Println(klines)

}
