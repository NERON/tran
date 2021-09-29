package manager

import (
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/indicators"
	"log"
	"math"
)

type SequenceItemData struct {
	Period     int
	CentralRSI int
	Up         float64
	Down       float64
	Percentage float64
}

func GenerateMapOfPeriods(symbol string, interval candlescommon.Interval, endTimestamp uint64, centralRSI float64) []SequenceItemData {

	fromTimestamp := uint64(0)
	isOver := false
	lastHandledCandle := uint64(0)

	lowReverse := indicators.NewRSILowReverseIndicator()
	RSI := indicators.NewRSIMultiplePeriods(250)

	currentPeriods := make(map[int]map[int]struct{})

	centralRSIs := []int{5, 10, 15}

	for _, cR := range centralRSIs {
		currentPeriods[cR] = make(map[int]struct{})
	}

	result := make([]SequenceItemData, 0)

	for !isOver {

		log.Println("Start fetching from: ", fromTimestamp)

		var candles []candlescommon.KLine

		if fromTimestamp == 0 {

			candles, _ = GetFirstKLines(symbol, interval, 1000)

		} else {

			candles, _ = GetKLinesInRange(symbol, interval, fromTimestamp, math.MaxUint64, 1000)
		}

		if len(candles) == 0 {
			break
		}

		lowsMap := indicators.GenerateMapLows(lowReverse, candles)

		for idx, candle := range candles {

			if endTimestamp >= candle.OpenTime && endTimestamp <= candle.CloseTime {
				isOver = true
				break
			}

			if _, ok := lowsMap[idx]; ok {

				for _, cR := range centralRSIs {

					bestPeriod, _, _ := RSI.GetBestPeriod(candle.LowPrice, float64(cR))
					up, down, _ := RSI.GetIntervalForPeriod(bestPeriod, float64(cR))

					if candle.LowPrice <= up && candle.LowPrice >= down {

						_, ok1 := currentPeriods[cR][bestPeriod]
						_, ok2 := currentPeriods[cR][bestPeriod-1]

						if ok2 {

							delete(currentPeriods[cR], bestPeriod-1)

						} else if ok1 {

							delete(currentPeriods[cR], bestPeriod)

						} else {
							currentPeriods[cR][bestPeriod] = struct{}{}
						}
					}

				}

			}

			lastHandledCandle = candle.OpenTime
			RSI.AddPoint(candle.ClosePrice)
		}

		fromTimestamp = candles[len(candles)-1].CloseTime
	}

	for cR, periodMap := range currentPeriods {

		for val, _ := range periodMap {

			up, down, _ := RSI.GetIntervalForPeriod(val, float64(cR))

			percentage := (down/up - 1) * 100

			if percentage < 0 && percentage > -5.5 {

				result = append(result, SequenceItemData{
					Period:     val,
					CentralRSI: cR,
					Up:         up,
					Down:       down,
					Percentage: percentage,
				})

			}

		}
	}

	log.Println(lastHandledCandle)
	return result

}
