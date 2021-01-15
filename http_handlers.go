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
	"sort"
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

				bestPeriod, rsiVal, _ := rsiP.GetBestPeriod(candle.LowPrice, float64(centralRSI))
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

			saveVal := 0

			for e := seqStack.Front(); e != nil && e.Value.(int) <= sequence[i]; e = seqStack.Front() {

				saveVal = e.Value.(int)

				_, ok := transitionMap[saveVal]

				if !ok {

					transitionMap[saveVal] = make(map[int]int)
				}

				transitionMap[saveVal][sequence[i]]++

				seqStack.Remove(e)

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

	bestSequenceList := list.New()

	for idx, candle := range candles {

		_, ok := lowsMap[idx]

		bestPeriod := 0

		if ok {

			bestPeriod, _, _ = rsiP.GetBestPeriod(candle.LowPrice, float64(centralRSI))

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
	}

	byte, err := json.Marshal(updateCandles)

	if err != nil {
		log.Println(err.Error())
	}

	w.Write(byte)

}
func GetIntervalHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	centralRSI, _ := strconv.ParseUint(vars["centralRSI"], 10, 64)

	if centralRSI == 0 {
		centralRSI = 20
	}

	intervals := []string{
		"6m",
		"8m",
		"9m",
		"10m",
		"12m",
		"15m",
		"16m",
		"20m",
		"24m",
		"30m",
		"32m",
		"36m",
		"40m",
		"45m",
		"48m",
		"1h",
		"72m",
		"80m",
		"90m",
		"96m",
		"2h",
		"144m",
		"160m",
		"3h",
		"4h",
		"288m",
		"6h",
		"8h",
		"12h",
	}

	type Result struct {
		Interval string
		Up       float64
		Down     float64
		Percent  float64
	}

	results := make([]Result, 0)

	for _, intervalStr := range intervals {

		interval := candlescommon.IntervalFromStr(intervalStr)

		candles, err := manager.GetLastKLines(vars["symbol"], interval, 1000)

		if err != nil {

			w.Write([]byte(err.Error()))
			return
		}

		candlesOld, err := manager.GetLastKLinesFromTimestamp(vars["symbol"], interval, candles[0].OpenTime, 500)

		if err != nil {

			w.Write([]byte(err.Error()))
			return
		}

		rsiP := indicators.NewRSIMultiplePeriods(250)

		for _, candleOld := range candlesOld {

			rsiP.AddPoint(candleOld.ClosePrice)

		}

		for _, candle := range candles {

			if candle.Closed {
				rsiP.AddPoint(candle.ClosePrice)
			}

		}

		up, down := rsiP.GetIntervalForPeriod(2, float64(centralRSI))

		results = append(results, Result{Interval: intervalStr, Up: up, Down: down, Percent: (down/up - 1) * 100})

	}

	byte, err := json.Marshal(results)

	if err != nil {
		log.Println(err.Error())
	}

	w.Write(byte)

}
func SaveCandlesHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	centralRSI, _ := strconv.ParseUint(vars["centralRSI"], 10, 64)

	if centralRSI == 0 {
		centralRSI = 20
	}

	intervals := []string{
		"1h",
		"72m",
		"80m",
		"90m",
		"96m",
		"2h",
		"144m",
	}

	type SequenceValue struct {
		Sequence        int
		LowCentralPrice bool
		CentralPrice    float64
	}

	type SequenceResult struct {
		Interval string
		Val      SequenceValue
		Up       float64
		Down     float64
	}

	type IntervalEnds struct {
		ID    string
		Value float64
		Type  int
	}

	results := make([]SequenceResult, 0)

	segments := make([]IntervalEnds, 0)

	for _, intervalStr := range intervals {

		interval := candlescommon.IntervalFromStr(intervalStr)

		candles, err := manager.GetLastKLines(vars["symbol"], interval, 1000)

		if err != nil {

			w.Write([]byte(err.Error()))
			return
		}

		if len(candles) == 0 {
			w.Write([]byte("Data not exist"))
			return
		}

		if candles[len(candles)-1].Closed == false {
			candles = candles[:len(candles)-1]
		}

		candlesOld, err := manager.GetLastKLinesFromTimestamp(vars["symbol"], interval, candles[0].OpenTime, 500)

		if err != nil {

			w.Write([]byte(err.Error()))
			return
		}

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

		rsiP := indicators.NewRSIMultiplePeriods(250)

		for _, candleOld := range candlesOld {

			rsiP.AddPoint(candleOld.ClosePrice)

		}

		bestSequenceList := list.New()

		for idx, candle := range candles {

			_, ok := lowsMap[idx]

			if ok {

				bestPeriod, _, centralPrice := rsiP.GetBestPeriod(candle.LowPrice, float64(centralRSI))

				lowCentral := candle.LowPrice <= centralPrice

				for e := bestSequenceList.Front(); e != nil && e.Value.(SequenceValue).Sequence <= bestPeriod; e = bestSequenceList.Front() {

					if e.Value.(SequenceValue).Sequence == bestPeriod {
						lowCentral = lowCentral || e.Value.(SequenceValue).LowCentralPrice
					}

					bestSequenceList.Remove(e)
				}

				bestSequenceList.PushFront(SequenceValue{LowCentralPrice: lowCentral, Sequence: bestPeriod, CentralPrice: centralPrice})

			}

			rsiP.AddPoint(candle.ClosePrice)

		}

		previousAddedSeq := 0

		for e := bestSequenceList.Front(); e != nil; e = e.Next() {

			sequenceData := e.Value.(SequenceValue)

			if previousAddedSeq != sequenceData.Sequence {

				up, down := rsiP.GetIntervalForPeriod(sequenceData.Sequence, float64(centralRSI))
				segments = append(segments, IntervalEnds{ID: fmt.Sprintf("%s_%d", intervalStr, sequenceData.Sequence), Value: up, Type: 0})
				segments = append(segments, IntervalEnds{ID: fmt.Sprintf("%s_%d", intervalStr, sequenceData.Sequence), Value: down, Type: 1})
				results = append(results, SequenceResult{Interval: intervalStr, Val: sequenceData, Up: up, Down: down})
				previousAddedSeq = sequenceData.Sequence
			}

			if sequenceData.LowCentralPrice && previousAddedSeq != sequenceData.Sequence+1 {
				sequenceData.Sequence += 1
				sequenceData.LowCentralPrice = false
				up, down := rsiP.GetIntervalForPeriod(sequenceData.Sequence, float64(centralRSI))

				segments = append(segments, IntervalEnds{ID: fmt.Sprintf("%s_%d", intervalStr, sequenceData.Sequence), Value: up, Type: 0})
				segments = append(segments, IntervalEnds{ID: fmt.Sprintf("%s_%d", intervalStr, sequenceData.Sequence), Value: down, Type: 1})

				results = append(results, SequenceResult{Interval: intervalStr, Val: sequenceData, Up: up, Down: down})

				previousAddedSeq = sequenceData.Sequence
			}
		}

	}

	sort.Slice(segments, func(i, j int) bool {

		if segments[i].Value == segments[j].Value {
			return segments[i].Type < segments[j].Type
		}

		return segments[i].Value > segments[j].Value
	})

	byte, err := json.Marshal(segments)

	if err != nil {
		log.Println(err.Error())
	}

	w.Write(byte)

}
