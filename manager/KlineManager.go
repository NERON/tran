package manager

import (
	"errors"
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/database"
	"github.com/NERON/tran/providers"
	"log"
	"math"
	"time"
)

func getKlinesFromDatabase(symbol string, interval candlescommon.Interval, endTimestamp uint64, limit int) ([]candlescommon.KLine, error) {

	rows, err := database.DatabaseManager.Query(fmt.Sprintf(`SELECT symbol, "openTime", "closeTime", "prevCandle", "openPrice", "closePrice", "lowPrice", "highPrice","volume", "quoteVolume", "takerVolume", "takerQuoteVolume"
	FROM public.tran_candles_%d%s WHERE symbol = $1 AND "openTime" < $2 ORDER BY "openTime" DESC LIMIT %d`, interval.Duration, interval.Letter, limit), symbol, endTimestamp)

	if err != nil {
		return nil, err
	}

	databaseCandles := make([]candlescommon.KLine, 0)

	prevCandleClose := uint64(0)

	for rows.Next() {

		kline := candlescommon.KLine{}

		err = rows.Scan(&kline.Symbol, &kline.OpenTime, &kline.CloseTime, &kline.PrevCloseCandleTimestamp, &kline.OpenPrice, &kline.ClosePrice, &kline.LowPrice, &kline.HighPrice, &kline.BaseVolume, &kline.QuoteVolume, &kline.TakerBuyBaseVolume, &kline.TakerBuyQuoteVolume)

		if err != nil {

			rows.Close()
			return nil, err
		}

		if prevCandleClose > 0 && prevCandleClose != kline.CloseTime {
			rows.Close()
			return nil, errors.New(fmt.Sprintf("gap found %d", kline.OpenTime))
		}

		kline.Closed = true

		databaseCandles = append(databaseCandles, kline)

		prevCandleClose = kline.PrevCloseCandleTimestamp
	}

	rows.Close()

	return databaseCandles, nil

}
func getKlinesFromDatabaseAscending(symbol string, interval candlescommon.Interval, startTimestamp uint64, limit int) ([]candlescommon.KLine, error) {

	rows, err := database.DatabaseManager.Query(fmt.Sprintf(`SELECT symbol, "openTime", "closeTime", "prevCandle", "openPrice", "closePrice", "lowPrice", "highPrice","volume", "quoteVolume", "takerVolume", "takerQuoteVolume"
	FROM public.tran_candles_%d%s WHERE symbol = $1 AND "openTime" > $2 ORDER BY "openTime" ASC LIMIT %d`, interval.Duration, interval.Letter, limit), symbol, startTimestamp)

	if err != nil {
		return nil, err
	}

	databaseCandles := make([]candlescommon.KLine, 0)

	for rows.Next() {

		kline := candlescommon.KLine{}

		err = rows.Scan(&kline.Symbol, &kline.OpenTime, &kline.CloseTime, &kline.PrevCloseCandleTimestamp, &kline.OpenPrice, &kline.ClosePrice, &kline.LowPrice, &kline.HighPrice, &kline.BaseVolume, &kline.QuoteVolume, &kline.TakerBuyBaseVolume, &kline.TakerBuyQuoteVolume)

		if err != nil {

			rows.Close()
			return nil, err
		}

		kline.Closed = true

		databaseCandles = append(databaseCandles, kline)

	}

	rows.Close()

	return databaseCandles, nil

}
func convertKlinesToNewTimestamp(klines []candlescommon.KLine, interval candlescommon.Interval) []candlescommon.KLine {

	if interval.Letter == "h" {
		klines = candlescommon.HoursGroupKlineDesc(klines, uint64(interval.Duration), true, false)
	} else if interval.Letter == "m" {
		klines = candlescommon.MinutesGroupKlineDesc(klines, uint64(interval.Duration), true, false)
	}

	return klines
}
func GetFirstKLines(symbol string, interval candlescommon.Interval, limit int) ([]candlescommon.KLine, error) {

	//get interval for loading data
	databaseInterval := GetOptimalDatabaseTimeframe(interval)

	//construct new timeframe
	databaseIn := candlescommon.Interval{Letter: interval.Letter, Duration: databaseInterval}

	//get minimum value
	_, min, err := IsAllCandlesLoaded(symbol, fmt.Sprintf("%d%s", databaseIn.Duration, databaseIn.Letter))

	//check for error
	if err != nil {
		log.Fatal(err.Error())
	}

	//fill values to the end
	if min != 0 {
		FillDatabaseWithPrevValues(symbol, databaseIn, math.MaxInt32)
	}

	fetchedData, err := getKlinesFromDatabaseAscending(symbol, databaseIn, 0, limit)

	if err != nil {
		log.Fatal(err.Error())
	}

	for i := 0; i < len(fetchedData)/2; i++ {
		j := len(fetchedData) - i - 1
		fetchedData[i], fetchedData[j] = fetchedData[j], fetchedData[i]
	}

	if interval.Letter == "h" {
		fetchedData = candlescommon.HoursGroupKlineDesc(fetchedData, uint64(interval.Duration), false, true)
	} else if interval.Letter == "m" {
		fetchedData = candlescommon.MinutesGroupKlineDesc(fetchedData, uint64(interval.Duration), false, true)
	}

	for i := 0; i < len(fetchedData)/2; i++ {
		j := len(fetchedData) - i - 1
		fetchedData[i], fetchedData[j] = fetchedData[j], fetchedData[i]
	}

	return fetchedData, nil
}
func GetKLinesInRange(symbol string, interval candlescommon.Interval, fromTimestamp uint64, endTimestamp uint64, limit int) ([]candlescommon.KLine, error) {

	//get interval for loading data
	databaseInterval := GetOptimalDatabaseTimeframe(interval)

	//construct new timeframe
	databaseIn := candlescommon.Interval{Letter: interval.Letter, Duration: databaseInterval}

	//get minimum value
	_, min, err := IsAllCandlesLoaded(symbol, fmt.Sprintf("%d%s", databaseIn.Duration, databaseIn.Letter))

	//check for error
	if err != nil {
		log.Fatal(err.Error())
	}

	if min != 0 {
		FillDatabaseWithPrevValues(symbol, databaseIn, math.MaxInt32)
	}

	klinesReceived := make([]candlescommon.KLine, 0)

	triedToLoad := false

	for len(klinesReceived) < limit {

		fetchedData, err := getKlinesFromDatabaseAscending(symbol, databaseIn, fromTimestamp, 1000)

		if err != nil {
			log.Fatal(err.Error())
		}

		for i := 0; i < len(fetchedData)/2; i++ {
			j := len(fetchedData) - i - 1
			fetchedData[i], fetchedData[j] = fetchedData[j], fetchedData[i]
		}

		if interval.Letter == "h" {
			fetchedData = candlescommon.HoursGroupKlineDesc(fetchedData, uint64(interval.Duration), false, true)
		} else if interval.Letter == "m" {
			fetchedData = candlescommon.MinutesGroupKlineDesc(fetchedData, uint64(interval.Duration), false, true)
		}

		if len(fetchedData) == 0 {

			if !triedToLoad {

				FillDatabaseToLatestValues(symbol, databaseIn)
				triedToLoad = true
				continue

			} else {
				break
			}

		}

		for i := 0; i < len(fetchedData)/2; i++ {
			j := len(fetchedData) - i - 1
			fetchedData[i], fetchedData[j] = fetchedData[j], fetchedData[i]
		}

		klinesReceived = append(klinesReceived, fetchedData...)

		if klinesReceived[len(klinesReceived)-1].OpenTime >= endTimestamp {
			break
		}

		fromTimestamp = klinesReceived[len(klinesReceived)-1].CloseTime
	}

	return klinesReceived, nil

}
func GetLastKLines(symbol string, interval candlescommon.Interval, limit int) ([]candlescommon.KLine, error) {

	databaseInterval := GetOptimalDatabaseTimeframe(interval)

	var loadInterval = uint(0)

	if databaseInterval != 0 {
		loadInterval = GetOptimalLoadTimeframe(candlescommon.Interval{Letter: interval.Letter, Duration: databaseInterval})
	}

	if loadInterval == 0 {
		loadInterval = GetOptimalLoadTimeframe(interval)
	}

	if loadInterval == 0 {
		return nil, errors.New("can't found optimal timeframe")
	}

	lastKlines, err := providers.GetLastKlines(symbol, fmt.Sprintf("%d%s", loadInterval, interval.Letter))

	if err != nil {
		return nil, err
	}

	if len(lastKlines) == 0 {
		return nil, errors.New("data empty")
	}

	if interval.Duration != loadInterval {

		lastKlines = convertKlinesToNewTimestamp(lastKlines, interval)

	}

	if interval.Letter == "d" || interval.Letter == "M" || interval.Letter == "w" || databaseInterval == 0 {

		if interval.Letter == "d" || interval.Letter == "w" {
			limit = 10000000000000
		}

		for len(lastKlines) < limit {

			fetchedKlines, err := providers.GetKlinesNew(symbol, fmt.Sprintf("%d%s", loadInterval, interval.Letter), providers.GetKlineRange{Direction: 0, FromTimestamp: lastKlines[len(lastKlines)-1].OpenTime})

			if err != nil {
				return nil, err
			}

			if len(fetchedKlines) == 0 {
				break
			}

			if interval.Duration != loadInterval {

				fetchedKlines = convertKlinesToNewTimestamp(fetchedKlines, interval)
			}

			lastKlines = append(lastKlines, fetchedKlines...)

			if lastKlines[len(lastKlines)-1].PrevCloseCandleTimestamp == 0 {
				break
			}
		}

	} else {

		databaseIn := candlescommon.Interval{Letter: interval.Letter, Duration: databaseInterval}

		_, min, err := IsAllCandlesLoaded(symbol, fmt.Sprintf("%d%s", databaseIn.Duration, databaseIn.Letter))

		if err != nil {
			log.Fatal(err.Error())
		}

		FillDatabaseToLatestValues(symbol, databaseIn)

		for len(lastKlines) < limit {

			fetchedKlines, err := getKlinesFromDatabase(symbol, databaseIn, lastKlines[len(lastKlines)-1].OpenTime, 1000)

			if err != nil {
				return nil, err
			}

			fetchedKlines = convertKlinesToNewTimestamp(fetchedKlines, interval)

			if len(fetchedKlines) == 0 && min != 0 {
				FillDatabaseWithPrevValues(symbol, databaseIn, 900)
				continue
			}

			lastKlines = append(lastKlines, fetchedKlines...)

			if fetchedKlines[len(fetchedKlines)-1].PrevCloseCandleTimestamp == 0 {
				break
			}

		}

	}

	if len(lastKlines) > limit {
		lastKlines = lastKlines[:limit]
	}

	for i := 0; i < len(lastKlines)/2; i++ {
		j := len(lastKlines) - i - 1
		lastKlines[i], lastKlines[j] = lastKlines[j], lastKlines[i]
	}

	if (interval.Letter == "d" || interval.Letter == "w") && interval.Duration != loadInterval {
		lastKlines = candlescommon.GroupKline(lastKlines, int(interval.Duration))
	}

	return lastKlines, nil
}
func GetLastKLinesFromTimestamp(symbol string, interval candlescommon.Interval, timestamp uint64, limit int) ([]candlescommon.KLine, error) {

	databaseInterval := GetOptimalDatabaseTimeframe(interval)

	var loadInterval = uint(0)

	if databaseInterval != 0 {
		loadInterval = GetOptimalLoadTimeframe(candlescommon.Interval{Letter: interval.Letter, Duration: databaseInterval})
	}

	if loadInterval == 0 {
		loadInterval = GetOptimalLoadTimeframe(interval)
	}

	if loadInterval == 0 {
		return nil, errors.New("can't found optimal timeframe")
	}

	lastKlines := make([]candlescommon.KLine, 0)

	if interval.Letter == "d" || interval.Letter == "M" || interval.Letter == "w" || databaseInterval == 0 {

		for len(lastKlines) < limit {

			fetchedKlines, err := providers.GetKlinesNew(symbol, fmt.Sprintf("%d%s", loadInterval, interval.Letter), providers.GetKlineRange{Direction: 0, FromTimestamp: timestamp})

			if err != nil {
				return nil, err
			}

			if len(fetchedKlines) == 0 {
				break
			}

			if interval.Duration != loadInterval {

				fetchedKlines = convertKlinesToNewTimestamp(fetchedKlines, interval)
			}

			lastKlines = append(lastKlines, fetchedKlines...)

			if lastKlines[len(lastKlines)-1].PrevCloseCandleTimestamp == 0 {
				break
			}

			timestamp = lastKlines[len(lastKlines)-1].OpenTime

		}

	} else {

		databaseIn := candlescommon.Interval{Letter: interval.Letter, Duration: databaseInterval}

		max, min, err := IsAllCandlesLoaded(symbol, fmt.Sprintf("%d%s", databaseIn.Duration, databaseIn.Letter))

		if err != nil {
			log.Fatal(err.Error())
		}

		if timestamp > uint64(max) {
			FillDatabaseToLatestValues(symbol, databaseIn)
		}

		for len(lastKlines) < limit {

			fetchedKlines, err := getKlinesFromDatabase(symbol, databaseIn, timestamp, 1000)

			if err != nil {
				log.Println(err.Error())
				return nil, err
			}

			fetchedKlines = convertKlinesToNewTimestamp(fetchedKlines, interval)

			if len(fetchedKlines) == 0 && min != 0 {
				FillDatabaseWithPrevValues(symbol, databaseIn, 900)
				continue
			} else if len(fetchedKlines) == 0 {
				log.Println("break because no more data")
				break
			}

			lastKlines = append(lastKlines, fetchedKlines...)
			timestamp = lastKlines[len(lastKlines)-1].OpenTime

			if lastKlines[len(lastKlines)-1].PrevCloseCandleTimestamp == 0 {
				break
			}

		}

	}

	if len(lastKlines) > limit {
		lastKlines = lastKlines[:limit]
	}

	for i := 0; i < len(lastKlines)/2; i++ {
		j := len(lastKlines) - i - 1
		lastKlines[i], lastKlines[j] = lastKlines[j], lastKlines[i]
	}

	return lastKlines, nil
}

func SaveCandles(klines []candlescommon.KLine, interval candlescommon.Interval) {

	t := time.Now()

	stmt, err := database.DatabaseManager.Prepare(fmt.Sprintf(`INSERT INTO public.tran_candles_%d%s(symbol, "openTime", "closeTime", "prevCandle", "openPrice", "closePrice", "lowPrice", "highPrice", volume, "quoteVolume", "takerVolume", "takerQuoteVolume")
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);`, interval.Duration, interval.Letter))

	if err != nil {

		log.Fatal(err.Error())
	}

	for _, kline := range klines {

		if !kline.Closed {
			continue
		}

		_, err = stmt.Exec(kline.Symbol, kline.OpenTime, kline.CloseTime, kline.PrevCloseCandleTimestamp, kline.OpenPrice, kline.ClosePrice, kline.LowPrice, kline.HighPrice, kline.BaseVolume, kline.QuoteVolume, kline.TakerBuyBaseVolume, kline.TakerBuyQuoteVolume)

		if err != nil {

			for _, klinee := range klines {
				log.Println(klinee)
			}
			log.Fatal(err.Error(), kline)

		}

	}

	stmt.Close()

	log.Println(time.Since(t))

}
