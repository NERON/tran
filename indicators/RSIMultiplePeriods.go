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

	bestDiff := 10000000000000000.0
	bestPeriod := 0

	for i:=0; i < len(rsip.RSIs); i++ {

		price, ok := rsip.RSIs[i].PredictPrice(centralRSI)

		if !ok {
			return bestPeriod
		}

		if math.Abs(price - priceFor) < bestDiff {
			bestPeriod = i + 2
			bestDiff = math.Abs(price - priceFor)
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
