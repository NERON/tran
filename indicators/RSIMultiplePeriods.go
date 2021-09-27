package indicators

import (
	"math"
)

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
	bestPeriod := 1

	for i := 0; i < len(rsip.RSIs); i++ {

		RSIValue, ok := rsip.RSIs[i].PredictForNextPoint(priceFor)

		if !ok {
			return bestPeriod
		}

		if math.Abs(RSIValue-centralRSI) < BestRSIDiff {
			bestPeriod = int(rsip.RSIs[i].Period)
			BestRSIDiff = math.Abs(RSIValue - centralRSI)
		}
	}

	return bestPeriod

}
func (rsip *RSIMultiplePeriods) GetBestPeriod(priceFor float64, centralRSI float64) (int, float64, float64) {

	bestDiff := 10000000000000000.0
	bestPeriod := 1
	bestPrice := 0.0
	bestRSIVal := 0.0

	for i := 0; i < len(rsip.RSIs); i++ {

		price, ok := rsip.RSIs[i].PredictPrice(centralRSI)
		rsiVal, _ := rsip.RSIs[i].PredictForNextPoint(priceFor)

		if !ok {
			return bestPeriod, bestRSIVal, bestPrice
		}

		if math.Abs(price-priceFor) < bestDiff && rsip.RSIs[i].Period > 1 {
			bestPeriod = int(rsip.RSIs[i].Period)
			bestPrice = price
			bestDiff = math.Abs(price - priceFor)
			bestRSIVal = rsiVal
		}

	}

	return bestPeriod, bestRSIVal, bestPrice
}

func (rsip *RSIMultiplePeriods) GetIntervalForPeriod(period int, centralRSI float64) (float64, float64, float64) {

	central, _ := rsip.RSIs[period-1].PredictPrice(centralRSI)
	upBorder := central
	lowBorder := central

	if period > 1 {

		Val, _ := rsip.RSIs[period-2].PredictPrice(centralRSI)

		upBorder = (upBorder + Val) / 2

	}

	if period < len(rsip.RSIs) {

		Val, ok := rsip.RSIs[period].PredictPrice(centralRSI)

		if ok {
			lowBorder = (lowBorder + Val) / 2
		}

	}

	return upBorder, lowBorder, central

}

func NewRSIMultiplePeriods(maxPeriod int) *RSIMultiplePeriods {

	RSIs := make([]RSI, maxPeriod)

	for i := 0; i < maxPeriod; i++ {
		RSIs[i] = RSI{Period: uint(i + 1)}
	}

	return &RSIMultiplePeriods{RSIs: RSIs}
}
