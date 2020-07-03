package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/indicators"
	"github.com/NERON/tran/providers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"html/template"
	"log"
	"math"
	"net/http"

	"time"
)

var DatabaseManager *sql.DB
var TemplateManager *template.Template

func SaveCandles() {
	klines := providers.GetKlines("ETHUSDT", "1h", 0, 0)

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
func RSIJSONHandler(w http.ResponseWriter, r *http.Request) {

	WINDOW := 24

	candles := providers.GetKlines("ETHUSDT", "1h", 0, 0)

	rsi := indicators.RSI{Period: 14}

	RSIs := make([]float64, 0)

	for _, candle := range candles {

		rsi.AddPoint(candle.ClosePrice)
		rsiVal, ok := rsi.Calculate()

		if ok {
			RSIs = append(RSIs, math.Round(rsiVal*1000)/1000)
		}
	}

	currentWindow := make([][]float64, 2)
	RSIsWindowed := make([][][]float64, 0)

	i := 0

	for ; i < WINDOW && i < len(RSIs); i++ {
		currentWindow[0] = append(currentWindow[0], RSIs[i])
	}


	for i += 1; i < len(RSIs); i++ {

		currentWindow[1] = make([]float64,1)
		currentWindow[1][0] = RSIs[i]

		RSIsWindowed = append(RSIsWindowed,currentWindow)

		currentWindowNew := make([][]float64,2)
		currentWindowNew[0] = append(currentWindowNew[0],currentWindow[0][1:]...)
		currentWindowNew[0] = append(currentWindowNew[0],RSIs[i])

		currentWindow = currentWindowNew

	}

	//output json
	byte, _ := json.Marshal(RSIsWindowed)

	w.Header().Add("Content-Disposition", "Attachment")

	http.ServeContent(w,r,"BTCUSDT.json",time.Now(),bytes.NewReader(byte))

}
func TestHandler(w http.ResponseWriter, r *http.Request) {

	rsiP := indicators.NewRSIMultiplePeriods(250)

	candles := providers.GetKlines("ETHUSDT", "1h", 0, 0)


	candlesOld := providers.GetKlines("ETHUSDT", "1h", 0, candles[0].OpenTime-1)


	for _,candleOld := range candlesOld {
		rsiP.AddPoint(candleOld.ClosePrice)

		log.Println(candleOld)

	}

	log.Println("tt",candles[0])

	lowReverse := indicators.NewRSILowReverseIndicator()
	lowsMap := make(map[int]struct{})

	for idx, candle := range candles {

		lowReverse.AddPoint(candle.LowPrice,0)

		if lowReverse.IsPreviousLow() {
			lowsMap[idx-1] = struct{}{}
		}

	}



	sequence := make([]int,0)


	for idx, candle := range candles {


		_, ok := lowsMap[idx]

		if ok {

			bestPeriod := rsiP.GetBestPeriod(candle.LowPrice,30)
			sequence = append(sequence,bestPeriod)
		}

		rsiP.AddPoint(candle.ClosePrice)

	}

	transitionMap := make(map[string]int)

	for idx, val := range sequence {

		if idx > 0 {

			transitionMap[fmt.Sprintf("%d-%d",sequence[idx-1],val)] = transitionMap[fmt.Sprintf("%d-%d",sequence[idx-1],val)] + 1
		}

		//transitionMap[fmt.Sprintf("%d",val)] = transitionMap[fmt.Sprintf("%d",val)] + 1
	}

	b, _ := json.Marshal(sequence)

	w.Write(b)




}
func ChartUpdateHandler(w http.ResponseWriter, r *http.Request) {

	type ChartUpdateCandle struct {
		OpenTime        uint64
		CloseTime       uint64
		OpenPrice       float64
		ClosePrice      float64
		LowPrice        float64
		HighPrice       float64
		IsRSIReverseLow bool
		RSIValue        float64
		RSIBestPeriod int
	}

	candles := providers.GetKlines("ETHUSDT", "1h", 0, 0)


	rsiP := indicators.NewRSIMultiplePeriods(250)

	candlesOld := providers.GetKlines("ETHUSDT", "4h", 0, candles[0].OpenTime-1)


	for _,candleOld := range candlesOld {

		rsiP.AddPoint(candleOld.ClosePrice)

		log.Println(candleOld)

	}

	log.Println(candles[0])

	updateCandles := make([]ChartUpdateCandle, 0)


	g := time.Now()

	lowReverse := indicators.NewRSILowReverseIndicator()
	lowsMap := make(map[int]struct{})

	for idx, candle := range candles {

		lowReverse.AddPoint(candle.LowPrice,0)

		if lowReverse.IsPreviousLow() {
			lowsMap[idx-1] = struct{}{}
		}

	}


	rsiRev := indicators.NewRSILowReverseIndicator()
	rsi := indicators.RSI{Period: 3}




	for idx, candle := range candles {

		rsiRev.AddPoint(candle.LowPrice, candle.ClosePrice)


		_, ok := lowsMap[idx]

		bestPeriod := 0

		if ok {

			bestPeriod = rsiP.GetBestPeriod(candle.LowPrice,30)

		}

		rsiP.AddPoint(candle.ClosePrice)


		calcRSI, _ := rsi.PredictForNextPoint(candle.LowPrice)

		updateCandles = append(updateCandles, ChartUpdateCandle{
			OpenTime:   candle.OpenTime,
			CloseTime:  candle.CloseTime,
			OpenPrice:  candle.OpenPrice,
			ClosePrice: candle.ClosePrice,
			LowPrice:   candle.LowPrice,
			HighPrice:  candle.HighPrice,
			RSIValue:   calcRSI,
			RSIBestPeriod:bestPeriod,
			IsRSIReverseLow: ok,
		})

		rsi.AddPoint(candle.ClosePrice)

	}

	log.Println(time.Since(g))
	byte, _ := json.Marshal(updateCandles)

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
	r.HandleFunc("/rsiJSON", RSIJSONHandler)
	r.HandleFunc("/test", TestHandler)

	return r
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
