package providers

import (
	"encoding/json"
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

func toFloat(value string) float64 {

	val, err := strconv.ParseFloat(value, 64)

	if err != nil {

		log.Fatal(err.Error())
	}

	return val
}

func GetKlines(symbol string, interval string, startTimestamp uint64, endTimestamp uint64) []candlescommon.KLine {

	result := make([]candlescommon.KLine, 0)

	for i := 0; i < 2; i++ {

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
		
		log.Println(urlS)

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		klines := make([]interface{}, 0)

		err = json.Unmarshal(body, &klines)

		if err != nil {

			log.Fatal("Get error: ", err.Error())
		}
		
		if len(klines) == 0 {
		   return result
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
		
		log.Println("temp result ",len(result))
		
		if len(result) > 0 {
			endTimestamp = result[len(result)-1].OpenTime - 1
		}
		

	}
	
	log.Println(len(result))

	itemCount := len(result)

	for i := 0; i < itemCount/2; i++ {

		mirrorIdx := itemCount - i - 1
		result[i], result[mirrorIdx] = result[mirrorIdx], result[i]

	}
	
	
	
	result[len(result)-1].Closed = false

	return result

}
