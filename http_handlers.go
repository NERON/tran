package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/indicators"
	"github.com/NERON/tran/manager"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
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

	interval := candlescommon.IntervalFromStr(vars["interval"])

	for _, symbol := range symbols {

		rsiP := indicators.NewRSIMultiplePeriods(250)

		candles, _ := manager.GetLastKLines(symbol, interval, 100000)

		candlesOld, _ := manager.GetLastKLinesFromTimestamp(symbol, interval, candles[0].OpenTime, 2000)

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

				_, ok := RSIValMap[bestPeriod]

				if !ok {
					RSIValMap[bestPeriod] = make(map[string]int, 0)
				}

				RSIValMap[bestPeriod][fmt.Sprintf("%.1f", rsiVal)]++

			}

			rsiP.AddPoint(candle.ClosePrice)

		}

		seqStack := list.New()

		for i := 0; i < len(sequence); i++ {

			if sequence[i] == 0 {
				continue
			}

			for e := seqStack.Front(); e != nil && e.Value.(int) <= sequence[i]+1; e = seqStack.Front() {

				_, ok := transitionMap[e.Value.(int)]

				if !ok {

					transitionMap[e.Value.(int)] = make(map[int]int)
				}

				transitionMap[e.Value.(int)][sequence[i]]++

				if e.Value.(int) <= sequence[i] {
					seqStack.Remove(e)
				}

			}

			counter, _ := counterMap[sequence[i]]
			counter.Counter++

			counterMap[sequence[i]] = counter

			seqStack.PushFront(sequence[i])
		}

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
		PrevCandleClose uint64
	}

	vars := mux.Vars(r)

	intervalStr := vars["interval"]

	centralRSI, _ := strconv.ParseUint(vars["centralRSI"], 10, 64)

	if centralRSI == 0 {
		centralRSI = 20
	}

	endTimestamp := uint64(0)

	if len(r.URL.Query()["endTimestamp"]) > 0 {

		endTimestamp, _ = strconv.ParseUint(r.URL.Query()["endTimestamp"][0], 10, 64)
	}

	interval := candlescommon.IntervalFromStr(intervalStr)

	var candles []candlescommon.KLine

	var err error

	if endTimestamp > 0 {

		calcEnd := endTimestamp

		if interval.Letter == "h" {
			calcEnd = (endTimestamp/uint64(interval.Duration)/3600/1000 + 1) * uint64(interval.Duration) * 3600 * 1000

		} else if interval.Letter == "m" {
			calcEnd = (endTimestamp/uint64(interval.Duration)/60/1000 + 1) * uint64(interval.Duration) * 60 * 1000
		}

		candles, err = manager.GetLastKLinesFromTimestamp(vars["symbol"], interval, calcEnd, 1000)

	} else {
		candles, err = manager.GetLastKLines(vars["symbol"], interval, 1000)
	}

	if err != nil {
		log.Println(err.Error())
	}

	if len(candles) == 0 {
		log.Println("candles null")
		return
	}

	rsiP := indicators.NewRSIMultiplePeriods(250)

	candlesOld, err := manager.GetLastKLinesFromTimestamp(vars["symbol"], interval, candles[0].OpenTime, 500)

	if err != nil {
		log.Println(err.Error())
	}

	for _, candleOld := range candlesOld {

		rsiP.AddPoint(candleOld.ClosePrice)

	}

	updateCandles := make([]ChartUpdateCandle, 0)

	lowReverse := indicators.NewRSILowReverseIndicator()
	lowsMap := make(map[int]struct{})

	if len(candlesOld) > 0 {
		lowReverse.AddPoint(candlesOld[len(candlesOld)-1].LowPrice, 0)
	}

	for idx, candle := range candles {

		lowReverse.AddPoint(candle.LowPrice, 0)

		if lowReverse.IsPreviousLow() {
			lowsMap[idx-1] = struct{}{}
		} else if idx > 0 && candle.OpenPrice < candle.ClosePrice && candles[idx-1].LowPrice >= candle.LowPrice {
			lowsMap[idx] = struct{}{}
		}

	}

	//TODO: Get low reverse for last element
	if endTimestamp > 0 && candles[len(candles)-1].OpenTime == endTimestamp {
		log.Println("Remove last element")
		candles = candles[:len(candles)-1]
	}

	rsiRev := indicators.NewRSILowReverseIndicator()
	rsi := indicators.RSI{Period: 3}

	bestSequenceList := list.New()

	for idx, candle := range candles {

		rsiRev.AddPoint(candle.LowPrice, candle.ClosePrice)

		_, ok := lowsMap[idx]

		bestPeriod := 0

		if ok {

			bestPeriod, _ = rsiP.GetBestPeriod(candle.LowPrice, float64(centralRSI))

			for e := bestSequenceList.Front(); e != nil && e.Value.(int) <= bestPeriod; e = bestSequenceList.Front() {
				bestSequenceList.Remove(e)
			}

			bestSequenceList.PushFront(bestPeriod)

		}

		rsiP.AddPoint(candle.ClosePrice)

		updateCandles = append(updateCandles, ChartUpdateCandle{
			OpenTime:        candle.OpenTime,
			CloseTime:       candle.CloseTime,
			OpenPrice:       candle.OpenPrice,
			ClosePrice:      candle.ClosePrice,
			LowPrice:        candle.LowPrice,
			HighPrice:       candle.HighPrice,
			RSIValue:        0,
			RSIBestPeriod:   bestPeriod,
			IsRSIReverseLow: ok,
			PrevCandleClose: candle.PrevCloseCandleTimestamp,
		})

		rsi.AddPoint(candle.ClosePrice)

	}

	byte, err := json.Marshal(updateCandles)

	if err != nil {
		log.Println(err.Error())
	}

	w.Write(byte)

}

func SaveCandlesHandler(w http.ResponseWriter, r *http.Request) {

}
