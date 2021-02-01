package providers

import (
	"encoding/json"
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"github.com/gorilla/websocket"
	"log"
	"sync"
)

type WsKlineEvent struct {
	Event  string  `json:"e"`
	Time   int64   `json:"E"`
	Symbol string  `json:"s"`
	Kline  WsKline `json:"k"`
}

type WsKline struct {
	StartTime            int64  `json:"t"`
	EndTime              int64  `json:"T"`
	Symbol               string `json:"s"`
	Interval             string `json:"i"`
	FirstTradeID         int64  `json:"f"`
	LastTradeID          int64  `json:"L"`
	Open                 string `json:"o"`
	Close                string `json:"c"`
	High                 string `json:"h"`
	Low                  string `json:"l"`
	Volume               string `json:"v"`
	TradeNum             int64  `json:"n"`
	IsFinal              bool   `json:"x"`
	QuoteVolume          string `json:"q"`
	ActiveBuyVolume      string `json:"V"`
	ActiveBuyQuoteVolume string `json:"Q"`
}

type BinanceWebsocketProvider struct {
}

func (p *BinanceWebsocketProvider) Start(streams []string) error {

	c, _, err := websocket.DefaultDialer.Dial("wss://stream.binance.com:9443/ws", nil)

	if err != nil {
		return err
	}

	type WSMethod struct {
		Method string   `json:"method"`
		Params []string `json:"params"`
		ID     uint64   `json:"id"`
	}

	subscribe := WSMethod{Method: "SUBSCRIBE", ID: 1}

	for _, stream := range streams {

		subscribe.Params = append(subscribe.Params, fmt.Sprintf("%s@kline_5m", stream))
	}

	err = c.WriteJSON(subscribe)

	if err != nil {
		return err
	}

	go func() {

		messageID := uint64(0)

		for {

			_, message, err := c.ReadMessage()

			if err != nil {
				log.Println(err.Error())
			}

			//skip first message
			if messageID > 0 {

				klineEvent := WsKlineEvent{}

				err = json.Unmarshal(message, &klineEvent)

				if err != nil {
					log.Fatal(err.Error())
				}

				log.Println(klineEvent)

			}

			messageID++

		}

	}()

	return nil

}
func (p *BinanceWebsocketProvider) SetHandler() error {

	return nil
}
func NewBinanceWebSocketProvider() *BinanceWebsocketProvider {

	return &BinanceWebsocketProvider{}
}

type symbolKlines struct {
	symbolName string

	archivedKlines []candlescommon.KLine
	activeKline    candlescommon.KLine

	archiveFilled bool

	loadCompleted chan struct{}

	mu *sync.RWMutex
}

func (s *symbolKlines) FillCache() {

	//try to lock access to structure
	s.mu.Lock()

	//check if archive already filled
	if s.archiveFilled {
		//release mutex
		s.mu.Unlock()
		//no need to wait
		return
	}

	//check if another goroutine already tr to fill cache
	if s.loadCompleted == nil {
		//init channel
		s.loadCompleted = make(chan struct{})

		//start loading process
		go s.loadProcedure()
	}

	//copy load channel
	loadChannel := s.loadCompleted

	//release mutex
	s.mu.Unlock()

	//wait until waiting goroutine finish
	<-loadChannel

}

func (s *symbolKlines) loadProcedure() {

	//loop variable
	success := false

	//iterate while not receive success
	for !success {

		klines, err := GetLastKlines(s.symbolName, "1m")

		if err != nil {

			log.Println("Error while try to fetch archive data: ", err.Error())
			continue
		}

		var oldKlines []candlescommon.KLine

		for len(klines) < 1440 {

			oldKlines, err = GetKlinesNew(s.symbolName, "1m", GetKlineRange{Direction: 0, FromTimestamp: klines[0].OpenTime})

			if err != nil {
				break
			}

			klines = append(oldKlines, klines...)

		}

		if err != nil {
			log.Println("Error in fetching old data: ", err.Error())
			continue
		}

		//go to protected zone
		s.mu.Lock()

		//we have different situations based on current kline value
		if s.activeKline.OpenTime == 0 {

			//if current kline is not set,set it as last kline
			s.activeKline = klines[len(klines)-1]

		}

		//if we have equal ends replace
		if s.activeKline.OpenTime == klines[len(klines)-1].OpenTime {

			//set current archived klines
			s.archivedKlines = klines[:len(klines)-1]

			//set archive filled to true
			s.archiveFilled = true

			//set success to true
			success = true
		}

		//release mutex
		s.mu.Unlock()

	}

	//try to lock access to structure
	s.mu.Lock()

	//copy load channel
	loadChannel := s.loadCompleted

	//set to nil load channel
	s.loadCompleted = nil

	//release mutex
	s.mu.Unlock()

	//close old channel to notify other that we are done
	close(loadChannel)

}

func NewSymbolKLines(symbol string) *symbolKlines {

	return &symbolKlines{mu: &sync.RWMutex{}, symbolName: symbol}
}

type LastKlinesCaches struct {
	symbols map[string]symbolKlines
}
