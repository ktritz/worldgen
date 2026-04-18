package climgen

type TradeGoodEndowment struct {
	Good      string
	Category  string
	Potential []float64
}

type TradeGoodDiagnostics struct {
	SourceFields       map[string][]float64
	AvailabilityByGood map[string]float64
	ScarcityByGood     map[string]float64
}

type TradeGoodResult struct {
	Goods       []TradeGoodEndowment
	Diagnostics *TradeGoodDiagnostics
}

type PolityGoodValue struct {
	Good  string
	Value float64
}

type PolityGoodBalance struct {
	PolityID     int
	MarketWealth float64
	Supply       map[string]float64
	Demand       map[string]float64
	Surplus      map[string]float64
	Exports      []PolityGoodValue
	Imports      []PolityGoodValue
}

type PolityGoodsResult struct {
	Balances             []PolityGoodBalance
	GlobalScarcityByGood map[string]float64
}

type TradeGoodInputs struct {
	Biome       *BiomeResult
	Vegetation  *VegetationResult
	Soils       *SoilResult
	Agriculture *AgricultureResult
	Wildlife    *WildlifeResult
	Water       *WaterResourceResult
	Coastal     *CoastalResourceResult
	Resources   *ResourceResult
	Elevation   []float64
	SeaLevel    float64
	Hydro       *HydrologyBiomeInputs
}
