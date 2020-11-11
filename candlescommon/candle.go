package candlescommon

import (
	"math"
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

func HoursGroupKline(klines []KLine, hours uint64) []KLine {

	groupedKlines := make([]KLine, 0)

	currentKline := klines[0]

	for i := 1; i < len(klines); i++ {

		if klines[i].OpenTime%(hours*3600*1000) == (hours-1)*3600*1000 {

			currentKline.PrevCloseCandleTimestamp = klines[i].CloseTime
			groupedKlines = append(groupedKlines, currentKline)
			currentKline = klines[i]

			continue
		}

		if klines[i].OpenTime%(hours*3600*1000) == 0 {

			currentKline.OpenTime = klines[i].OpenTime
			currentKline.OpenPrice = klines[i].OpenPrice

		}

		currentKline.HighPrice = math.Max(currentKline.HighPrice, klines[i].HighPrice)
		currentKline.LowPrice = math.Min(currentKline.LowPrice, klines[i].LowPrice)

		currentKline.BaseVolume += klines[i].BaseVolume
		currentKline.TakerBuyBaseVolume += klines[i].TakerBuyBaseVolume
		currentKline.QuoteVolume += klines[i].QuoteVolume
		currentKline.TakerBuyQuoteVolume += klines[i].TakerBuyQuoteVolume

	}

	return groupedKlines

}

func HoursGroupKlineAsc(klines []KLine, hours uint64) []KLine {

	groupedKlines := make([]KLine, 0)

	i := 0

	//iterate and skip all unfinished klines in the beginning
	for i < len(klines) && klines[i].OpenTime%(hours*3600*1000) != 0 {
		i++
	}

	var currentKline KLine
	var prevCloseCandle uint64
	var counter uint64

	for ; i < len(klines); i++ {

		if klines[i].OpenTime%(hours*3600*1000) == 0 {

			if currentKline.OpenTime > 0 {
				prevCloseCandle = currentKline.CloseTime
				groupedKlines = append(groupedKlines, currentKline)
			}

			currentKline = klines[i]
			currentKline.PrevCloseCandleTimestamp = prevCloseCandle
			counter = 0
			continue

		}

		currentKline.ClosePrice = klines[i].ClosePrice
		currentKline.CloseTime = klines[i].CloseTime
		currentKline.Closed = klines[i].Closed

		currentKline.HighPrice = math.Max(currentKline.HighPrice, klines[i].HighPrice)
		currentKline.LowPrice = math.Min(currentKline.LowPrice, klines[i].LowPrice)

		currentKline.BaseVolume += klines[i].BaseVolume
		currentKline.TakerBuyBaseVolume += klines[i].TakerBuyBaseVolume
		currentKline.QuoteVolume += klines[i].QuoteVolume
		currentKline.TakerBuyQuoteVolume += klines[i].TakerBuyQuoteVolume
		counter++

	}

	if currentKline.Closed == false || counter == (hours-1) {
		groupedKlines = append(groupedKlines, currentKline)
	}

	return groupedKlines
}
func MinutesGroupKlineAsc(klines []KLine, originalMinutes uint64, minutes uint64) []KLine {

	groupedKlines := make([]KLine, 0)

	i := 0

	//iterate and skip all unfinished klines in the beginning
	for i < len(klines) && klines[i].OpenTime%(minutes*60*1000) != 0 {
		i++
	}

	var currentKline KLine
	var prevCloseCandle uint64
	var counter uint64

	for ; i < len(klines); i++ {

		if klines[i].OpenTime%(minutes*60*1000) == 0 {

			if currentKline.OpenTime > 0 {
				prevCloseCandle = currentKline.CloseTime
				groupedKlines = append(groupedKlines, currentKline)
			}

			currentKline = klines[i]
			currentKline.PrevCloseCandleTimestamp = prevCloseCandle
			counter = 0
			continue

		}

		currentKline.ClosePrice = klines[i].ClosePrice
		currentKline.CloseTime = klines[i].CloseTime
		currentKline.Closed = klines[i].Closed

		currentKline.HighPrice = math.Max(currentKline.HighPrice, klines[i].HighPrice)
		currentKline.LowPrice = math.Min(currentKline.LowPrice, klines[i].LowPrice)

		currentKline.BaseVolume += klines[i].BaseVolume
		currentKline.TakerBuyBaseVolume += klines[i].TakerBuyBaseVolume
		currentKline.QuoteVolume += klines[i].QuoteVolume
		currentKline.TakerBuyQuoteVolume += klines[i].TakerBuyQuoteVolume
		counter++

	}

	if currentKline.Closed == false || counter == (minutes/originalMinutes-1) {
		groupedKlines = append(groupedKlines, currentKline)
	}

	return groupedKlines
}
