package manager

type rsiData struct {
}
type RSIPeriodManager struct {
	data map[string]map[string]rsiData
}

func (periodManager *RSIPeriodManager) initializeData(symbol string) {

}

func (periodManager *RSIPeriodManager) GetTimeframesForPrice(symbol string, MaxPrice float64, MinPrice float64) {

}

func NewRSIPeriodManager() {

}
