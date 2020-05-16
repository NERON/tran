package indicators

import "math"

type RSI struct {
	Period uint

	avgGain float64
	avgLoss float64

	pointsCount uint
	lastValue   float64
}

func (rsi *RSI) AddPoint(value float64) {

	rsi.pointsCount++

	if rsi.pointsCount > 1 && rsi.pointsCount <= rsi.Period+1 {

		rsi.avgGain += math.Max(value - rsi.lastValue,0)
		rsi.avgLoss += math.Max(rsi.lastValue - value,0)

		if rsi.pointsCount == rsi.Period + 1 {

			rsi.avgGain /= float64(rsi.Period)
			rsi.avgLoss /= float64(rsi.Period)
		}

	} else if rsi.pointsCount > rsi.Period+1 {

		rsi.avgGain = (float64(rsi.Period-1) * rsi.avgGain + math.Max(value - rsi.lastValue,0)) / float64(rsi.Period)
		rsi.avgLoss = (float64(rsi.Period-1) * rsi.avgLoss + math.Max(rsi.lastValue - value,0)) / float64(rsi.Period)
	}

	rsi.lastValue = value

}

func (rsi *RSI) Calculate() (float64,bool){

	if rsi.pointsCount <= rsi.Period + 1 {
		return 0,false
	}

	return 100 - 100 / ( 1  + rsi.avgGain / rsi.avgLoss), true
}
func (rsi *RSI) PredictForNextPoint(value float64) (float64,bool) {

	avgG := rsi.avgGain
	avgL := rsi.avgLoss
	pC := rsi.pointsCount
	lV := rsi.lastValue

	rsi.AddPoint(value)

	result, notNaN := rsi.Calculate()

	rsi.avgGain = avgG
	rsi.avgLoss = avgL
	rsi.pointsCount = pC
	rsi.lastValue = lV

	return result,notNaN

}
func (rsi * RSI) PredictPrice(RSIValue float64) (float64,bool) {

	currentRSI,ok := rsi.Calculate()

	if !ok {
		return 0,false
	}

	coef := RSIValue / (100-RSIValue)

	if currentRSI >= RSIValue {

		return float64(rsi.Period - 1) * (rsi.avgLoss - rsi.avgGain/coef ) + rsi.lastValue,true

	} else {

		return float64(rsi.Period - 1) * (rsi.avgLoss*coef - rsi.avgGain) + rsi.lastValue,true
	}

}