package manager

import (
	"database/sql"
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/database"
	"github.com/NERON/tran/providers"
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
func GetOptimalDatabaseTimeframe(interval candlescommon.Interval) uint {

	//detect more suitable interval
	timeframes := database.GetDatabaseSupportedTimeframes()
	optimalTimeFrame := uint(0)

	for _, val := range timeframes[interval.Letter] {

		if interval.Duration%val == 0 {
			optimalTimeFrame = val
		}
	}

	return optimalTimeFrame
}

func FillDatabaseToLatestValues(symbol string, interval candlescommon.Interval) {

	//choose optimal load timeframe
	timeframe := GetOptimalLoadTimeframe(interval)

	//get latest values in database
	latestDBKlines, err := GetLastKlineForSymbol(symbol, fmt.Sprintf("%d%s", interval.Duration, interval.Letter))

	//check for database error
	if err != nil {
		return
	}

	if latestDBKlines == 0 {

		klines, _ := providers.GetLastKlines(symbol, fmt.Sprintf("%d%s", timeframe, interval.Letter))

		if interval.Letter == "h" {
			klines = candlescommon.HoursGroupKlineDesc(klines, uint64(interval.Duration))
		} else if interval.Letter == "m" {
			klines = candlescommon.MinutesGroupKlineDesc(klines, uint64(interval.Duration))
		}

		SaveCandles(klines)

	} else {

	}
}
