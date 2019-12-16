package main

import (
	"database/sql"
	"encoding/json"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/indicators"
	"github.com/NERON/tran/providers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"html/template"
	"log"
	"net/http"

	"time"
)

var DatabaseManager *sql.DB
var TemplateManager *template.Template

func SaveCandles() {
	klines := providers.GetKlines("BTCUSDT", "1h", 0, 0)

	stmt, err := DatabaseManager.Prepare(`INSERT INTO public.candles_data(
	"Symbol", "Interval", "OpenTime", "CloseTime", "OpenPrice", "ClosePrice", "LowPrice", "HighPrice", "Volume", "QuoteVolume", "TakerVolume", "TakerQuoteVolume", "PrevCandleCloseTime", "UniqueID")
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,DEFAULT) ON CONFLICT DO NOTHING;`)

	if err != nil {
		log.Fatal(err.Error())
	}

	for i := 0; i < len(klines)-1; i++ {

		kline := klines[i]

		_, err = stmt.Exec(kline.Symbol, 60, time.Unix(0, int64(kline.OpenTime)*int64(1000000)).UTC(), time.Unix(0, int64(kline.CloseTime)*int64(1000000)).UTC(), kline.OpenPrice, kline.ClosePrice, kline.LowPrice, kline.HighPrice, kline.BaseVolume, kline.QuoteVolume, kline.TakerBuyBaseVolume, kline.TakerBuyQuoteVolume, time.Unix(0, int64(kline.PrevCloseCandleTimestamp)*int64(1000000)).UTC())

		if err != nil {
			log.Fatal(err.Error())
		}
	}

	stmt.Close()
}
func LoadCandles(symbol string, interval uint) ([]candlescommon.KLine, error) {

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
										 ORDER BY "OpenTime" ASC`, symbol, interval)

	if err != nil {
		return nil, err
	}

	result := make([]candlescommon.KLine, 0)

	for rows.Next() {

		kline := candlescommon.KLine{}

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
			return nil, err
		}

		result = append(result, kline)
	}

	rows.Close()

	return result, nil

}

func IndexHandler(w http.ResponseWriter, r *http.Request) {

	TemplateManager.ExecuteTemplate(w, "chartPage.tpl", nil)
}
func ChartUpdateHandler(w http.ResponseWriter, r *http.Request) {

	candles := providers.GetKlines("BTCUSDT","1h",0,0)

	byte, _ := json.Marshal(candles)

	w.Write(byte)


}

func GetLastCandles(symbol string, interval uint, limit int) []candlescommon.KLine {

	candles, err := LoadCandles(symbol, interval)

	if err != nil {

	}

	prevCandleCloseTime := uint64(0)

	//check for candle data consistency
	for _, candle := range candles {

		if prevCandleCloseTime > 0 && prevCandleCloseTime != candle.PrevCloseCandleTimestamp {

			//find incostistency
			log.Println("incostistency found: ", prevCandleCloseTime, candle.PrevCloseCandleTimestamp)
		}

		prevCandleCloseTime = candle.CloseTime
	}

	//check limit
	if len(candles) < limit {

		log.Println("load candles from provider")
	}

	return candles
}

func InitRouting() *mux.Router {

	r := mux.NewRouter()

	r.HandleFunc("/", IndexHandler)
	r.HandleFunc("/chart/{symbol}/{interval}", ChartUpdateHandler)

	return r
}
func Test() {

	SaveCandles()

	rsi := indicators.RSI{Period: 14}

	klines, _ := LoadCandles("BTCUSDT", 60)

	prevPrevRSI, prevRSI := -2.0, -1.0

	for idx, kline := range klines {

		rsi.AddPoint(kline.LowPrice)
		calcRSI, isNotNaN := rsi.Calculate()

		if isNotNaN {

			if prevRSI <= prevPrevRSI && prevRSI <= calcRSI {
				log.Println(prevPrevRSI, prevRSI, calcRSI, klines[idx].OpenTime)
			}

			prevPrevRSI = prevRSI
			prevRSI = calcRSI

		}

	}

}
func main() {

	var err error
	TemplateManager, err = template.ParseFiles("./tran_dir/templates/chartPage.tpl")

	err = OpenDatabaseConnection()

	if err != nil {

		log.Fatal("Database connection error: ", err.Error())
	}

	router := InitRouting()

	log.Fatal(http.ListenAndServe(":8085", router))

}
