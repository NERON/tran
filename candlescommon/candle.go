package candlescommon

import (
	"math"
	"strconv"
)

type KLine struct {
	Symbol                   string
	OpenTime                 uint64
	CloseTime                uint64
	OpenPrice                float64
	ClosePrice               float64
	HighPrice                float64
	LowPrice                 float64
	BaseVolume               float64
	QuoteVolume              float64
	TakerBuyBaseVolume       float64
	TakerBuyQuoteVolume      float64
	PrevCloseCandleTimestamp uint64
	Closed                   bool
}

type Interval struct {
	Letter   string
	Duration uint
}

func IntervalFromStr(intervalStr string) Interval {

	duration, _ := strconv.ParseUint(intervalStr[:len(intervalStr)-1], 10, 64)

	interval := Interval{Duration: uint(duration), Letter: string(intervalStr[len(intervalStr)-1])}

	return interval

}

func GroupKline(klines []KLine, groupCount int) []KLine {

	newKlines := make([]KLine, 0)

	newKline := KLine{}

	prevClosed := uint64(0)

	for idx, kline := range klines {

		if idx%groupCount == 0 {

			if idx > 0 {
				prevClosed = newKline.CloseTime
				newKlines = append(newKlines, newKline)
			}

			newKline = kline
			newKline.PrevCloseCandleTimestamp = prevClosed

		} else {

			newKline.HighPrice = math.Max(newKline.HighPrice, kline.HighPrice)
			newKline.LowPrice = math.Min(newKline.LowPrice, kline.LowPrice)
			newKline.ClosePrice = kline.ClosePrice
			newKline.CloseTime = kline.CloseTime
			newKline.BaseVolume += kline.BaseVolume
			newKline.TakerBuyBaseVolume += kline.TakerBuyBaseVolume
			newKline.QuoteVolume += newKline.QuoteVolume
			newKline.TakerBuyQuoteVolume += kline.TakerBuyQuoteVolume
		}

	}

	return newKlines
}

func HoursGroupKlineDesc(klines []KLine, hours uint64) []KLine {
	return MinutesGroupKlineDesc(klines, hours*60)
}

func MinutesGroupKlineDesc(klines []KLine, minutes uint64) []KLine {

	//grouped klines
	groupedKlines := make([]KLine, 0)

	//check if it's not null
	if len(klines) == 0 {
		return groupedKlines
	}

	//set index
	index := len(klines) - 1

	//if first kline in array isn't have start open time iterating...
	if klines[index].OpenTime%(minutes*60*1000) != 0 && klines[index].PrevCloseCandleTimestamp != 0 {

		division := klines[index].OpenTime / (minutes * 60 * 1000)

		for ; index >= 0; index-- {

			newDivision := klines[index].OpenTime / (minutes * 60 * 1000)

			//find next open time start...
			if newDivision > division {
				break
			}
		}

	}

	currentKline := KLine{Closed: true}

	var division = uint64(0)

	//iterate over next values
	for ; index >= 0; index-- {

		newDivision := klines[index].OpenTime / (minutes * 60 * 1000)

		//if we find, that divisor have been increased, we should create new Kline
		if newDivision > division {

			//if it's not first kline, save previous first kline
			if currentKline.OpenTime > 0 {
				//prepend item
				groupedKlines = append([]KLine{currentKline}, groupedKlines...)
			}

			//assign new kline
			currentKline = klines[index]

			//open time set to start interval
			currentKline.OpenTime = newDivision * (minutes * 60 * 1000)

			//close time is set to as open time + minutes - 1 ms
			currentKline.CloseTime = (newDivision+1)*(minutes*60*1000) - 1

			//calculate start time for previous kline
			prevCandleDivisor := currentKline.PrevCloseCandleTimestamp / (minutes * 60 * 1000)
			currentKline.PrevCloseCandleTimestamp = (prevCandleDivisor+1)*(minutes*60*1000) - 1
		}

		if klines[index].PrevCloseCandleTimestamp == 0 {
			currentKline.PrevCloseCandleTimestamp = 0
		}

		//set close price to current
		currentKline.ClosePrice = klines[index].ClosePrice
		//choose max price
		currentKline.HighPrice = math.Max(currentKline.HighPrice, klines[index].HighPrice)
		//choose min price
		currentKline.LowPrice = math.Min(currentKline.LowPrice, klines[index].LowPrice)

		//add volume data...
		currentKline.BaseVolume += klines[index].BaseVolume
		currentKline.TakerBuyBaseVolume += klines[index].TakerBuyBaseVolume
		currentKline.QuoteVolume += klines[index].QuoteVolume
		currentKline.TakerBuyQuoteVolume += klines[index].TakerBuyQuoteVolume

		//closed status sets based on last kline
		currentKline.Closed = klines[index].Closed

		division = newDivision

	}

	//we should handle two situations,when we should also prepend a kline
	//first: last candle is not closed
	//second: last original kline completes the new kline, in this situation we should check their close time
	if currentKline.Closed == false || klines[0].CloseTime == currentKline.CloseTime {

		//prepend item
		groupedKlines = append([]KLine{currentKline}, groupedKlines...)

	}

	return groupedKlines

}
