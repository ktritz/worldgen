package climgen

const TradeGoodsSchemaVersion = "trade-goods/v1"

type TradeGoodsSettings struct {
	SchemaVersion string                       `json:"schemaVersion"`
	Scarcity      TradeGoodsScarcitySettings   `json:"scarcity,omitempty"`
	Multimodal    TradeGoodsMultimodalSettings `json:"multimodal,omitempty"`
	Production    TradeGoodsProductionSettings `json:"production,omitempty"`
	Demand        TradeGoodsDemandSettings     `json:"demand,omitempty"`
	Goods         []TradeGoodSpec              `json:"goods"`
}

type TradeGoodsScarcitySettings struct {
	RawAvailability         TradeGoodsRawAvailabilitySettings    `json:"rawAvailability,omitempty"`
	InputAvailability       TradeGoodsInputAvailabilitySettings  `json:"inputAvailability,omitempty"`
	CategoryAvailabilityFit map[string]TradeGoodsAvailabilityFit `json:"categoryAvailabilityFit,omitempty"`
	CategoryScarcityPower   map[string]float64                   `json:"categoryScarcityPower,omitempty"`
	DemandResponse          map[string]TradeGoodsResponseCurve   `json:"demandResponse,omitempty"`
	SupplyResponse          map[string]TradeGoodsResponseCurve   `json:"supplyResponse,omitempty"`
	TradeValueResponse      map[string]TradeGoodsResponseCurve   `json:"tradeValueResponse,omitempty"`
}

type TradeGoodsRawAvailabilitySettings struct {
	MeanWeight              float64 `json:"meanWeight,omitempty"`
	CoverageWeight          float64 `json:"coverageWeight,omitempty"`
	StrongCoverageWeight    float64 `json:"strongCoverageWeight,omitempty"`
	PeakWeight              float64 `json:"peakWeight,omitempty"`
	CoverageThreshold       float64 `json:"coverageThreshold,omitempty"`
	StrongCoverageThreshold float64 `json:"strongCoverageThreshold,omitempty"`
}

type TradeGoodsInputAvailabilitySettings struct {
	MinWeight float64 `json:"minWeight,omitempty"`
	AvgWeight float64 `json:"avgWeight,omitempty"`
}

type TradeGoodsAvailabilityFit struct {
	Offset float64 `json:"offset,omitempty"`
	Scale  float64 `json:"scale,omitempty"`
}

type TradeGoodsResponseCurve struct {
	Base  float64 `json:"base,omitempty"`
	Slope float64 `json:"slope,omitempty"`
}

type TradeGoodsMultimodalSettings struct {
	CapacityScaleByMode             map[string]float64                 `json:"capacityScaleByMode,omitempty"`
	CapacityFactorByMode            map[string]float64                 `json:"capacityFactorByMode,omitempty"`
	VolumeBaseByMode                map[string]float64                 `json:"volumeBaseByMode,omitempty"`
	EndpointNeedShareByCategory     map[string]float64                 `json:"endpointNeedShareByCategory,omitempty"`
	EndpointSurplusReliefByCategory map[string]float64                 `json:"endpointSurplusReliefByCategory,omitempty"`
	LocalNeedResponse               map[string]TradeGoodsResponseCurve `json:"localNeedResponse,omitempty"`
	GlobalRarityResponse            map[string]TradeGoodsResponseCurve `json:"globalRarityResponse,omitempty"`
	SingleMarketWealthBase          float64                            `json:"singleMarketWealthBase,omitempty"`
	SingleMarketWealthScale         float64                            `json:"singleMarketWealthScale,omitempty"`
	DualMarketWealthBase            float64                            `json:"dualMarketWealthBase,omitempty"`
	DualMarketWealthScale           float64                            `json:"dualMarketWealthScale,omitempty"`
	SingleMarketFeederScale         float64                            `json:"singleMarketFeederScale,omitempty"`
	DualMarketFeederScale           float64                            `json:"dualMarketFeederScale,omitempty"`
	LowCapacityVolumeThreshold      float64                            `json:"lowCapacityVolumeThreshold,omitempty"`
}

type TradeGoodsProductionSettings struct {
	CategorySupplyScale             map[string]float64 `json:"categorySupplyScale,omitempty"`
	CategorySpecializationScale     map[string]float64 `json:"categorySpecializationScale,omitempty"`
	ManufacturingBaseScale          map[string]float64 `json:"manufacturingBaseScale,omitempty"`
	ManufacturingWorkshopScale      map[string]float64 `json:"manufacturingWorkshopScale,omitempty"`
	MarketWorkshopBias              map[string]float64 `json:"marketWorkshopBias,omitempty"`
	MarketInputRichnessScale        map[string]float64 `json:"marketInputRichnessScale,omitempty"`
	MarketConversionShare           map[string]float64 `json:"marketConversionShare,omitempty"`
	MarketDominancePenalty          map[string]float64 `json:"marketDominancePenalty,omitempty"`
	MarketImportedInputSupportScale map[string]float64 `json:"marketImportedInputSupportScale,omitempty"`
	MarketExternalInputSupportScale map[string]float64 `json:"marketExternalInputSupportScale,omitempty"`
	MarketInputReservationByInput   map[string]float64 `json:"marketInputReservationByInput,omitempty"`
	MarketInputReservationStrength  float64            `json:"marketInputReservationStrength,omitempty"`
	MarketInputReservationCap       float64            `json:"marketInputReservationCap,omitempty"`
	RawCatchmentSpecializationScale float64            `json:"rawCatchmentSpecializationScale,omitempty"`
	RawCatchmentSpecializationFloor float64            `json:"rawCatchmentSpecializationFloor,omitempty"`
	RawPotentialPivot               float64            `json:"rawPotentialPivot,omitempty"`
	ManufacturingPivot              float64            `json:"manufacturingPivot,omitempty"`
}

type TradeGoodsDemandSettings struct {
	CategoryDemandScale         map[string]float64 `json:"categoryDemandScale,omitempty"`
	GoodDemandScale             map[string]float64 `json:"goodDemandScale,omitempty"`
	LocalSupplyReliefByCategory map[string]float64 `json:"localSupplyReliefByCategory,omitempty"`
	LocalSupplyReliefByGood     map[string]float64 `json:"localSupplyReliefByGood,omitempty"`
	DriverSpecializationScale   map[string]float64 `json:"driverSpecializationScale,omitempty"`
	MarketCategoryDemandScale   map[string]float64 `json:"marketCategoryDemandScale,omitempty"`
	MarketGoodDemandScale       map[string]float64 `json:"marketGoodDemandScale,omitempty"`
	MarketWealthPullScale       map[string]float64 `json:"marketWealthPullScale,omitempty"`
	MarketFeederPullScale       map[string]float64 `json:"marketFeederPullScale,omitempty"`
	DriverSpecializationPivot   float64            `json:"driverSpecializationPivot,omitempty"`
}

type TradeGoodSpec struct {
	Name                       string             `json:"name"`
	Category                   string             `json:"category"`
	BaseValue                  float64            `json:"baseValue"`
	Bulkiness                  float64            `json:"bulkiness"`
	Perishability              float64            `json:"perishability"`
	RawCatchmentSensitivity    float64            `json:"rawCatchmentSensitivity,omitempty"`
	MarketConversionScale      float64            `json:"marketConversionScale,omitempty"`
	MarketContextSensitivity   float64            `json:"marketContextSensitivity,omitempty"`
	MarketDominancePenalty     float64            `json:"marketDominancePenalty,omitempty"`
	MarketInputReservePriority float64            `json:"marketInputReservePriority,omitempty"`
	MarketMinNodeKind          string             `json:"marketMinNodeKind,omitempty"`
	LocalInputCapabilityFloor  map[string]float64 `json:"localInputCapabilityFloor,omitempty"`
	MarketInputCapabilityFloor map[string]float64 `json:"marketInputCapabilityFloor,omitempty"`
	SourceWeights              map[string]float64 `json:"sourceWeights,omitempty"`
	Inputs                     map[string]float64 `json:"inputs,omitempty"`
	ProductionDrivers          map[string]float64 `json:"productionDrivers,omitempty"`
	DemandDrivers              map[string]float64 `json:"demandDrivers,omitempty"`
	ProfileProductionAffinity  map[string]float64 `json:"profileProductionAffinity,omitempty"`
	ProfileDemandAffinity      map[string]float64 `json:"profileDemandAffinity,omitempty"`
}
