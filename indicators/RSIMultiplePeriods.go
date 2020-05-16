package indicators

import "math"

type RSIMultiplePeriods struct {
	RSIs []RSI
}

func(rsip *RSIMultiplePeriods) AddPoint(addPrice float64) {

	for i:=0; i < len(rsip.RSIs); i++ {

		rsip.RSIs[i].AddPoint(addPrice)
	}
}

func(rsip *RSIMultiplePeriods) GetBestPeriod(priceFor float64,centralRSI float64) int {

	bestRSIDiff := 100.0
	bestPeriod := 0

	for i:=0; i < len(rsip.RSIs); i++ {

		rsi, ok := rsip.RSIs[i].PredictForNextPoint(priceFor)

		if !ok {
			return bestPeriod
		}

		if math.Abs(rsi - centralRSI) < bestRSIDiff {
			bestPeriod = i + 2
			bestRSIDiff = math.Abs(rsi - centralRSI)
		}
	}

	return bestPeriod
}

func NewRSIMultiplePeriods(maxPeriod int) *RSIMultiplePeriods {

	RSIs := make([]RSI,maxPeriod)

	for i := 0; i < maxPeriod - 1; i++ {
		RSIs[i] = RSI{Period:uint(i+2)}
	}

	return &RSIMultiplePeriods{RSIs:RSIs}
}
