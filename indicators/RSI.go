package indicators

import "math"

type RSI struct {
	Period uint

	AvgGain float64
	AvgLoss float64

	PointsCount uint
	LastValue   float64
}

func (rsi *RSI) AddPoint(value float64) {

	rsi.PointsCount++

	if rsi.PointsCount > 1 && rsi.PointsCount <= rsi.Period+1 {

		rsi.AvgGain += math.Max(value-rsi.LastValue, 0)
		rsi.AvgLoss += math.Max(rsi.LastValue-value, 0)

		if rsi.PointsCount == rsi.Period+1 {

			rsi.AvgGain /= float64(rsi.Period)
			rsi.AvgLoss /= float64(rsi.Period)
		}

	} else if rsi.PointsCount > rsi.Period+1 {

		rsi.AvgGain = (float64(rsi.Period-1)*rsi.AvgGain + 2*math.Max(value-rsi.LastValue, 0)) / float64(rsi.Period+1)
		rsi.AvgLoss = (float64(rsi.Period-1)*rsi.AvgLoss + 2*math.Max(rsi.LastValue-value, 0)) / float64(rsi.Period+1)
	}

	rsi.LastValue = value

}

func (rsi *RSI) Calculate() (float64, bool) {

	if rsi.PointsCount < rsi.Period+1 {
		return 0, false
	}

	return 100 - 100/(1+rsi.AvgGain/rsi.AvgLoss), true
}
func (rsi *RSI) PredictForNextPoint(value float64) (float64, bool) {

	avgG := rsi.AvgGain
	avgL := rsi.AvgLoss
	pC := rsi.PointsCount
	lV := rsi.LastValue

	rsi.AddPoint(value)

	result, notNaN := rsi.Calculate()

	rsi.AvgGain = avgG
	rsi.AvgLoss = avgL
	rsi.PointsCount = pC
	rsi.LastValue = lV

	return result, notNaN

}
func (rsi *RSI) PredictPrice(RSIValue float64) (float64, bool) {

	currentRSI, ok := rsi.Calculate()

	if !ok {
		return 0, false
	}

	coef := RSIValue / (100 - RSIValue)

	if currentRSI >= RSIValue {

		return float64(rsi.Period-1)*(rsi.AvgLoss-rsi.AvgGain/coef) / 2 + rsi.LastValue, true

	} else {

		return float64(rsi.Period-1)*(rsi.AvgLoss*coef-rsi.AvgGain) / 2  + rsi.LastValue, true
	}

}
