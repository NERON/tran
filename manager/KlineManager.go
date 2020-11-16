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

func getClosestInterval(interval string) (string, time.Duration) {

	if interval == "1h" {
		return "1h", time.Hour
	} else if interval == "2h" {
		return "1h", 2 * time.Hour
	}

	return "", 1000000 * time.Hour
}

type DatabaseGap struct {
	From uint64
	To   uint64
}

func getKlinesFromDatabase(symbol string, interval string, endTimestamp uint64, limit int) ([]candlescommon.KLine, []DatabaseGap, error) {

	rows, err := database.DatabaseManager.Query(fmt.Sprintf(`SELECT symbol, "openTime", "closeTime", "prevCandle", "openPrice", "closePrice", "lowPrice", "highPrice"
	FROM public.tran_candles_%s WHERE symbol = $1 AND "openTime" <= $2 ORDER BY "openTime" DESC LIMIT %d`, interval, limit), symbol, endTimestamp)

	if err != nil {
		return nil, nil, err
	}

	databaseCandles := make([]candlescommon.KLine, 0)

	var candleClose = uint64(0)
	var prevOpenTime = uint64(0)

	var gaps = make([]DatabaseGap, 0)

	for rows.Next() {

		kline := candlescommon.KLine{}

		err = rows.Scan(&kline.Symbol, &kline.OpenTime, &kline.CloseTime, &kline.PrevCloseCandleTimestamp, &kline.OpenPrice, &kline.ClosePrice, &kline.LowPrice, &kline.HighPrice)

		if err != nil {

			rows.Close()
			return nil, nil, err
		}

		if candleClose > 0 && kline.CloseTime != candleClose {
			gaps = append(gaps, DatabaseGap{prevOpenTime, kline.OpenTime})
		}

		candleClose = kline.PrevCloseCandleTimestamp
		prevOpenTime = kline.OpenTime

		databaseCandles = append(databaseCandles, kline)

	}

	rows.Close()

	return databaseCandles, gaps, nil

}

func GetLastKLines(symbol string, interval candlescommon.Interval, limit int) ([]candlescommon.KLine, error) {

	loadInterval := GetOptimalLoadTimeframe(interval)

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

	if interval.Letter == "d" || interval.Letter == "M" || interval.Letter == "w" {

		for len(lastKlines) < limit {

			fetchedKlines, err := providers.GetKlinesNew(symbol, fmt.Sprintf("%d%s", loadInterval, interval.Letter), providers.GetKlineRange{Direction: 0, FromTimestamp: lastKlines[len(lastKlines)-1].OpenTime})

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
