package manager

import (
	"container/list"
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/indicators"
	"log"
	"sync"
)

type rsiData struct {
	reverse            indicators.ReverseLowInterface
	rsiP               *indicators.RSIMultiplePeriods
	bestPeriods        *list.List
	lastInsertedCandle uint64
	mutex              *sync.Mutex
}

type SequenceValue struct {
	Sequence        int
	LowCentralPrice bool
	CentralPrice    float64
	Fictive         bool
	Timestamp       uint64
	Central         float64
	Lower           float64
	Down            float64
	Count           uint
}

type RSIPeriodManager struct {
	data       map[string]map[string]rsiData
	centralRSI float64
	outerMutex *sync.Mutex
}

func (r *RSIPeriodManager) fillData(data *rsiData, symbol string, interval candlescommon.Interval) {

	if data.lastInsertedCandle == 0 {

		candles, ok := KLineCacher.GetLatestKLines(symbol, interval)

		var err error

		if ok {

			candlesGet, err := GetLastKLinesFromTimestamp(symbol, interval, candles[0].OpenTime, 100000)

			if err == nil {

				candles = append(candlesGet, candles...)

			}

		} else {

			candles, err = GetLastKLines(symbol, interval, 1000)

		}

		isCorrect := candlescommon.CheckCandles(candles)

		if !isCorrect || err != nil {
			log.Fatal(candles)
		}

		if candles[len(candles)-1].Closed == false {
			candles = candles[:len(candles)-1]
		}

		candlesOld, err := GetLastKLinesFromTimestamp(symbol, interval, candles[0].OpenTime, 500)

		if err != nil {

			return
		}

		lowsMap := make(map[int]struct{})

		if len(candlesOld) > 0 {
			data.reverse.AddPoint(candlesOld[len(candlesOld)-1].LowPrice, 0)
		}

		for idx, candle := range candles {

			data.reverse.AddPoint(candle.LowPrice, 0)

			if data.reverse.IsPreviousLow() {
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

				bestPeriod, _, centralPrice := rsiP.GetBestPeriod(candle.LowPrice, r.centralRSI)

				periods := make([]int, 0)

				up, down, _ := rsiP.GetIntervalForPeriod(bestPeriod, r.centralRSI)

				if bestPeriod > 2 || (bestPeriod == 2 && candle.LowPrice <= up) {

					if (centralPrice-candle.LowPrice)/(centralPrice-down) > 0.88 {
						periods = append(periods, bestPeriod+1)

					}

					periods = append(periods, bestPeriod)

					for _, period := range periods {

						lowCentral := true

						sequence := SequenceValue{LowCentralPrice: lowCentral, Sequence: period, CentralPrice: centralPrice, Fictive: bestPeriod != period, Timestamp: candle.OpenTime, Central: centralPrice, Lower: candle.LowPrice, Down: down, Count: 1}

						for e := bestSequenceList.Front(); e != nil && e.Value.(SequenceValue).Sequence <= period; e = bestSequenceList.Front() {

							if sequence.Sequence == e.Value.(SequenceValue).Sequence {
								sequence.Count += e.Value.(SequenceValue).Count
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

		if minValue != nil && minValue.Value.(SequenceValue).Sequence != 2 {

			bestSequenceList.PushFront(SequenceValue{LowCentralPrice: false, Sequence: 2, CentralPrice: 0})

		}

	} else {

	}
}

func (r *RSIPeriodManager) GetBestPeriods(symbol string, interval candlescommon.Interval) *list.List {

	//lock to get rsi data
	r.outerMutex.Lock()

	symbolTimestamps, ok := r.data[symbol]

	//check for symbol
	if !ok {
		r.data[symbol] = make(map[string]rsiData, 0)
		symbolTimestamps = r.data[symbol]
	}

	innerData, ok := symbolTimestamps[fmt.Sprintf("%d%s", interval.Duration, interval.Letter)]

	//check for timestamp
	if !ok {

		symbolTimestamps[fmt.Sprintf("%d%s", interval.Duration, interval.Letter)] = rsiData{
			bestPeriods: list.New(),
			mutex:       &sync.Mutex{},
			rsiP:        indicators.NewRSIMultiplePeriods(250),
			reverse:     indicators.NewRSILowReverseIndicator(),
		}

		innerData = symbolTimestamps[fmt.Sprintf("%d%s", interval.Duration, interval.Letter)]
	}

	//lock inner data
	innerData.mutex.Lock()

	//fill data
	r.fillData(&innerData, symbol, interval)

	//unlock outer mutex
	r.outerMutex.Unlock()

	//get or generate data
	innerData.mutex.Unlock()

	return list.New()

}

func NewRSIPeriodManager(centralRSI float64) *RSIPeriodManager {
	return &RSIPeriodManager{data: make(map[string]map[string]rsiData), outerMutex: &sync.Mutex{}, centralRSI: centralRSI}
}
