package indicators

import (
	"github.com/NERON/tran/candlescommon"
)

type ReverseLowInterface interface {
	AddPoint(calcValue float64, addValue float64)
	IsPreviousLow() bool
}

type rsiLowReverse struct {
	lastRSIValues []float64
}

func (r *rsiLowReverse) AddPoint(calcValue float64, addValue float64) {

	r.lastRSIValues[0] = r.lastRSIValues[1]
	r.lastRSIValues[1] = r.lastRSIValues[2]
	r.lastRSIValues[2] = calcValue

}

func (r *rsiLowReverse) IsPreviousLow() bool {

	//if not values filled,we can get value
	if r.lastRSIValues[0] < 0 {
		return false
	}

	return r.lastRSIValues[1] <= r.lastRSIValues[0] && r.lastRSIValues[1] < r.lastRSIValues[2]
}

func NewRSILowReverseIndicator() ReverseLowInterface {

	lastValues := []float64{-1, -1, -1}

	return &rsiLowReverse{lastRSIValues: lastValues}

}

func GenerateMapLows(lowReverse ReverseLowInterface, candles []candlescommon.KLine) map[int]struct{} {

	lowsMap := make(map[int]struct{})

	for idx, candle := range candles {

		lowReverse.AddPoint(candle.LowPrice, 0)

		if lowReverse.IsPreviousLow() {

			lowsMap[idx-1] = struct{}{}

		} else if idx > 0 && candle.OpenPrice <= candle.ClosePrice && candles[idx-1].LowPrice >= candle.LowPrice {
			lowsMap[idx] = struct{}{}
		}

	}

	return lowsMap

}
