package manager

import (
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/indicators"
	"log"
	"math"
)

func GenerateMapOfPeriods(symbol string, interval candlescommon.Interval, endTimestamp uint64, centralRSI float64) map[float64]map[int]struct{} {

	fromTimestamp := uint64(0)
	isOver := false
	lastHandledCandle := uint64(0)

	lowReverse := indicators.NewRSILowReverseIndicator()
	RSI := indicators.NewRSIMultiplePeriods(250)

	currentPeriods := make(map[float64]map[int]struct{})

	centralRSIs := []float64{5, 10, 15}

	for _, cR := range centralRSIs {
		currentPeriods[cR] = make(map[int]struct{})
	}

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

					bestPeriod, _, _ := RSI.GetBestPeriod(candle.LowPrice, cR)
					up, _, _ := RSI.GetIntervalForPeriod(bestPeriod, cR)

					if bestPeriod > 2 || (bestPeriod == 2 && candle.LowPrice <= up) {

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

	log.Println(lastHandledCandle)

	log.Println(currentPeriods)
	return currentPeriods

}
