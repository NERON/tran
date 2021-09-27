package manager

import (
	"container/list"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/indicators"
	"github.com/NERON/tran/providers"
	"log"
	"math"
)

type KLineManageData struct {
	LastCandleOpenTime uint64
	ClosePrice         float64
	MinPrice           float64
	CurrentPeriod      int
	Timestamps         []uint64
}

type SequenceData struct {
	Period        int
	HasFullFilled bool
	Timestamps    []uint64
}

type RSIIntervalManager struct {
	Timeframe int

	prevCandle       KLineManageData
	rsi              *indicators.RSIMultiplePeriods
	bestSequenceList *list.List
	PeriodCount      map[int]uint64
}

func (r *RSIIntervalManager) AddMinuteCandle(kline candlescommon.KLine, isCounting bool) {

	val1 := r.prevCandle.LastCandleOpenTime / uint64(r.Timeframe) / 60 / 1000
	val2 := kline.OpenTime / uint64(r.Timeframe) / 60 / 1000

	if val1 > 0 && val1 < val2 {
		r.rsi.AddPoint(r.prevCandle.ClosePrice)
		r.PeriodCount[r.prevCandle.CurrentPeriod] = r.PeriodCount[r.prevCandle.CurrentPeriod] + 1
		r.prevCandle.MinPrice = math.MaxFloat64
		r.prevCandle.CurrentPeriod = 0
		r.prevCandle.Timestamps = nil
	}

	if isCounting {

		period, _, _ := r.rsi.GetBestPeriod(kline.LowPrice, 15)
		up, down, _ := r.rsi.GetIntervalForPeriod(period, 15)

		fillPercentage := (up-kline.LowPrice)/(up-down)*100 > 95

		if period >= 2 && kline.LowPrice <= up && kline.LowPrice >= down && r.prevCandle.MinPrice >= kline.LowPrice {

			sequence := SequenceData{Period: period}

			for e := r.bestSequenceList.Front(); e != nil && e.Value.(SequenceData).Period <= period; e = r.bestSequenceList.Front() {

				if e.Value.(SequenceData).Period == period {

					sequence = e.Value.(SequenceData)

				}

				r.bestSequenceList.Remove(e)
			}

			sequence.HasFullFilled = fillPercentage || sequence.HasFullFilled

			if r.prevCandle.CurrentPeriod+1 == period {

				sequence.Timestamps = append(sequence.Timestamps, r.prevCandle.Timestamps...)
				r.prevCandle.CurrentPeriod = period
				r.prevCandle.Timestamps = append([]uint64{}, kline.OpenTime)

			} else if period == r.prevCandle.CurrentPeriod {

				r.prevCandle.Timestamps = append(r.prevCandle.Timestamps, kline.OpenTime)

			} else {

				r.prevCandle.CurrentPeriod = period
				r.prevCandle.Timestamps = append([]uint64{}, kline.OpenTime)

			}

			sequence.Timestamps = append(sequence.Timestamps, kline.OpenTime)

			r.bestSequenceList.PushFront(sequence)
			r.prevCandle.MinPrice = math.Min(kline.LowPrice, r.prevCandle.MinPrice)

		}

	}

	r.prevCandle.LastCandleOpenTime = kline.OpenTime
	r.prevCandle.ClosePrice = kline.ClosePrice

}
func (r *RSIIntervalManager) GetIntervalsForCandle(period int, centralRSI float64, newCandleTimestamp uint64) (float64, float64) {

	val1 := r.prevCandle.LastCandleOpenTime / uint64(r.Timeframe) / 60 / 1000
	val2 := newCandleTimestamp / uint64(r.Timeframe) / 60 / 1000

	var copyRSIs []indicators.RSI

	if val1 > 0 && val1 < val2 {

		copyRSIs = make([]indicators.RSI, len(r.rsi.RSIs))
		copy(copyRSIs, r.rsi.RSIs)
		r.rsi.AddPoint(r.prevCandle.ClosePrice)
	}

	up, down, _ := r.rsi.GetIntervalForPeriod(period, centralRSI)

	if val1 > 0 && val1 < val2 {

		r.rsi.RSIs = copyRSIs
	}

	return up, down

}
func TestPeriod() {

	symbol := "ETHUSDT"

	fromTimestamp := uint64(0)

	lowReverse := indicators.NewRSILowReverseIndicator()

	intervals := []int{
		1,
		2,
		3,
		4,
		5,
		6,
		8,
		9,
		10,
		12,
		15,
		16,
		18,
		20,
		24,
		30,
		32,
		36,
		40,
		45,
		48,
		60,
		72,
		80,
		90,
		96,
		120,
		144,
		160,
		180,
		240,
		288,
		360,
		480,
		720,
	}

	rsiManager := make([]RSIIntervalManager, 0, len(intervals))

	rsiOne := indicators.NewRSIMultiplePeriods(250)

	for _, interval := range intervals {

		rsiManager = append(rsiManager, RSIIntervalManager{
			Timeframe:        interval,
			rsi:              indicators.NewRSIMultiplePeriods(250),
			bestSequenceList: list.New(),
			PeriodCount:      make(map[int]uint64),
		})
	}

	endTimestamp := uint64(1569354240000)

	end := false

	for !end {

		klines, err := providers.GetKlinesNew(symbol, "1m", providers.GetKlineRange{Direction: 1, FromTimestamp: fromTimestamp})

		if err != nil {
			log.Fatal(err.Error())
		}

		if len(klines) == 0 {
			break
		}

		//Reverse values
		for i := 0; i < len(klines)/2; i++ {
			j := len(klines) - i - 1
			klines[i], klines[j] = klines[j], klines[i]
		}

		lowsMap := indicators.GenerateMapLows(lowReverse, klines)

		//remove las added element
		klines = klines[:len(klines)-1]
		lowReverse.RemoveLastAdded()

		for idx, kline := range klines {

			if kline.OpenTime >= endTimestamp {
				end = true
				break
			}

			_, ok := lowsMap[idx]

			for idx, rsi := range rsiManager {

				rsi.AddMinuteCandle(kline, ok)
				rsiManager[idx] = rsi
			}

			rsiOne.AddPoint(kline.ClosePrice)

		}

		fromTimestamp = klines[len(klines)-1].OpenTime

	}

	for _, rsi := range rsiManager {

		for e := rsi.bestSequenceList.Front(); e != nil; e = e.Next() {

			f := e.Value.(SequenceData)
			up, down := rsi.GetIntervalsForCandle(f.Period, 15, endTimestamp)

			if down >= up {
				continue
			}
			log.Println(rsi.Timeframe, f.Period, len(f.Timestamps), f.HasFullFilled, f.Timestamps, up, down)
		}
	}

}
