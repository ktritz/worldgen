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
	FromCivilization   int
	ToCivilization     int
	Internal           bool
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

type MultimodalTradeCategoryDiagnostics struct {
	Category          string
	TotalScore        float64
	TotalVolume       float64
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

type MultimodalTradeDiagnostics struct {
	TotalScore                float64
	TotalVolume               float64
	TotalMatched              float64
	InternalScore             float64
	InternalVolume            float64
	InternalMatched           float64
	ExternalExchanges         int
	InternalExchanges         int
	AvgCapacity               float64
	AvgVolumeCapacity         float64
	AvgMarketFit              float64
	SourcePreCapMatched       float64
	SourceCapacity            float64
	SinkPreCapMatched         float64
	SinkPolityDeficitCapacity float64
	SinkEndpointCapacity      float64
	SinkEffectiveCapacity     float64
	SourceScaledKeys          int
	SinkScaledKeys            int
	SinkEndpointDominatedKeys int
	RouteCandidates           int
	RouteActive               int
	CandidateGoods            int
	AcceptedGoods             int
	NoSourceSurplus           int
	NoSinkNeed                int
	NoEndpointSupply          int
	SourceConstrained         int
	NeedConstrained           int
	LowCapacity               int
	LowMarketFit              int
	LowScoreFiltered          int
	ByCategory                map[string]MultimodalTradeCategoryDiagnostics
	ByMode                    map[string]MultimodalTradeModeDiagnostics
}

type MultimodalTradeModeDiagnostics struct {
	Mode              string
	RouteCorridors    int
	RouteCandidates   int
	RouteActive       int
	SkippedUnknown    int
	SkippedSamePolity int
	ExternalExchanges int
	InternalExchanges int
	TotalScore        float64
	TotalVolume       float64
	TotalMatched      float64
	InternalScore     float64
	InternalVolume    float64
	Capacity          float64
	VolumeCapacity    float64
	MarketFit         float64
}
