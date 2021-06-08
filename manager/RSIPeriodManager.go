package manager

import (
	"container/list"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/database"
	"github.com/NERON/tran/indicators"
	"log"
	"math"
	"sort"
)

var errWrongSavedTimestamp = errors.New("candle that was used previously are missed")

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

func GetPeriodsFromDatabase(symbol string, interval string, timestamp int64) (*list.List, uint64, *indicators.RSIMultiplePeriods, error) {

	var listJSon string
	var RSIJSon string
	var lastUpdate uint64

	err := database.DatabaseManager.QueryRow(`SELECT  list,"lastUpdate","lastRSI" FROM public."tran_bestPeriodsList" WHERE symbol=$1 AND interval=$2 AND "lastUpdate" < $3 ORDER BY "lastUpdate" DESC LIMIT 1;`, symbol, interval, timestamp).Scan(&listJSon, &lastUpdate, &RSIJSon)

	if err != nil && err != sql.ErrNoRows {
		return nil, 0, nil, err
	}

	if err == sql.ErrNoRows {
		return list.New(), 0, nil, nil
	}

	var sequenceArray []SequenceValue

	err = json.Unmarshal([]byte(listJSon), &sequenceArray)

	if err != nil {
		return nil, 0, nil, err
	}

	sequenceList := list.New()

	for _, val := range sequenceArray {
		sequenceList.PushBack(val)
	}

	var RSI indicators.RSIMultiplePeriods

	err = json.Unmarshal([]byte(RSIJSon), &RSI)

	if err != nil {
		return nil, 0, nil, err
	}

	return sequenceList, lastUpdate, &RSI, nil
}
func generateMapLows(lowReverse indicators.ReverseLowInterface, candles []candlescommon.KLine) map[int]struct{} {

	lowsMap := make(map[int]struct{})

	for idx, candle := range candles {

		lowReverse.AddPoint(candle.LowPrice, 0)

		if lowReverse.IsPreviousLow() {

			lowsMap[idx-1] = struct{}{}

		} else if idx > 0 && candle.OpenPrice < candle.ClosePrice && candles[idx-1].LowPrice >= candle.LowPrice {
			lowsMap[idx] = struct{}{}
		}
	}

	return lowsMap

}
func GetSequncesWithUpdate(symbol string, interval candlescommon.Interval, timestamp int64) (*list.List, uint64, *indicators.RSIMultiplePeriods, error) {

	prevCandle := candlescommon.KLine{}

	done := false
	newEndTimestamp := uint64(0)

	lastSavedSequences, lastKlineTimestamp, _, err := GetPeriodsFromDatabase(symbol, fmt.Sprintf("%d%s", interval.Duration, interval.Letter), timestamp)

	centralRSI := 15
	if err != nil {
		return nil, 0, nil, err
	}

	if lastSavedSequences == nil {
		lastSavedSequences = list.New()
	}

	commonBestSequenceList := list.New()

	var LastRSI *indicators.RSIMultiplePeriods

	for {

		var candles []candlescommon.KLine
		var err error

		//if no previous data get last klines, else try to find
		if prevCandle.OpenTime == 0 {

			if timestamp == math.MaxInt64 {
				//get last candles
				candles, err = GetLastKLines(symbol, interval, 1000)
			} else {

				candles, err = GetLastKLinesFromTimestamp(symbol, interval, uint64(timestamp), 1000)

			}

			//truncate unclosed candle, it can't be used for count
			if len(candles) > 0 {
				candles = candles[:len(candles)-1]
			}

		} else {

			candles, err = GetLastKLinesFromTimestamp(symbol, interval, prevCandle.OpenTime, 1000)
		}

		//return if error found
		if err != nil {
			return nil, 0, nil, err
		}

		//don't do anything if no candles
		if len(candles) == 0 {
			return lastSavedSequences, lastKlineTimestamp, nil, nil
		}

		//check if receive more than inserted in database
		if lastKlineTimestamp > 0 && candles[0].OpenTime < lastKlineTimestamp {

			//find position
			idx := sort.Search(len(candles), func(i int) bool {
				return candles[i].OpenTime >= lastKlineTimestamp
			})

			//check if it's true position
			if idx < len(candles) && candles[idx].OpenTime == lastKlineTimestamp {

				//start from candles that a not counted
				candles = candles[idx+1:]

				//if no candles left go break the cycle
				if len(candles) == 0 {
					break
				}

				done = true

			} else {

				return nil, 0, nil, errWrongSavedTimestamp
			}
		}

		//get old candles data
		candlesOld, err := GetLastKLinesFromTimestamp(symbol, interval, candles[0].OpenTime, 500)

		//check for errors
		if err != nil {
			return nil, 0, nil, err
		}

		//initialize Low reverse counter for calculating potential low points
		reverseLow := indicators.NewRSILowReverseIndicator()

		//if old candles is present insert last candle for checking is first candle in position
		if len(candlesOld) > 0 {
			reverseLow.AddPoint(candlesOld[len(candlesOld)-1].LowPrice, 0)
		}

		//check if previous candle present, if true we should append it, for calculating value for last candle
		if prevCandle.OpenTime > 0 {
			candles = append(candles, prevCandle)
		}

		//generate map with indexes of all potential lows
		lowsMap := generateMapLows(reverseLow, candles)

		//remove previously inserted last candle
		if prevCandle.OpenTime > 0 {
			candles = candles[:len(candles)-1]

		} else {

			//remove pre-last element because it should be only used for pre-counts
			delete(lowsMap, len(candles)-1)

			//remove element
			candles = candles[:len(candles)-1]

			//set new timestamp
			if len(candles) > 0 {

				newEndTimestamp = candles[len(candles)-1].OpenTime
			}

		}

		//start calculating
		rsiP := indicators.NewRSIMultiplePeriods(250)

		//first insert all old candles
		for _, candleOld := range candlesOld {

			rsiP.AddPoint(candleOld.ClosePrice)

		}

		bestSequenceList := list.New()

		for idx, candle := range candles {

			_, ok := lowsMap[idx]

			if ok {

				bestPeriod, _, centralPrice := rsiP.GetBestPeriod(candle.LowPrice, float64(centralRSI))

				periods := make([]int, 0)

				up, down, _ := rsiP.GetIntervalForPeriod(bestPeriod, float64(centralRSI))

				if bestPeriod > 2 || (bestPeriod == 2 && candle.LowPrice <= up) {

					periods = append(periods, bestPeriod)

					for _, period := range periods {

						sequence := SequenceValue{LowCentralPrice: true, Sequence: period, CentralPrice: centralPrice, Fictive: bestPeriod != period, Timestamp: candle.OpenTime, Central: centralPrice, Lower: candle.LowPrice, Down: down, Count: 1}

						if sequence.Fictive {
							sequence.Count -= 1
						}
						for e := bestSequenceList.Front(); e != nil && e.Value.(SequenceValue).Sequence <= period; e = bestSequenceList.Front() {

							if sequence.Sequence == e.Value.(SequenceValue).Sequence || sequence.Sequence == e.Value.(SequenceValue).Sequence+1 {
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

		if prevCandle.OpenTime == 0 {
			LastRSI = rsiP
		}

		maxValue := commonBestSequenceList.Back()

		for e := bestSequenceList.Front(); e != nil; e = e.Next() {

			if maxValue != nil {

				if maxValue.Value.(SequenceValue).Sequence == e.Value.(SequenceValue).Sequence || maxValue.Value.(SequenceValue).Sequence == e.Value.(SequenceValue).Sequence+1 {

					val := maxValue.Value.(SequenceValue)
					val.Count += e.Value.(SequenceValue).Count

					log.Println("Added Old Data: ", interval, e.Value, maxValue, val)
					commonBestSequenceList.Remove(maxValue)
					commonBestSequenceList.PushBack(val)

				} else if maxValue.Value.(SequenceValue).Sequence < e.Value.(SequenceValue).Sequence {

					log.Println("Added Old Data Without Merge: ", interval, e.Value, maxValue)
					commonBestSequenceList.PushBack(e.Value)

				}

			} else {

				commonBestSequenceList.PushBack(e.Value)
			}

			maxValue = commonBestSequenceList.Back()
		}

		if len(candles) == 0 || candles[0].PrevCloseCandleTimestamp == 0 || done {
			break
		}

		prevCandle = candles[0]

		log.Println(symbol, interval, prevCandle.OpenTime, lastKlineTimestamp, timestamp)
	}

	if newEndTimestamp > lastKlineTimestamp {

		//Merge Sequences
		maxValue := commonBestSequenceList.Back()

		for e := lastSavedSequences.Front(); e != nil; e = e.Next() {

			if maxValue != nil {

				if maxValue.Value.(SequenceValue).Sequence == e.Value.(SequenceValue).Sequence || maxValue.Value.(SequenceValue).Sequence == e.Value.(SequenceValue).Sequence+1 {

					val := maxValue.Value.(SequenceValue)
					val.Count += e.Value.(SequenceValue).Count

					log.Println("Merge Old Data: ", interval, e.Value, maxValue, val)

					commonBestSequenceList.Remove(maxValue)
					commonBestSequenceList.PushBack(val)

				} else if maxValue.Value.(SequenceValue).Sequence < e.Value.(SequenceValue).Sequence {

					log.Println("Global Merge: ", interval, e.Value, maxValue)

					commonBestSequenceList.PushBack(e.Value)

				}

			} else {

				commonBestSequenceList.PushBack(e.Value)
			}

			maxValue = commonBestSequenceList.Back()
		}

		var sequences = make([]SequenceValue, 0)

		for e := commonBestSequenceList.Front(); e != nil; e = e.Next() {
			sequences = append(sequences, e.Value.(SequenceValue))
		}

		js, err := json.Marshal(sequences)

		if err != nil {

			return nil, 0, nil, err
		}

		lastRSIJSON, err := json.Marshal(LastRSI)

		if err != nil {

			return nil, 0, nil, err
		}

		_, err = database.DatabaseManager.Exec(`INSERT INTO public."tran_bestPeriodsList"(symbol, "interval", "list","lastUpdate","lastRSI") VALUES ($1, $2, $3,$4,$5);`, symbol, fmt.Sprintf("%d%s", interval.Duration, interval.Letter), js, newEndTimestamp, lastRSIJSON)

		if err != nil {

			return nil, 0, nil, err
		}
	}

	return commonBestSequenceList, newEndTimestamp, LastRSI, nil
}
