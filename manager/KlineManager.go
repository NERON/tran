package manager

import (
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/database"
	"github.com/NERON/tran/providers"
	"log"
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

func GetLastKLines(symbol string, interval string, limit int) ([]uint64, error) {

	fetchedKlines := providers.GetKlines(symbol, interval, 0, 0, true)

	if len(fetchedKlines) >= limit {
		//return fetchedKlines[:limit], nil
	}

	fetchedKlines = fetchedKlines[:500]

	rows, err := database.DatabaseManager.Query(fmt.Sprintf(`SELECT symbol, "openTime", "closeTime", "prevCandle", "openPrice", "closePrice", "lowPrice", "highPrice"
	FROM public.tran_candles_%s WHERE symbol = $1 AND "openTime" <= $2 ORDER BY "openTime" DESC LIMIT %d`, interval, limit-len(fetchedKlines)), symbol, fetchedKlines[len(fetchedKlines)-1].OpenTime)

	if err != nil {
		return nil, err
	}

	databaseCandles := make([]candlescommon.KLine, 0)

	var candleClose = uint64(0)
	var prevOpenTime = uint64(0)

	var gaps = make([]uint64, 0)

	for rows.Next() {

		kline := candlescommon.KLine{}

		err = rows.Scan(&kline.Symbol, &kline.OpenTime, &kline.CloseTime, &kline.PrevCloseCandleTimestamp, &kline.OpenPrice, &kline.ClosePrice, &kline.LowPrice, &kline.HighPrice)

		if err != nil {

			rows.Close()
			return nil, err
		}

		if candleClose > 0 && kline.CloseTime != candleClose {
			gaps = append(gaps, prevOpenTime)
		}

		candleClose = kline.PrevCloseCandleTimestamp
		prevOpenTime = kline.OpenTime

		databaseCandles = append(databaseCandles, kline)

	}

	rows.Close()

	return gaps, nil

}

func SaveCandles(klines []candlescommon.KLine) {

	t := time.Now()

	stmt, err := database.DatabaseManager.Prepare(`INSERT INTO public.tran_candles_1h(symbol, "openTime", "closeTime", "prevCandle", "openPrice", "closePrice", "lowPrice", "highPrice", volume, "quoteVolume", "takerVolume", "takerQuoteVolume")
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) ON CONFLICT DO NOTHING;`)

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
