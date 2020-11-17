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

func isAllCandlesLoaded(symbol string, timeframe string) (bool, error) {

	value := 0
	err := database.DatabaseManager.QueryRow(fmt.Sprintf(`SELECT 1 FROM (SELECT "prevCandle" FROM public.tran_candles_%s WHERE symbol =$1  ORDER BY "openTime" ASC LIMIT 1) t WHERE t."prevCandle" = 0;`, timeframe), symbol).Scan(&value)

	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	return value == 1, nil
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

			log.Println("load data fyck")

		}

	}
}

func checkIntervals(klines []candlescommon.KLine, minutes uint) bool {

	for i := 0; i < len(klines); i++ {

		if klines[i].OpenTime%uint64(minutes*60*1000) != 0 {
			return false
		}
	}

	return true
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

	counter := uint(0)

	currentTimeframe := timeframe

	for counter < limit {

		loadedKlines, _ := providers.GetKlinesNew(symbol, fmt.Sprintf("%d%s", currentTimeframe, interval.Letter), providers.GetKlineRange{FromTimestamp: firstDBKline, Direction: 0})

		if len(loadedKlines) == 0 {
			break
		}

		if interval.Letter == "m" {

			correctness := checkIntervals(loadedKlines, currentTimeframe)

			if !correctness && currentTimeframe != 1 {
				currentTimeframe = 1
				log.Println("Some candles have wrong open time!")
				continue
			} else if !correctness && currentTimeframe == 1 {
				log.Fatal("can't get value 1m is wrong")
			}
		}

		if interval.Letter == "h" && interval.Duration != timeframe {
			loadedKlines = candlescommon.HoursGroupKlineDesc(loadedKlines, uint64(interval.Duration))
		} else if interval.Letter == "m" && interval.Duration != timeframe {
			loadedKlines = candlescommon.MinutesGroupKlineDesc(loadedKlines, uint64(interval.Duration))
		}

		SaveCandles(loadedKlines, interval)

		firstDBKline = loadedKlines[len(loadedKlines)-1].OpenTime
		counter += uint(len(loadedKlines))
		currentTimeframe = timeframe

	}

}
