package climgen

type TradeModeValue struct {
	Mode  string
	Value float64
}

type TradeGoodFlowValue struct {
	Good      string
	Score     float64
	Volume    float64
	Matched   float64
	Transport float64
	LocalNeed float64
	Rarity    float64
	MarketFit float64
}

type TradeGoodExchange struct {
	FromPolity         int
	ToPolity           int
	Mode               string
	RouteID            int
	RouteFlow          float64
	TravelCost         float64
	Value              float64
	Volume             float64
	Matched            float64
	Capacity           float64
	VolumeCapacity     float64
	AvgTransportFactor float64
	AvgLocalNeedFactor float64
	AvgRarityFactor    float64
	AvgMarketFit       float64
	Goods              []TradeGoodFlowValue
}

type TradeGoodPairFlow struct {
	FromPolity int
	ToPolity   int
	Value      float64
	Volume     float64
	Matched    float64
	Modes      []TradeModeValue
	ModeVolume []TradeModeValue
	Goods      []TradeGoodFlowValue
}

type MultimodalTradeResult struct {
	Exchanges   []TradeGoodExchange
	Pairs       []TradeGoodPairFlow
	Diagnostics MultimodalTradeDiagnostics
}

type MultimodalTradeDiagnostics struct {
	TotalScore        float64
	TotalVolume       float64
	TotalMatched      float64
	AvgCapacity       float64
	AvgVolumeCapacity float64
	AvgMarketFit      float64
	RouteCandidates   int
	RouteActive       int
	CandidateGoods    int
	AcceptedGoods     int
	NoSourceSurplus   int
	NoSinkNeed        int
	NoEndpointSupply  int
	SourceConstrained int
	NeedConstrained   int
	LowCapacity       int
	LowMarketFit      int
	LowScoreFiltered  int
}
