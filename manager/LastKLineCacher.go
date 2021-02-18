package manager

import (
	"fmt"
	"github.com/NERON/tran/candlescommon"
	"github.com/NERON/tran/providers"
	"log"
	"strconv"
	"strings"
	"sync"
)

var KLineCacher *LastKlinesCaches

type symbolKlines struct {
	symbolName        string
	intervalTimeframe string
	archiveLength     uint

	archivedKlines []candlescommon.KLine
	activeKline    candlescommon.KLine

	archiveFilled bool

	loadCompleted chan struct{}

	mu *sync.RWMutex
}

func (s *symbolKlines) SetActiveKline(kline candlescommon.KLine) {

	//try to lock access to structure
	s.mu.Lock()

	//check if equal open time or it first set
	if s.activeKline.OpenTime == kline.OpenTime || s.activeKline.OpenTime == 0 {

		//copy prev candle close timestamp
		kline.PrevCloseCandleTimestamp = s.activeKline.PrevCloseCandleTimestamp

		//set current kline
		s.activeKline = kline

	} else {

		if s.activeKline.CloseTime+1 != kline.OpenTime {

			s.activeKline.PrevCloseCandleTimestamp = 0
			log.Println("Wrong time ", s.activeKline, kline)
		}

		//if don't receive message about kline closing or not all fields filled, we have inconsistency
		if s.activeKline.Closed != true || s.activeKline.PrevCloseCandleTimestamp == 0 {

			log.Println("Missed prev", s.activeKline, len(s.archivedKlines), s.intervalTimeframe)

			//set archive data to null
			s.archivedKlines = nil

			//set flag that archive corrupted
			s.archiveFilled = false

		} else {

			//append kline to archive
			s.archivedKlines = append(s.archivedKlines, s.activeKline)

			if len(s.archivedKlines) > int(s.archiveLength) {
				s.archivedKlines = s.archivedKlines[1:]
			}

		}

		//copy prev close time
		kline.PrevCloseCandleTimestamp = s.activeKline.CloseTime
		//set new kline
		s.activeKline = kline

	}

	//unlock resource
	s.mu.Unlock()
}
func (s *symbolKlines) GetData() []candlescommon.KLine {

	//create result
	result := make([]candlescommon.KLine, 0)

	for {

		//try to lock access to structure
		s.mu.Lock()

		//check if archive filled and copy
		if s.archiveFilled {

			//copy archive
			result = append(result, s.archivedKlines...)

			//copy active kline
			result = append(result, s.activeKline)

			//set closed to false
			result[len(result)-1].Closed = false

		}

		//unlock resource
		s.mu.Unlock()

		//check if result was filled
		if len(result) == 0 {

			s.FillCache()
		} else {
			break
		}

	}

	return result
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

		klines, err := providers.GetLastKlines(s.symbolName, s.intervalTimeframe)

		if err != nil {

			log.Println("Error while try to fetch archive data: ", err.Error())
			continue
		}

		var oldKlines []candlescommon.KLine

		for len(klines) < int(s.archiveLength) {

			oldKlines, err = providers.GetKlinesNew(s.symbolName, s.intervalTimeframe, providers.GetKlineRange{Direction: 0, FromTimestamp: klines[len(klines)-1].OpenTime})

			if err != nil {
				break
			}

			klines = append(klines, oldKlines...)

		}

		if err != nil {
			log.Println("Error in fetching old data: ", err.Error())
			continue
		}

		for i := 0; i < len(klines)/2; i++ {
			j := len(klines) - i - 1
			klines[i], klines[j] = klines[j], klines[i]
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

			//set prev closed
			s.activeKline.PrevCloseCandleTimestamp = klines[len(klines)-1].PrevCloseCandleTimestamp

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

func newSymbolKLines(symbol string, timeframe string, archiveLength uint) *symbolKlines {

	return &symbolKlines{mu: &sync.RWMutex{}, symbolName: symbol, intervalTimeframe: timeframe, archiveLength: archiveLength}

}

type LastKlinesCaches struct {
	symbols map[string]map[string]*symbolKlines
	ws      *providers.BinanceWebsocketProvider
}

func toFloat(value string) float64 {

	val, err := strconv.ParseFloat(value, 64)

	if err != nil {

		log.Fatal(err.Error())
	}

	return val
}
func (s *LastKlinesCaches) GetLatestKLines(symbol string, interval candlescommon.Interval) ([]candlescommon.KLine, bool) {

	klineCacher, ok := s.symbols[fmt.Sprintf("1%s", interval.Letter)][symbol]

	if !ok {
		return nil, false
	}

	klineData := klineCacher.GetData()

	for i := 0; i < len(klineData)/2; i++ {
		j := len(klineData) - i - 1
		klineData[i], klineData[j] = klineData[j], klineData[i]
	}

	if interval.Letter == "h" {
		klineData = candlescommon.HoursGroupKlineDesc(klineData, uint64(interval.Duration), true)
	} else if interval.Letter == "m" {
		klineData = candlescommon.MinutesGroupKlineDesc(klineData, uint64(interval.Duration), true)
	}

	for i := 0; i < len(klineData)/2; i++ {
		j := len(klineData) - i - 1
		klineData[i], klineData[j] = klineData[j], klineData[i]
	}

	return klineData, true

}

func NewLastKlinesCacher(symbols []string) (*LastKlinesCaches, error) {

	klines := &LastKlinesCaches{
		symbols: make(map[string]map[string]*symbolKlines),
	}

	archiveLengths := []uint{1500, 50}

	for idx, interval := range []string{"1m", "1h"} {

		klines.symbols[interval] = make(map[string]*symbolKlines)

		for _, symbol := range symbols {

			klines.symbols[interval][symbol] = newSymbolKLines(symbol, interval, archiveLengths[idx])
		}

	}

	klines.ws = providers.NewBinanceWebSocketProvider(func(messageID uint64, wsKline providers.WsKline) {

		IntervalCacher, ok := klines.symbols[wsKline.Interval]

		if !ok {
			log.Println("Interval not exist ", wsKline.Symbol)
			return
		}

		klineCacher, ok := IntervalCacher[wsKline.Symbol]

		if !ok {
			log.Println("Interval not exist ", wsKline.Symbol)
			return
		}

		kline := candlescommon.KLine{
			Symbol:              wsKline.Symbol,
			OpenTime:            uint64(wsKline.StartTime),
			CloseTime:           uint64(wsKline.EndTime),
			OpenPrice:           toFloat(wsKline.Open),
			LowPrice:            toFloat(wsKline.Low),
			HighPrice:           toFloat(wsKline.High),
			ClosePrice:          toFloat(wsKline.Close),
			Closed:              wsKline.IsFinal,
			QuoteVolume:         toFloat(wsKline.QuoteVolume),
			BaseVolume:          toFloat(wsKline.Volume),
			TakerBuyQuoteVolume: toFloat(wsKline.ActiveBuyQuoteVolume),
			TakerBuyBaseVolume:  toFloat(wsKline.ActiveBuyVolume),
		}

		klineCacher.SetActiveKline(kline)

	})

	for idx, str := range symbols {

		symbols[idx] = strings.ToLower(str)
	}

	err := klines.ws.Start(symbols, []string{"1m", "1h"})

	if err != nil {
		return nil, err
	}
	return klines, nil
}
