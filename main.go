package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/NERON/tran/indicators"
)

var DatabaseManager *sql.DB

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
}

func toFloat(value string) float64 {

	val, err := strconv.ParseFloat(value, 64)

	if err != nil {

		log.Fatal(err.Error())
	}

	return val
}

func GetKlines(symbol string, interval string, startTimestamp uint64, endTimestamp uint64) []KLine {

	result := make([]KLine, 0)

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

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		klines := make([]interface{}, 0)

		err = json.Unmarshal(body, &klines)

		if err != nil {

			log.Fatal("Get error: ", err.Error())
		}

		var prevKline *KLine = nil

		for j := len(klines) - 1; j > 0; j-- {

			data := klines[j]

			k := data.([]interface{})

			kline := KLine{}
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

			if prevKline != nil {

				result[len(result)-1].PrevCloseCandleTimestamp = kline.CloseTime
			}

			prevKline = &kline
			result = append(result, kline)

		}

		endTimestamp = result[len(result)-1].OpenTime - 1
		log.Println("SUCCESS API", endTimestamp)

	}

	itemCount := len(result)

	for i := 0; i < itemCount/2; i++ {

		mirrorIdx := itemCount - i - 1
		result[i], result[mirrorIdx] = result[mirrorIdx], result[i]

	}

	return result

}
func SaveCandles() {
	klines := GetKlines("BTCUSDT", "1h", 0, 0)

	stmt, err := DatabaseManager.Prepare(`INSERT INTO public.candles_data(
	"Symbol", "Interval", "OpenTime", "CloseTime", "OpenPrice", "ClosePrice", "LowPrice", "HighPrice", "Volume", "QuoteVolume", "TakerVolume", "TakerQuoteVolume", "PrevCandleCloseTime", "UniqueID")
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,DEFAULT) ON CONFLICT DO NOTHING;`)

	if err != nil {
		log.Fatal(err.Error())
	}

	for i :=0; i < len(klines) - 1; i++ {

		kline := klines[i]

		_, err = stmt.Exec(kline.Symbol, 60, time.Unix(0, int64(kline.OpenTime)*int64(1000000)).UTC(), time.Unix(0, int64(kline.CloseTime)*int64(1000000)).UTC(), kline.OpenPrice, kline.ClosePrice, kline.LowPrice, kline.HighPrice, kline.BaseVolume, kline.QuoteVolume, kline.TakerBuyBaseVolume, kline.TakerBuyQuoteVolume, time.Unix(0, int64(kline.PrevCloseCandleTimestamp)*int64(1000000)).UTC())

		if err != nil {
			log.Fatal(err.Error())
		}
	}

	stmt.Close()
}
func LoadCandles(symbol string, interval uint) ([]KLine, error) {

	var rows, err = DatabaseManager.Query(`SELECT  
										 "Symbol", 
										 extract(epoch from "OpenTime")::bigint * 1000, 
										 extract(epoch from "CloseTime")::bigint * 1000, 
										 "OpenPrice", 
										 "ClosePrice", 
										 "LowPrice", 
										 "HighPrice", 
										 "Volume", 
										 "QuoteVolume", 
										 "TakerVolume", 
										 "TakerQuoteVolume", 
										 extract(epoch from "PrevCandleCloseTime")::bigint * 1000
										 FROM public.candles_data 
										 WHERE "Symbol" = $1 AND "Interval" = $2
										 ORDER BY "OpenTime" ASC`,symbol,interval)

	if err != nil {
		return nil, err
	}

	result := make([]KLine,0)

	for rows.Next() {

		kline := KLine{}

		err = rows.Scan(&kline.Symbol,
			      &kline.OpenTime,
			      &kline.CloseTime,
			      &kline.OpenPrice,
			      &kline.ClosePrice,
			      &kline.LowPrice,
			      &kline.HighPrice,
			      &kline.BaseVolume,
			      &kline.QuoteVolume,
			      &kline.TakerBuyBaseVolume,
			      &kline.TakerBuyQuoteVolume,
			      &kline.PrevCloseCandleTimestamp)

		if err != nil {

			rows.Close()
			return nil,err
		}


		result  = append(result, kline)
	}

	rows.Close()

	return result,nil



}


func OpenDatabaseConnection() error {

	var err error
	DatabaseManager, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@157.230.174.164/%s", "neronru", "TESTSHIT", "trades"))

	return err
}

func main() {


	err := OpenDatabaseConnection()

	if err != nil {

		log.Fatal("Database connection error: ", err.Error())
	}

	SaveCandles()

	rsi:= indicators.RSI{Period:14}

	klines,err := LoadCandles("BTCUSDT",60)

	prevPrevRSI, prevRSI  := -2.0, -1.0


	for idx,kline :=range klines {

		calcRSI,isNotNaN := rsi.PredictForNextPoint(kline.LowPrice)
		rsi.AddPoint(kline.ClosePrice)

		if isNotNaN {

			if prevRSI <= prevPrevRSI && prevRSI < calcRSI {
				log.Println(prevPrevRSI,prevRSI,calcRSI,klines[idx].OpenTime)
			}

			prevPrevRSI = prevRSI
			prevRSI = calcRSI
			
		}

	}




}
