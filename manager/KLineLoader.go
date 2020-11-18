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

func checkKlinesForInterval(klines []candlescommon.KLine, interval candlescommon.Interval) bool {

	if interval.Letter == "m" {

		for i := 0; i < len(klines); i++ {

			if klines[i].OpenTime%uint64(interval.Duration*60*1000) != 0 {
				log.Println("Wrong open value", klines[i])
				return false
			}

			if klines[i].CloseTime-klines[i].OpenTime+1 != uint64(interval.Duration*60*1000) {
				log.Println("Wrong close value", klines[i])
				return false
			}

			if klines[i].PrevCloseCandleTimestamp != 0 && (klines[i].PrevCloseCandleTimestamp+1)%uint64(interval.Duration*60*1000) != 0 {
				log.Println("Wrong prev close value", klines[i])
				return false
			}
		}
	}

	return true
}
func fixKlinesForInterval(klines []candlescommon.KLine, interval candlescommon.Interval) {

	for i := 0; i < len(klines); i++ {

		klines[i].OpenTime = (klines[i].OpenTime / uint64(interval.Duration*60*1000)) * uint64(interval.Duration*60*1000)
		klines[i].CloseTime = klines[i].OpenTime + uint64(interval.Duration*60*1000) - 1

		if i > 0 {
			klines[i-1].PrevCloseCandleTimestamp = klines[i].CloseTime
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

	counter := uint(0)

	brokenKlines := make([]candlescommon.KLine, 0)

	for counter < limit {

		loadedKlines, _ := providers.GetKlinesNew(symbol, fmt.Sprintf("%d%s", timeframe, interval.Letter), providers.GetKlineRange{FromTimestamp: firstDBKline, Direction: 0})

		log.Println("loaded klines", len(loadedKlines))

		correct := checkKlinesForInterval(loadedKlines, candlescommon.Interval{Letter: interval.Letter, Duration: timeframe})

		if !correct && len(loadedKlines) > 0 {

			brokenKlines = append(brokenKlines, loadedKlines...)
			log.Println("Broken klines", len(brokenKlines), brokenKlines[len(brokenKlines)-1].OpenTime)
			firstDBKline = brokenKlines[len(brokenKlines)-1].OpenTime
			continue

		} else if len(brokenKlines) > 0 {

			fixKlinesForInterval(brokenKlines, candlescommon.Interval{Letter: interval.Letter, Duration: timeframe})
			brokenKlines = append(brokenKlines, loadedKlines...)
			loadedKlines = brokenKlines
			brokenKlines = nil
			log.Println("emit fixed klines")
		}

		if len(loadedKlines) == 0 {
			break
		}

		originalKlines := loadedKlines

		if interval.Letter == "h" && interval.Duration != timeframe {
			loadedKlines = candlescommon.HoursGroupKlineDesc(loadedKlines, uint64(interval.Duration))
		} else if interval.Letter == "m" && interval.Duration != timeframe {
			loadedKlines = candlescommon.MinutesGroupKlineDesc(loadedKlines, uint64(interval.Duration))
		}

		var prevClose = uint64(0)
		for _, loadKline := range loadedKlines {

			if loadKline.OpenTime == 1557882720000 || loadKline.CloseTime == 1557891359999 {
				log.Println(loadKline)
			}

			if prevClose != 0 && loadKline.CloseTime != prevClose {

				log.Println("found gap", loadKline)

				for _, originalLoad := range originalKlines {

					log.Println(originalLoad)
				}

				return
			}

			prevClose = loadKline.PrevCloseCandleTimestamp
		}

		SaveCandles(loadedKlines, interval)

		firstDBKline = loadedKlines[len(loadedKlines)-1].OpenTime
		counter += uint(len(loadedKlines))

	}

}
