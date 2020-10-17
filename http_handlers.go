package main

import (
	"bytes"
	"encoding/json"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/database"
	"github.com/NERON/tran/indicators"
	"github.com/NERON/tran/providers"
	"github.com/gorilla/mux"
	"log"
	"math"
	"net/http"
	"time"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {

	type Data struct {
		Symbol    string
		Timeframe string
	}

	vars := mux.Vars(r)

	TemplateManager.ExecuteTemplate(w, "chartPage.html", Data{vars["symbol"], vars["interval"]})
}
func RSIJSONHandler(w http.ResponseWriter, r *http.Request) {

	WINDOW := 24

	candles := providers.GetKlines("ETHUSDT", "1h", 0, 0)

	rsi := indicators.RSI{Period: 14}

	RSIs := make([]float64, 0)

	for _, candle := range candles {

		rsi.AddPoint(candle.ClosePrice)
		rsiVal, ok := rsi.Calculate()

		if ok {
			RSIs = append(RSIs, math.Round(rsiVal*1000)/1000)
		}
	}

	currentWindow := make([][]float64, 2)
	RSIsWindowed := make([][][]float64, 0)

	i := 0

	for ; i < WINDOW && i < len(RSIs); i++ {
		currentWindow[0] = append(currentWindow[0], RSIs[i])
	}

	i++

	for ; i < len(RSIs); i++ {

		currentWindow[1] = make([]float64, 1)
		currentWindow[1][0] = RSIs[i]

		RSIsWindowed = append(RSIsWindowed, currentWindow)

		currentWindowNew := make([][]float64, 2)
		currentWindowNew[0] = append(currentWindowNew[0], currentWindow[0][1:]...)
		currentWindowNew[0] = append(currentWindowNew[0], RSIs[i])

		currentWindow = currentWindowNew

	}

	//output json
	byte, _ := json.Marshal(RSIsWindowed)

	w.Header().Add("Content-Disposition", "Attachment")

	http.ServeContent(w, r, "BTCUSDT.json", time.Now(), bytes.NewReader(byte))

}

func TestHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	transitionMap := make(map[int]map[int]int)

	counterMap := make(map[int]int)

	symbols := []string{"BTCUSDT", "ETHUSDT", "LTCUSDT"}

	for _, symbol := range symbols {

		rsiP := indicators.NewRSIMultiplePeriods(250)

		candles := providers.GetKlinesTest(symbol, vars["interval"], 0, 0, 20)

		candlesOld := providers.GetKlines(symbol, vars["interval"], 0, candles[0].OpenTime-1)

		for _, candleOld := range candlesOld {
			rsiP.AddPoint(candleOld.ClosePrice)

		}

		lowReverse := indicators.NewRSILowReverseIndicator()
		lowsMap := make(map[int]struct{})

		for idx, candle := range candles {

			lowReverse.AddPoint(candle.LowPrice, 0)

			if lowReverse.IsPreviousLow() {
				lowsMap[idx-1] = struct{}{}
			}

		}

		sequence := make([]int, 0)

		for idx, candle := range candles {

			_, ok := lowsMap[idx]

			if ok {

				bestPeriod := rsiP.GetBestPeriod(candle.LowPrice, 20)
				sequence = append(sequence, bestPeriod)
			}

			rsiP.AddPoint(candle.ClosePrice)

		}

		for i := 0; i < len(sequence); i++ {

			j := i + 1

			counterMap[sequence[i]]++

			for j < len(sequence) && sequence[i] > sequence[j] {
				j++
			}

			if j < len(sequence) {

				_, ok := transitionMap[sequence[i]]

				if !ok {
					transitionMap[sequence[i]] = make(map[int]int)
				}

				transitionMap[sequence[i]][sequence[j]]++
			}

		}

		time.Sleep(time.Second * 30)

	}

	candles := make([]candlescommon.KLine, 0)
	b, _ := json.Marshal(transitionMap)
	t, _ := json.Marshal(candles)
	c, _ := json.Marshal(counterMap)

	type Test struct {
		Data string
		Test string
		Stat string
	}

	err := TemplateManager.ExecuteTemplate(w, "RSIReverseStat.html", Test{string(b), string(t), string(c)})

	if err != nil {
		log.Println(err)
	}

}

func ChartUpdateHandler(w http.ResponseWriter, r *http.Request) {

	type ChartUpdateCandle struct {
		OpenTime        uint64
		CloseTime       uint64
		OpenPrice       float64
		ClosePrice      float64
		LowPrice        float64
		HighPrice       float64
		IsRSIReverseLow bool
		RSIValue        float64
		RSIBestPeriod   int
	}

	vars := mux.Vars(r)

	interval := vars["interval"]

	if interval == "2w" {
		vars["interval"] = "1w"
	}

	candles := providers.GetKlines(vars["symbol"], vars["interval"], 0, 0)

	rsiP := indicators.NewRSIMultiplePeriods(250)

	candlesOld := providers.GetKlines(vars["symbol"], vars["interval"], 0, candles[0].OpenTime-1)

	if interval == "2w" {
		candles = candlescommon.GroupKline(candles, 2)
	}

	for _, candleOld := range candlesOld {

		rsiP.AddPoint(candleOld.ClosePrice)

		log.Println(candleOld)

	}

	updateCandles := make([]ChartUpdateCandle, 0)

	lowReverse := indicators.NewRSILowReverseIndicator()
	lowsMap := make(map[int]struct{})

	for idx, candle := range candles {

		lowReverse.AddPoint(candle.LowPrice, 0)

		if lowReverse.IsPreviousLow() {
			lowsMap[idx-1] = struct{}{}
		}

	}

	rsiRev := indicators.NewRSILowReverseIndicator()
	rsi := indicators.RSI{Period: 3}

	for idx, candle := range candles {

		rsiRev.AddPoint(candle.LowPrice, candle.ClosePrice)

		_, ok := lowsMap[idx]

		bestPeriod := 0

		if ok {

			bestPeriod = rsiP.GetBestPeriod(candle.LowPrice, 20)

		}

		rsiP.AddPoint(candle.ClosePrice)

		calcRSI, _ := rsi.PredictForNextPoint(candle.LowPrice)

		updateCandles = append(updateCandles, ChartUpdateCandle{
			OpenTime:        candle.OpenTime,
			CloseTime:       candle.CloseTime,
			OpenPrice:       candle.OpenPrice,
			ClosePrice:      candle.ClosePrice,
			LowPrice:        candle.LowPrice,
			HighPrice:       candle.HighPrice,
			RSIValue:        calcRSI,
			RSIBestPeriod:   bestPeriod,
			IsRSIReverseLow: ok,
		})

		rsi.AddPoint(candle.ClosePrice)

	}

	byte, _ := json.Marshal(updateCandles)

	w.Write(byte)

}

func SaveCandlesHandler(w http.ResponseWriter, r *http.Request) {

	klines := providers.GetKlines("ETHUSDT", "1h", 0, 0)

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

}
