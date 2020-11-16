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

	rows, err := database.DatabaseManager.Query(fmt.Sprintf(`SELECT symbol, "openTime", "closeTime", "prevCandle", "openPrice", "closePrice", "lowPrice", "highPrice"
	FROM public.tran_candles_%d%s WHERE symbol = $1 AND "openTime" < $2 ORDER BY "openTime" DESC LIMIT %d`, interval.Duration, interval.Letter, limit), symbol, endTimestamp)

	if err != nil {
		return nil, err
	}

	databaseCandles := make([]candlescommon.KLine, 0)

	prevCandleClose := uint64(0)

	for rows.Next() {

		kline := candlescommon.KLine{}

		err = rows.Scan(&kline.Symbol, &kline.OpenTime, &kline.CloseTime, &kline.PrevCloseCandleTimestamp, &kline.OpenPrice, &kline.ClosePrice, &kline.LowPrice, &kline.HighPrice)

		if err != nil {

			rows.Close()
			return nil, err
		}

		if prevCandleClose > 0 && prevCandleClose != kline.CloseTime {
			rows.Close()
			return nil, errors.New("gap found")
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
		klines = candlescommon.HoursGroupKlineDesc(klines, uint64(interval.Duration))
	} else if interval.Letter == "m" {
		klines = candlescommon.MinutesGroupKlineDesc(klines, uint64(interval.Duration))
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

			if lastKlines[len(lastKlines)-1].PrevCloseCandleTimestamp == math.MaxUint64 {
				break
			}
		}

	} else {

		databaseIn := candlescommon.Interval{Letter: interval.Letter, Duration: databaseInterval}

		for len(lastKlines) < limit {

			fetchedKlines, err := getKlinesFromDatabase(symbol, databaseIn, lastKlines[len(lastKlines)-1].OpenTime, 1000)

			if err != nil {
				return nil, err
			}

			if len(fetchedKlines) == 0 {
				FillDatabaseToLatestValues(symbol, databaseIn)
				FillDatabaseWithPrevValues(symbol, databaseIn, 1000)
				continue
			}

			fetchedKlines = convertKlinesToNewTimestamp(fetchedKlines, interval)

			log.Println("Loading", len(lastKlines))

			lastKlines = append(lastKlines, fetchedKlines...)
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
func GetLastKLinesFromTimestamp(symbol string, interval candlescommon.Interval, timestamp uint64, limit int) ([]candlescommon.KLine, error) {

	loadInterval := GetOptimalLoadTimeframe(interval)

	if loadInterval == 0 {
		return nil, errors.New("can't found optimal timeframe")
	}

	lastKlines := make([]candlescommon.KLine, 0)

	if interval.Letter == "d" || interval.Letter == "M" || interval.Letter == "w" {

		for len(lastKlines) < limit {

			fetchedKlines, err := providers.GetKlinesNew(symbol, fmt.Sprintf("%d%s", loadInterval, interval.Letter), providers.GetKlineRange{Direction: 0, FromTimestamp: timestamp})

			if err != nil {
				return nil, err
			}

			if len(fetchedKlines) == 0 {
				break
			}

			lastKlines = append(lastKlines, fetchedKlines...)

			if lastKlines[len(lastKlines)-1].PrevCloseCandleTimestamp == math.MaxUint64 {
				break
			}

			timestamp = lastKlines[len(lastKlines)-1].OpenTime
		}

	} else {

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
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) ON CONFLICT DO NOTHING;`, interval.Duration, interval.Letter))

	if err != nil {

		log.Fatal(err.Error())
	}

	for _, kline := range klines {

		if kline.PrevCloseCandleTimestamp == 0 || !kline.Closed {
			continue
		}

		_, err = stmt.Exec(kline.Symbol, kline.OpenTime, kline.CloseTime, kline.PrevCloseCandleTimestamp, kline.OpenPrice, kline.ClosePrice, kline.LowPrice, kline.HighPrice, kline.BaseVolume, kline.QuoteVolume, kline.TakerBuyBaseVolume, kline.TakerBuyQuoteVolume)

		if err != nil {

			log.Fatal(err.Error())
		}

	}

	stmt.Close()

	log.Println(time.Since(t))

}
