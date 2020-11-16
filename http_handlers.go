package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/indicators"
	"github.com/NERON/tran/manager"
	"github.com/NERON/tran/providers"
	"github.com/gorilla/mux"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {

	type Data struct {
		Symbol     string
		Timeframe  string
		CentralRSI string
	}

	vars := mux.Vars(r)

	TemplateManager.ExecuteTemplate(w, "chartPage.html", Data{vars["symbol"], vars["interval"], vars["centralRSI"]})
}
func RSIJSONHandler(w http.ResponseWriter, r *http.Request) {

	WINDOW := 24

	candles := providers.GetKlines("ETHUSDT", "1h", 0, 0, false)

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

	type CounterVal struct {
		Counter   uint64
		LastKline candlescommon.KLine
	}

	counterMap := make(map[int]CounterVal)

	centralRSI, _ := strconv.ParseUint(vars["centralRSI"], 10, 64)

	symbols := []string{"BTCUSDT", "ETHUSDT", "LTCUSDT"}

	RSIValMap := make(map[int]map[string]int, 0)

	for _, symbol := range symbols {

		rsiP := indicators.NewRSIMultiplePeriods(250)

		candles := providers.GetKlinesTest(symbol, vars["interval"], 0, 0, 20)

		candlesOld := providers.GetKlines(symbol, vars["interval"], 0, candles[0].OpenTime-1, false)

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
		klinesSeq := make([]candlescommon.KLine, 0)

		for idx, candle := range candles {

			_, ok := lowsMap[idx]

			if ok {

				bestPeriod, rsiVal := rsiP.GetBestPeriod(candle.LowPrice, float64(centralRSI))
				sequence = append(sequence, bestPeriod)
				klinesSeq = append(klinesSeq, candle)

				if bestPeriod > 2 {

					_, ok := RSIValMap[bestPeriod]

					if !ok {
						RSIValMap[bestPeriod] = make(map[string]int, 0)
					}

					RSIValMap[bestPeriod][fmt.Sprintf("%.1f", rsiVal)]++
				}

			}

			rsiP.AddPoint(candle.ClosePrice)

		}

		for i := 0; i < len(sequence); i++ {

			j := i + 1

			counterV := counterMap[sequence[i]]
			counterV.Counter += 1
			counterV.LastKline = klinesSeq[i]
			counterMap[sequence[i]] = counterV

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

	b, _ := json.Marshal(transitionMap)
	t, err := json.Marshal(RSIValMap)
	c, _ := json.Marshal(counterMap)

	if err != nil {
		w.Write([]byte(err.Error()))
	}

	type Test struct {
		Data string
		Test string
		Stat string
	}

	err = TemplateManager.ExecuteTemplate(w, "RSIReverseStat.html", Test{string(b), string(t), string(c)})

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

	if interval == "3h" {
		vars["interval"] = "1h"
	}

	if interval == "45m" {
		vars["interval"] = "15m"
	}

	if interval == "288m" {
		vars["interval"] = "3m"
	}

	centralRSI, _ := strconv.ParseUint(vars["centralRSI"], 10, 64)

	if centralRSI == 0 {
		centralRSI = 20
	}

	endTimestamp := uint64(0)

	if len(r.URL.Query()["endTimestamp"]) > 0 {
		endTimestamp, _ = strconv.ParseUint(r.URL.Query()["endTimestamp"][0], 10, 64)
	}

	candles := providers.GetKlines(vars["symbol"], vars["interval"], 0, endTimestamp, false)

	if interval == "3h" {
		candles = candlescommon.HoursGroupKlineAsc(candles, 3)
	}

	if interval == "45m" {
		candles = candlescommon.MinutesGroupKlineAsc(candles, 15, 45)
	}
	if interval == "288m" {
		candles = candlescommon.MinutesGroupKlineAsc(candles, 3, 288)
	}

	rsiP := indicators.NewRSIMultiplePeriods(250)

	candlesOld := providers.GetKlines(vars["symbol"], vars["interval"], 0, candles[0].OpenTime-1, false)

	if interval == "3h" {
		candlesOld = candlescommon.HoursGroupKlineAsc(candlesOld, 3)
	}

	if interval == "45m" {
		candlesOld = candlescommon.MinutesGroupKlineAsc(candlesOld, 15, 45)
	}

	if interval == "288m" {
		candlesOld = candlescommon.MinutesGroupKlineAsc(candles, 3, 288)
	}

	if interval == "2w" {
		candles = candlescommon.GroupKline(candles, 2)
	}

	for _, candleOld := range candlesOld {

		rsiP.AddPoint(candleOld.ClosePrice)

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

			bestPeriod, _ = rsiP.GetBestPeriod(candle.LowPrice, float64(centralRSI))

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

	/*
		time, _ := strconv.ParseUint(vars["time"], 10, 64)

		var klines []candlescommon.KLine
		var err error

		if time == 0 {
			klines, err = providers.GetLastKlines("ETHUSDT", "3m")
		} else {

			direction, _ := strconv.ParseUint(vars["direction"], 10, 64)

			if direction == 0 {
				klines, err = providers.GetKlinesNew("ETHUSDT", "3m", providers.GetKlineRange{Direction: 0, FromTimestamp: time})
			} else {
				klines, err = providers.GetKlinesNew("ETHUSDT", "3m", providers.GetKlineRange{Direction: 1, FromTimestamp: time})
			}

		}

		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		klines = candlescommon.MinutesGroupKlineDesc(klines, 72)

		byte, _ := json.Marshal(klines)

		w.Write(byte)*/

	vars := mux.Vars(r)
	time, _ := strconv.ParseUint(vars["time"], 10, 64)

	var klines []candlescommon.KLine
	var err error

	if time == 0 {
		klines, err = manager.GetLastKLines("ETHUSDT", candlescommon.Interval{Letter: "h", Duration: 3}, 1000)
	} else {
		klines, err = manager.GetLastKLinesFromTimestamp("ETHUSDT", candlescommon.Interval{Letter: "d", Duration: 1}, time, 50)
	}

	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	byte, _ := json.Marshal(klines)

	w.Write(byte)
}
