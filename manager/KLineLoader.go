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
		log.Println(err.Error())
		return
	}

	//generare interval string
	intervalString := fmt.Sprintf("%d%s", timeframe, interval.Letter)

	if latestDBKlines == 0 {

		klines, _ := providers.GetLastKlines(symbol, intervalString)

		if interval.Letter == "h" && interval.Duration != timeframe {
			klines = candlescommon.HoursGroupKlineDesc(klines, uint64(interval.Duration))
		} else if interval.Letter == "m" && interval.Duration != timeframe {
			klines = candlescommon.MinutesGroupKlineDesc(klines, uint64(interval.Duration))
		}

		SaveCandles(klines, interval)

	} else {

		for {

			loadedKlines, _ := providers.GetKlinesNew(symbol, intervalString, providers.GetKlineRange{FromTimestamp: latestDBKlines, Direction: 1})

			if len(loadedKlines) == 0 {
				break
			}

			if interval.Letter == "h" && interval.Duration != timeframe {
				loadedKlines = candlescommon.HoursGroupKlineDesc(loadedKlines, uint64(interval.Duration))
			} else if interval.Letter == "m" && interval.Duration != timeframe {
				loadedKlines = candlescommon.MinutesGroupKlineDesc(loadedKlines, uint64(interval.Duration))
			}

			for i := 0; i < len(loadedKlines)/2; i++ {
				j := len(loadedKlines) - i - 1
				loadedKlines[i], loadedKlines[j] = loadedKlines[j], loadedKlines[i]
			}

			SaveCandles(loadedKlines, interval)

			if loadedKlines[len(loadedKlines)-1].Closed == false {
				break
			}

			latestDBKlines = loadedKlines[len(loadedKlines)-1].OpenTime

		}

	}
}
func FillDatabaseWithPrevValues(symbol string, interval candlescommon.Interval, limit uint) {

	//choose optimal load timeframe
	timeframe := GetOptimalLoadTimeframe(interval)

	//get latest values in database
	firstDBKline, err := GetFirstKlineForSymbol(symbol, fmt.Sprintf("%d%s", interval.Duration, interval.Letter))

	//check for database error
	if err != nil {
		log.Println(err.Error())
		return
	}

	if firstDBKline == 0 {
		return
	}

	//generate interval string
	intervalString := fmt.Sprintf("%d%s", timeframe, interval.Letter)

	counter := uint(0)

	for counter < limit {

		loadedKlines, _ := providers.GetKlinesNew(symbol, intervalString, providers.GetKlineRange{FromTimestamp: firstDBKline, Direction: 0})

		if len(loadedKlines) == 0 {
			break
		}

		if interval.Letter == "h" && interval.Duration != timeframe {
			loadedKlines = candlescommon.HoursGroupKlineDesc(loadedKlines, uint64(interval.Duration))
		} else if interval.Letter == "m" && interval.Duration != timeframe {
			loadedKlines = candlescommon.MinutesGroupKlineDesc(loadedKlines, uint64(interval.Duration))
		}

		SaveCandles(loadedKlines, interval)

		firstDBKline = loadedKlines[len(loadedKlines)-1].OpenTime
		counter += uint(len(loadedKlines))

		log.Println("saved value ", counter)

	}

}
