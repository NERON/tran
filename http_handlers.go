package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/indicators"
	"github.com/NERON/tran/manager"
	"github.com/gorilla/mux"
	"gonum.org/v1/gonum/stat/combin"
	"log"
	"math"
	"net/http"
	"sort"
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

func GetTriplesHandler(w http.ResponseWriter, r *http.Request) {

	type Data struct {
		Symbol     string
		CentralRSI string
		Mode       string
	}

	vars := mux.Vars(r)

	TemplateManager.ExecuteTemplate(w, "triples.html", Data{Symbol: vars["symbol"], CentralRSI: vars["centralRSI"], Mode: vars["mode"]})
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
		Up              float64
		Down            float64
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

	candlesOld, err := manager.GetLastKLinesFromTimestamp(vars["symbol"], interval, candles[0].OpenTime, 3000)

	log.Println("candles length", len(candlesOld))
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
		} else if idx > 0 && candle.OpenPrice < candle.ClosePrice && candle.LowPrice < candle.OpenPrice {
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
		up := float64(0)
		down := float64(0)

		if ok {

			bestPeriod, _, _ = rsiP.GetBestPeriod(candle.LowPrice, float64(centralRSI))
			up, down, _ = rsiP.GetIntervalForPeriod(bestPeriod, float64(centralRSI))

			if bestPeriod > 2 || (bestPeriod == 2 && candle.LowPrice <= up) {

				for e := bestSequenceList.Front(); e != nil && e.Value.(int) <= bestPeriod; e = bestSequenceList.Front() {
					bestSequenceList.Remove(e)
				}

				bestSequenceList.PushFront(bestPeriod)

			} else {

				bestPeriod = 0
				up = 0
				down = 0
			}

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
			Up:              up,
			Down:            down,
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

		candles, errr := manager.KLineCacher.GetLatestKLines(vars["symbol"], interval)

		log.Println(candles)

		//candles, err := manager.GetLastKLines(vars["symbol"], interval, 1000)

		if errr != true {
			log.Println("Error")
			//w.Write([]byte(err.Error()))
			return
		}

		candlesGet, err := manager.GetLastKLinesFromTimestamp(vars["symbol"], interval, candles[0].OpenTime, 5000)

		if err != nil {

			w.Write([]byte(err.Error()))
			return
		}

		candles = append(candlesGet, candles...)

		candlesOld, err := manager.GetLastKLinesFromTimestamp(vars["symbol"], interval, candles[0].OpenTime, 5000)

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

		up, down, _ := rsiP.GetIntervalForPeriod(2, float64(centralRSI))

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

	intervalRange, _ := strconv.ParseUint(vars["mode"], 10, 64)

	var intervals []string

	if intervalRange == 0 {

		intervals = []string{
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

	} else {

		intervals = []string{
			"5m",
			"6m",
			"8m",
			"9m",
			"10m",
			"12m",
			"15m",
			"16m",
			"18m",
			"20m",
			"24m",
			"30m",
			"32m",
			"36m",
			"40m",
			"45m",
			"48m",
		}

	}

	type SequenceResult struct {
		Interval string
		Val      int
		Up       float64
		Down     float64
		Count    uint
	}

	type IntervalEnds struct {
		ID    string
		Value float64
		Type  int
	}

	segments := make([]IntervalEnds, 0)
	segmentsMap := make(map[string]SequenceResult, 0)

	var duration time.Duration

	for _, intervalStr := range intervals {

		interval := candlescommon.IntervalFromStr(intervalStr)

		var err error

		candles, ok := manager.KLineCacher.GetLatestKLines(vars["symbol"], interval)

		if ok {

			candlesGet, err := manager.GetLastKLinesFromTimestamp(vars["symbol"], interval, candles[0].OpenTime, 500)

			if err == nil {

				candles = append(candlesGet, candles...)

			}

		} else {

			candles, err = manager.GetLastKLines(vars["symbol"], interval, 500)
		}

		isCorrect := candlescommon.CheckCandles(candles)

		if !isCorrect {
			log.Fatal(candles)
		}

		if err != nil {

			w.Write([]byte(err.Error()))
			return
		}

		if len(candles) == 0 {
			w.Write([]byte("Data not exist"))
			return
		}

		bestSequenceList, lastUpdate, err := manager.GetPeriodsFromDatabase(vars["symbol"], intervalStr)

		if lastUpdate <= candles[0].OpenTime {
			bestSequenceList, lastUpdate, err = manager.GetSequncesWithUpdate(vars["symbol"], interval)
		}

		if err != nil {
			log.Fatal(err.Error())
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

		t := time.Now()

		for idx, candle := range candles {

			_, ok := lowsMap[idx]

			if ok && candle.OpenTime > lastUpdate {

				bestPeriod, _, centralPrice := rsiP.GetBestPeriod(candle.LowPrice, float64(centralRSI))

				periods := make([]int, 0)

				up, down, _ := rsiP.GetIntervalForPeriod(bestPeriod, float64(centralRSI))

				if bestPeriod > 2 || (bestPeriod == 2 && candle.LowPrice <= up) {

					if (centralPrice-candle.LowPrice)/(centralPrice-down) > 0.88 {
						periods = append(periods, bestPeriod+1)

					}

					periods = append(periods, bestPeriod)

					for _, period := range periods {

						lowCentral := true

						sequence := manager.SequenceValue{LowCentralPrice: lowCentral, Sequence: period, CentralPrice: centralPrice, Fictive: bestPeriod != period, Timestamp: candle.OpenTime, Central: centralPrice, Lower: candle.LowPrice, Down: down, Count: 1}

						if sequence.Fictive {
							sequence.Count -= 1
						}
						for e := bestSequenceList.Front(); e != nil && e.Value.(manager.SequenceValue).Sequence <= period; e = bestSequenceList.Front() {

							if sequence.Sequence == e.Value.(manager.SequenceValue).Sequence {
								sequence.Count += e.Value.(manager.SequenceValue).Count
							}

							bestSequenceList.Remove(e)
						}

						bestSequenceList.PushFront(sequence)
					}

				} else {

					bestPeriod = 0
					up = 0
					down = 0
				}

			}

			rsiP.AddPoint(candle.ClosePrice)

		}

		minValue := bestSequenceList.Front()

		if minValue != nil && minValue.Value.(manager.SequenceValue).Sequence != 2 {

			bestSequenceList.PushFront(manager.SequenceValue{LowCentralPrice: false, Sequence: 2, CentralPrice: 0})

		}

		previousAddedSeq := 0

		for e := bestSequenceList.Front(); e != nil; e = e.Next() {

			sequenceData := e.Value.(manager.SequenceValue)

			sign := ""

			if sequenceData.Fictive {
				sign = "*"
			}

			if sequenceData.Count > 1 {
				sign += "!"
			}

			if previousAddedSeq < sequenceData.Sequence {

				up, down, _ := rsiP.GetIntervalForPeriod(sequenceData.Sequence, float64(centralRSI))

				if up <= down || up <= 0 || down <= 0 {
					continue
				}

				segments = append(segments, IntervalEnds{ID: fmt.Sprintf("%s_%d%s", intervalStr, sequenceData.Sequence, sign), Value: up, Type: 0})
				segments = append(segments, IntervalEnds{ID: fmt.Sprintf("%s_%d%s", intervalStr, sequenceData.Sequence, sign), Value: down, Type: 1})
				segmentsMap[fmt.Sprintf("%s_%d%s", intervalStr, sequenceData.Sequence, sign)] = SequenceResult{Interval: intervalStr, Val: sequenceData.Sequence, Up: up, Down: down, Count: sequenceData.Count}
			}

			if sequenceData.LowCentralPrice == true && e.Next() != nil {
				sequenceData.LowCentralPrice = e.Next().Value.(manager.SequenceValue).Sequence > sequenceData.Sequence+1
			}

			if sequenceData.LowCentralPrice {

				sequenceData.Sequence += 1
				sequenceData.LowCentralPrice = false

				up, down, _ := rsiP.GetIntervalForPeriod(sequenceData.Sequence, float64(centralRSI))

				if up <= down || up <= 0 || down <= 0 {
					continue
				}

				segments = append(segments, IntervalEnds{ID: fmt.Sprintf("%s_%d%s", intervalStr, sequenceData.Sequence, sign), Value: up, Type: 0})
				segments = append(segments, IntervalEnds{ID: fmt.Sprintf("%s_%d%s", intervalStr, sequenceData.Sequence, sign), Value: down, Type: 1})

				segmentsMap[fmt.Sprintf("%s_%d%s", intervalStr, sequenceData.Sequence, sign)] = SequenceResult{Interval: intervalStr, Val: sequenceData.Sequence, Up: up, Down: down, Count: sequenceData.Count}

			}

			previousAddedSeq = sequenceData.Sequence
		}
		duration += time.Since(t)

	}

	log.Println(duration)

	type Res struct {
		Combination []string
		Up          float64
		Down        float64
		Percentage  float64
		HasRepeats  bool
	}

	sort.Slice(segments, func(i, j int) bool {

		if segments[i].Value == segments[j].Value {
			return segments[i].Type > segments[j].Type
		}

		return segments[i].Value > segments[j].Value
	})

	test := make([]Res, 0)

	t := time.Now()

	intersectionList := make([]string, 0)

	for _, end := range segments {

		if end.Type == 0 {
			intersectionList = append(intersectionList, end.ID)

		} else {

			index := 0

			for idx, val := range intersectionList {

				index = idx

				if val == end.ID {
					break
				}
			}

			if index >= len(intersectionList) {
				log.Println("Error", end.ID, intersectionList, segments)
				return
			}

			//remove data
			intersectionList[index] = intersectionList[len(intersectionList)-1]
			intersectionList = intersectionList[:len(intersectionList)-1]

			if len(intersectionList) >= 1 {

				//generate combinations
				gen := combin.NewCombinationGenerator(len(intersectionList), 1)

				for gen.Next() {

					combinations := gen.Combination(nil)

					up := 99999999999999999999999.0
					down := 0.0
					isRepeated := false

					combination := []string{intersectionList[combinations[0]], end.ID}

					for _, comb := range combination {

						val, _ := segmentsMap[comb]
						up = math.Min(up, val.Up)
						down = math.Max(down, val.Down)
						isRepeated = isRepeated || val.Count > 1
					}
					test = append(test, Res{combination, up, down, (down/up - 1) * 100, isRepeated})

				}

			}

		}
	}

	sort.Slice(test, func(i, j int) bool {

		if test[i].Down == test[j].Down {
			return test[i].Up > test[j].Up

		}
		return test[i].Down > test[j].Down
	})

	log.Println("intersection time", time.Since(t))

	exclude := make([]Res, 0)

	currentDownNoRepeats := 0.0
	currentDownRepeats := 0.0

	for _, val := range test {

		if val.Percentage < -3 {
			exclude = append(exclude, val)

		} else if val.HasRepeats && currentDownRepeats != val.Down {

			exclude = append(exclude, val)
			currentDownRepeats = val.Down

		} else if !val.HasRepeats && currentDownNoRepeats != val.Down {
			exclude = append(exclude, val)
			currentDownNoRepeats = val.Down
		}

	}

	sort.Slice(exclude, func(i, j int) bool {

		return exclude[i].Up > exclude[j].Up
	})

	byte, err := json.Marshal(exclude)

	if err != nil {
		log.Println(err.Error())
	}

	w.Write(byte)

}
