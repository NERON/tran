package manager

import (
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/indicators"
	"log"
	"math"
	"time"
)

type SequenceItemData struct {
	Period             int
	CentralRSI         int
	Up                 float64
	Down               float64
	Percentage         float64
	OriginalPercentage float64
}

type PeriodInfo struct {
	Percentage float64
}

func GenerateMapOfPeriods(symbol string, interval candlescommon.Interval, endTimestamp uint64, centralRSI float64) []SequenceItemData {

	fromTimestamp := uint64(0)
	isOver := false
	lastHandledCandle := uint64(0)

	lowReverse := indicators.NewRSILowReverseIndicator()
	RSI := indicators.NewRSIMultiplePeriods(250)

	currentPeriods := make(map[int]map[int]PeriodInfo)

	centralRSIs := []int{15}

	maxPeriod := 0
	tmpt := uint64(0)

	for _, cR := range centralRSIs {
		currentPeriods[cR] = make(map[int]PeriodInfo)
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

					if candle.OpenTime == 1637290560000 || candle.OpenTime == 1637287680000 {
						bestPeriod += 1
					}

					up, down, _ := RSI.GetIntervalForPeriod(bestPeriod, float64(cR))

					percentage := (down/up - 1) * 100

					if bestPeriod > 2 || (bestPeriod == 2 && candle.LowPrice <= up) {

						filledPercentage := (up - candle.LowPrice) / (up - down) * 100

						_, ok1 := currentPeriods[cR][bestPeriod]
						_, ok2 := currentPeriods[cR][bestPeriod-1]

						if ok2 {

							delete(currentPeriods[cR], bestPeriod-1)

						} else if ok1 {

							delete(currentPeriods[cR], bestPeriod)

						} else {

							currentPeriods[cR][bestPeriod] = PeriodInfo{
								Percentage: percentage,
							}
						}

						str := ""

						if filledPercentage > 79 {
							str = "                       !"
						}

						log.Println(cR, time.Unix(int64(candle.OpenTime/1000), 0), bestPeriod, ok1, ok2, filledPercentage, str)

						if bestPeriod > maxPeriod {
							maxPeriod = bestPeriod
							tmpt = candle.CloseTime
						}

					}

				}

			}

			lastHandledCandle = candle.OpenTime
			RSI.AddPoint(candle.LowPrice)

		}

		fromTimestamp = candles[len(candles)-1].CloseTime
	}

	for cR, periodMap := range currentPeriods {

		for val, periodInfo := range periodMap {

			up, down, _ := RSI.GetIntervalForPeriod(val, float64(cR))

			percentage := (down/up - 1) * 100

			if true {

				result = append(result, SequenceItemData{
					Period:             val,
					CentralRSI:         cR,
					Up:                 math.Floor(up*1000000000) / 1000000000,
					Down:               math.Floor(down*1000000000) / 1000000000,
					Percentage:         percentage,
					OriginalPercentage: periodInfo.Percentage,
				})

			}

		}
	}

	log.Println(maxPeriod, tmpt)
	log.Println(lastHandledCandle)
	return result

}
