package candlescommon

import (
	"github.com/NERON/tran/database"
	"log"
	"math"
	"time"
)

type KLine struct {
	Symbol                   string
	OpenTime                 uint64
	CloseTime                uint64
	OpenPrice                float64
	ClosePrice               float64
	HighPrice                float64
	LowPrice                 float64
	BaseVolume               float64
	QuoteVolume              float64
	TakerBuyBaseVolume       float64
	TakerBuyQuoteVolume      float64
	PrevCloseCandleTimestamp uint64
	Closed                   bool
}

func GroupKline(klines []KLine, groupCount int) []KLine {

	newKlines := make([]KLine, 0)

	newKline := KLine{}

	prevClosed := uint64(0)

	for idx, kline := range klines {

		if idx%groupCount == 0 {

			if idx > 0 {
				prevClosed = newKline.CloseTime
				newKlines = append(newKlines, newKline)
			}

			newKline = kline
			newKline.PrevCloseCandleTimestamp = prevClosed

		} else {

			newKline.HighPrice = math.Max(newKline.HighPrice, kline.HighPrice)
			newKline.LowPrice = math.Min(newKline.LowPrice, kline.LowPrice)
			newKline.ClosePrice = kline.ClosePrice
			newKline.CloseTime = kline.CloseTime
			newKline.BaseVolume += kline.BaseVolume
			newKline.TakerBuyBaseVolume += kline.TakerBuyBaseVolume
			newKline.QuoteVolume += newKline.QuoteVolume
			newKline.TakerBuyQuoteVolume += kline.TakerBuyQuoteVolume
		}

	}

	return newKlines
}

type KLineManager struct{}

func (manager KLineManager) GetChartData(symbol string, interval time.Duration, startTimestamp uint64, endTimestamp uint64) ([]KLine, error) {

	klines := make([]KLine, 0)

	//get data from database
	rows, err := database.DatabaseManager.Query("")

	if err != nil {
		return nil, err
	}

	for rows.Next() {

		kline := KLine{}

		err = rows.Scan(&kline.Symbol,
			&kline.OpenTime,
			&kline.CloseTime,
			&kline.OpenPrice,
			&kline.ClosePrice,
			&kline.LowPrice,
			&kline.HighPrice,
			&kline.BaseVolume,
			&kline.QuoteVolume,
			&kline.TakerBuyBaseVolume,
			&kline.TakerBuyQuoteVolume,
			&kline.PrevCloseCandleTimestamp)

		if err != nil {

			rows.Close()
			return nil, err
		}

	}

	rows.Close()

	var prevKlineCloseTime uint64

	type Gap struct {
		From uint64
		To   uint64
	}

	gaps := make([]Gap, 0)

	//check data for correctness
	for _, kline := range klines {

		if prevKlineCloseTime > 0 && prevKlineCloseTime != prevKlineCloseTime {
			gaps = append(gaps, Gap{From: prevKlineCloseTime + 1, To: kline.OpenTime})
		}

		prevKlineCloseTime = kline.CloseTime
		klines = append(klines, kline)

	}

	return klines, nil

}

func SaveCandles(klines []KLine) {

	log.Println("Start saving")
	stmt, err := database.DatabaseManager.Prepare(`INSERT INTO public.tran_candles_1h(symbol, "openTime", "closeTime", "prevCandle", "openPrice", "closePrice", "lowPrice", "highPrice", volume, "quoteVolume", "takerVolume", "takerQuoteVolume")
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) ON CONFLICT DO NOTHING;`)

	if err != nil {

		log.Fatal(err.Error())
	}

	for _, kline := range klines {

		t := time.Now()
		if kline.PrevCloseCandleTimestamp == 0 || !kline.Closed {
			continue
		}

		_, err = stmt.Exec(kline.Symbol, kline.OpenTime, kline.CloseTime, kline.PrevCloseCandleTimestamp, kline.OpenPrice, kline.ClosePrice, kline.LowPrice, kline.HighPrice, kline.BaseVolume, kline.QuoteVolume, kline.TakerBuyBaseVolume, kline.TakerBuyQuoteVolume)

		if err != nil {

			log.Fatal(err.Error())
		}
		log.Println(time.Since(t))

	}

	stmt.Close()

	log.Println("saved")
}
