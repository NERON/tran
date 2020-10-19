package indicators

import "math"

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

type RSIMultiplePeriods struct {
	RSIs []RSI
}

func (rsip *RSIMultiplePeriods) AddPoint(addPrice float64) {

	for i := 0; i < len(rsip.RSIs); i++ {

		rsip.RSIs[i].AddPoint(addPrice)
	}
}
func (rsip *RSIMultiplePeriods) GetBestPeriodByRSIValue(priceFor float64, centralRSI float64) int {

	BestRSIDiff := 99999.0
	bestPeriod := 0

	for i := 0; i < len(rsip.RSIs); i++ {

		RSIValue, ok := rsip.RSIs[i].PredictForNextPoint(priceFor)

		if !ok {
			return bestPeriod
		}

		if math.Abs(RSIValue-centralRSI) < BestRSIDiff {
			bestPeriod = i + 2
			BestRSIDiff = math.Abs(RSIValue - centralRSI)
		}
	}

	return bestPeriod

}
func (rsip *RSIMultiplePeriods) GetBestPeriod(priceFor float64, centralRSI float64) (int, float64) {

	bestDiff := 10000000000000000.0
	bestPeriod := 0
	bestRSIVal := 0.0

	for i := 0; i < len(rsip.RSIs); i++ {

		price, ok := rsip.RSIs[i].PredictPrice(centralRSI)
		rsiVal, _ := rsip.RSIs[i].PredictForNextPoint(priceFor)

		price = toFixed(price, 8)

		if !ok {
			return bestPeriod, bestRSIVal
		}

		if math.Abs(price-priceFor) < bestDiff {
			bestPeriod = i + 2
			bestDiff = math.Abs(price - priceFor)
			bestRSIVal = rsiVal
		}
	}

	return bestPeriod, bestRSIVal
}

func NewRSIMultiplePeriods(maxPeriod int) *RSIMultiplePeriods {

	RSIs := make([]RSI, maxPeriod)

	for i := 0; i < maxPeriod-1; i++ {
		RSIs[i] = RSI{Period: uint(i + 2)}
	}

	return &RSIMultiplePeriods{RSIs: RSIs}
}
