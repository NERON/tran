package providers

import (
	"encoding/json"
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

type BinanceProvider struct {
	baseUrl string
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
