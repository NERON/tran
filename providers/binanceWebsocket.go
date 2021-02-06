package providers

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
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
	handler func(messageID uint64, kline WsKline)
}

func (p *BinanceWebsocketProvider) Start(symbols []string, intervals []string) error {

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

	for _, stream := range symbols {

		for _, interval := range intervals {

			subscribe.Params = append(subscribe.Params, fmt.Sprintf("%s@kline_%s", stream, interval))
		}

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

				p.handler(messageID, klineEvent.Kline)

			}

			messageID++

		}

	}()

	return nil

}

func NewBinanceWebSocketProvider(Handler func(messageID uint64, kline WsKline)) *BinanceWebsocketProvider {

	return &BinanceWebsocketProvider{
		handler: Handler,
	}
}
