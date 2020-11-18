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

var limiter = rate.NewLimiter(rate.Limit(15), 3)

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

func NewStandardBinanceProvider() BinanceProvider {

	return BinanceProvider{
		baseUrl: "https://api.binance.com/",
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

func GetKlines(symbol string, interval string, startTimestamp uint64, endTimestamp uint64, reverseOrder bool) []candlescommon.KLine {

	result := make([]candlescommon.KLine, 0)

	for i := 0; i < 3; i++ {

		urlS := fmt.Sprintf("https://api.binance.com/api/v1/klines?symbol=%s&interval=%s&limit=1000", symbol, interval)

		if endTimestamp > 0 {

			urlS = fmt.Sprintf(urlS+"&endTime=%d", endTimestamp)
		}

		if startTimestamp > 0 {

			urlS = fmt.Sprintf(urlS+"&startTime=%d", startTimestamp)
		}

		resp, err := http.Get(urlS)

		if err != nil {

			log.Fatal("Get error: ", err.Error())
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		klines := make([]interface{}, 0)

		err = json.Unmarshal(body, &klines)

		log.Println("klines length", len(klines))

		if err != nil {

			log.Fatal("Get error: ", err.Error())
		}

		for j := len(klines) - 1; j > 0; j-- {

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

			result = append(result, kline)

		}

		if len(result) == 0 {
			break
		}

		endTimestamp = result[len(result)-1].OpenTime - 1

	}

	if !reverseOrder {

		itemCount := len(result)

		for i := 0; i < itemCount/2; i++ {

			mirrorIdx := itemCount - i - 1
			result[i], result[mirrorIdx] = result[mirrorIdx], result[i]

		}

		if len(result) > 0 {
			result[len(result)-1].Closed = false
		}

	} else {

		if len(result) > 0 {
			result[0].Closed = false
		}

	}

	return result

}

func GetKlinesTest(symbol string, interval string, startTimestamp uint64, endTimestamp uint64, limit int) []candlescommon.KLine {

	result := make([]candlescommon.KLine, 0)

	for i := 0; i < limit; i++ {

		urlS := fmt.Sprintf("https://api.binance.com/api/v1/klines?symbol=%s&interval=%s&limit=1000", symbol, interval)

		if endTimestamp > 0 {

			urlS = fmt.Sprintf(urlS+"&endTime=%d", endTimestamp)
		}

		if startTimestamp > 0 {

			urlS = fmt.Sprintf(urlS+"&startTime=%d", startTimestamp)
		}

		resp, err := http.Get(urlS)

		if err != nil {

			log.Fatal("Get error: ", err.Error())
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		klines := make([]interface{}, 0)

		err = json.Unmarshal(body, &klines)

		if err != nil {

			log.Fatal("Get error: ", err.Error())
		}

		for j := len(klines) - 1; j > 0; j-- {

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

			result = append(result, kline)

		}

		if len(result) == 0 || len(klines) == 0 {
			break
		}

		endTimestamp = result[len(result)-1].OpenTime - 1

		time.Sleep(time.Second * 1)

	}

	itemCount := len(result)

	for i := 0; i < itemCount/2; i++ {

		mirrorIdx := itemCount - i - 1
		result[i], result[mirrorIdx] = result[mirrorIdx], result[i]

	}

	if len(result) > 0 {
		result[len(result)-1].Closed = false
	}

	log.Println("result length", len(result))

	return result

}
func GetLastKlines(symbol string, interval string) ([]candlescommon.KLine, error) {

	klines, err := getKline(symbol, interval, GetKlineRange{Direction: 1, FromTimestamp: 0})

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

	log.Println("fetching data...", interval, ranges)

	klines, err := getKline(symbol, interval, ranges)

	if err != nil {
		return nil, err
	}

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
	}

	if len(klines) > 0 && klines[len(klines)-1].PrevCloseCandleTimestamp == math.MaxUint64 {
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

func GetAvailableSymbols() ([]string, error) {

	resp, err := http.Get("https://api.binance.com/api/v3/exchangeInfo")

	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	err = resp.Body.Close()

	if err != nil {
		return nil, err
	}

	type SymbolInfoJSON struct {
		Symbol string `json:"symbol"`
	}

	type ExchangeInfoJSON struct {
		Symbols []SymbolInfoJSON `json:"symbols"`
	}

	exchangeInfo := ExchangeInfoJSON{}

	err = json.Unmarshal(body, &exchangeInfo)

	if err != nil {
		return nil, err
	}

	symbolsNames := make([]string, 0)

	for _, symbol := range exchangeInfo.Symbols {
		symbolsNames = append(symbolsNames, symbol.Symbol)
	}

	return symbolsNames, nil

}
