package manager

import (
	"errors"
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/database"
	"github.com/NERON/tran/providers"
	"log"
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
func convertKlinesToNewTimestamp(klines []candlescommon.KLine, interval candlescommon.Interval) []candlescommon.KLine {

	if interval.Letter == "h" {
		klines = candlescommon.HoursGroupKlineDesc(klines, uint64(interval.Duration), true)
	} else if interval.Letter == "m" {
		klines = candlescommon.MinutesGroupKlineDesc(klines, uint64(interval.Duration), true)
	}

	return klines
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

func Test() {

	klines, err := providers.GetLastKlines("ETHUSDT", "1m")

	if err != nil {
		log.Fatal(err.Error())
	}

	klines = candlescommon.MinutesGroupKlineDesc(klines, 5, true)

	for {

		oklines, err := providers.GetKlinesNew("ETHUSDT", "1m", providers.GetKlineRange{Direction: 0, FromTimestamp: klines[len(klines)-1].OpenTime})

		if err != nil {
			log.Fatal(err.Error())
		}
		oklines = candlescommon.MinutesGroupKlineDesc(oklines, 5, true)
		klines = append(klines, oklines...)

		prevClose := uint64(0)

		for i := 0; i < len(klines); i++ {

			if prevClose > 0 && klines[i].CloseTime != prevClose {

				for j := 0; j <= i; j++ {
					log.Println(j, klines[j])
				}
				log.Fatal("END")
			}
			prevClose = klines[i].PrevCloseCandleTimestamp

		}

		if len(oklines) == 0 || oklines[len(oklines)-1].PrevCloseCandleTimestamp == 0 {
			break
		}

	}

	interval := candlescommon.Interval{Duration: 5, Letter: "m"}

	for i := 0; i < len(klines); i++ {

		if klines[i].OpenTime%uint64(interval.Duration*60*1000) != 0 {
			log.Println("Wrong open value", klines[i])

		}

		if klines[i].CloseTime-klines[i].OpenTime+1 != uint64(interval.Duration*60*1000) {
			//log.Println("Wrong close value", klines[i])

		}

		if klines[i].CloseTime < klines[i].OpenTime {
			//log.Println("close time fucked", klines[i])

		}

		if klines[i].PrevCloseCandleTimestamp != 0 && (klines[i].PrevCloseCandleTimestamp+1)%uint64(interval.Duration*60*1000) != 0 {
			//log.Println("Wrong prev close value", klines[i])

		}
	}

}
