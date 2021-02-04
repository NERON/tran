package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"golang.org/x/time/rate"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
)

var limiter = rate.NewLimiter(rate.Limit(20), 3)

type BinanceProvider struct {
	baseUrl string
}

type GetKlineRange struct {
	Direction     uint
	FromTimestamp uint64
}

func GetSupportedTimeframes() map[string][]uint {

	return map[string][]uint{
		"m": {1, 3, 5, 15, 30},
		"h": {1, 2, 4, 6, 8, 12},
		"d": {1, 3},
		"w": {1},
		"M": {1},
	}
}

func (provider *BinanceProvider) GetServerTime() time.Time {

	return time.Now()

}

func toFloat(value string) float64 {

	val, err := strconv.ParseFloat(value, 64)

	if err != nil {

		log.Fatal(err.Error())
	}

	return val
}

func GetLastKlines(symbol string, interval string) ([]candlescommon.KLine, error) {

	t := time.Now()

	klines, err := getKline(symbol, interval, GetKlineRange{Direction: 1, FromTimestamp: 0})

	log.Println("API GetLastKlines time ", time.Since(t))

	if err != nil {
		return nil, err
	}

	if len(klines) == 0 {
		return klines, nil
	}

	if klines[len(klines)-1].PrevCloseCandleTimestamp == math.MaxUint64 {
		klines[len(klines)-1].PrevCloseCandleTimestamp = 0
		return klines, nil
	}

	return klines[:len(klines)-1], nil

}

func GetKlinesNew(symbol string, interval string, ranges GetKlineRange) ([]candlescommon.KLine, error) {

	t := time.Now()

	klines, err := getKline(symbol, interval, ranges)

	if err != nil {
		return nil, err
	}

	log.Println("API GetKLinesNew time ", interval, ranges, time.Since(t))

	if len(klines) == 0 {
		return klines, nil
	}

	//if last kline is set to closed we should remove it, because we don't know about it's real state
	if klines[0].Closed {
		klines = klines[1:]
	}

	//if kline with smallest open time has prevCloseCandle equals 0, we should remove them
	if len(klines) > 0 && klines[len(klines)-1].PrevCloseCandleTimestamp == 0 {
		klines = klines[:len(klines)-1]
	} else if len(klines) > 0 && klines[len(klines)-1].PrevCloseCandleTimestamp == math.MaxUint64 {
		klines[len(klines)-1].PrevCloseCandleTimestamp = 0
	}

	return klines, nil
}
func getKline(symbol string, interval string, ranges GetKlineRange) ([]candlescommon.KLine, error) {

	limiter.Wait(context.Background())

	urlS := fmt.Sprintf("https://api.binance.com/api/v1/klines?symbol=%s&interval=%s&limit=1000", symbol, interval)

	if ranges.Direction == 0 {

		urlS = fmt.Sprintf(urlS+"&endTime=%d", ranges.FromTimestamp)

	} else if ranges.Direction == 1 && ranges.FromTimestamp > 0 {

		urlS = fmt.Sprintf(urlS+"&startTime=%d", ranges.FromTimestamp)
	}

	resp, err := http.Get(urlS)

	if err != nil {

		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	klines := make([]interface{}, 0)

	err = json.Unmarshal(body, &klines)

	if err != nil {

		return nil, err
	}

	result := make([]candlescommon.KLine, 0)

	if len(klines) == 0 {
		return result, nil
	}

	for j := len(klines) - 1; j >= 0; j-- {

		data := klines[j]

		k := data.([]interface{})

		kline := candlescommon.KLine{}
		kline.Symbol = symbol
		kline.OpenTime = uint64(k[0].(float64))
		kline.OpenPrice = toFloat(k[1].(string))
		kline.HighPrice = toFloat(k[2].(string))
		kline.LowPrice = toFloat(k[3].(string))
		kline.ClosePrice = toFloat(k[4].(string))
		kline.BaseVolume = toFloat(k[5].(string))
		kline.CloseTime = uint64(k[6].(float64))
		kline.QuoteVolume = toFloat(k[7].(string))
		kline.TakerBuyBaseVolume = toFloat(k[9].(string))
		kline.TakerBuyQuoteVolume = toFloat(k[10].(string))
		kline.Closed = true

		if len(result) > 0 {

			result[len(result)-1].PrevCloseCandleTimestamp = kline.CloseTime
		}

		if kline.OpenTime > kline.CloseTime {
			kline.CloseTime = kline.OpenTime
		}

		result = append(result, kline)

	}

	//if we fetch last klines, or result no more than 1000, we reach the end
	if ranges.Direction == 1 && (ranges.FromTimestamp == 0 || len(result) < 1000) {
		result[0].Closed = false
	}

	//if we fetch old klines and
	if ranges.Direction == 0 && len(result) < 1000 {
		result[len(result)-1].PrevCloseCandleTimestamp = math.MaxUint64
	}

	return result, nil

}
