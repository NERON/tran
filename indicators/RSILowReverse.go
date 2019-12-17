package indicators

import "log"

type rsiLowReverse struct {
	rsi           RSI
	lastRSIValues []float64
}

func (r *rsiLowReverse) AddPoint(calcValue float64,addValue float64) {

	val, isNotNan := r.rsi.PredictForNextPoint(calcValue)

	if isNotNan {
		r.lastRSIValues[0] = r.lastRSIValues[1]
		r.lastRSIValues[1] = r.lastRSIValues[2]
		r.lastRSIValues[2] = val

		log.Println(r.lastRSIValues)
	}

	r.rsi.AddPoint(addValue)

}

func (r *rsiLowReverse) IsPreviousLow() bool {

	//if not values filled,we can get value
	if r.lastRSIValues[0] < 0 {
		return false
	}

	return r.lastRSIValues[1]  <= r.lastRSIValues[0]  && r.lastRSIValues[1] <= r.lastRSIValues[2]
}

func NewRSILowReverseIndicator() *rsiLowReverse {

	lastValues := []float64{-1,-1,-1}

	return &rsiLowReverse{rsi: RSI{Period: 3}, lastRSIValues: lastValues}

}
