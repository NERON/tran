package manager

import (
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/indicators"
	"log"
	"math"
)

func GenerateMapOfPeriods(symbol string, interval candlescommon.Interval, endTimestamp uint64) map[int]struct{} {

	centralRSI := float64(15)
	fromTimestamp := uint64(0)
	isOver := false

	lowReverse := indicators.NewRSILowReverseIndicator()
	RSI := indicators.NewRSIMultiplePeriods(250)

	currentPeriods := make(map[int]struct{})

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

			if candle.OpenTime >= endTimestamp {
				isOver = true
				break
			}

			if _, ok := lowsMap[idx]; ok {

				bestPeriod, _, _ := RSI.GetBestPeriod(candle.LowPrice, centralRSI)
				up, _, _ := RSI.GetIntervalForPeriod(bestPeriod, centralRSI)

				if bestPeriod > 2 || (bestPeriod == 2 && candle.LowPrice <= up) {

					_, ok1 := currentPeriods[bestPeriod]
					_, ok2 := currentPeriods[bestPeriod-1]

					if ok2 {

						delete(currentPeriods, bestPeriod-1)

					} else if ok1 {

						delete(currentPeriods, bestPeriod)

					} else {
						currentPeriods[bestPeriod] = struct{}{}
					}
				}

			}

			RSI.AddPoint(candle.ClosePrice)
		}

		log.Println(candles[len(candles)-1].CloseTime)
		fromTimestamp = candles[len(candles)-1].CloseTime
	}

	return currentPeriods

}
