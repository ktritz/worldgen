package climgen

func DefaultTradeGoodsScarcitySettings() TradeGoodsScarcitySettings {
	return TradeGoodsScarcitySettings{
		RawAvailability: TradeGoodsRawAvailabilitySettings{
			MeanWeight:              0.34,
			CoverageWeight:          0.28,
			StrongCoverageWeight:    0.18,
			PeakWeight:              0.20,
			CoverageThreshold:       0.10,
			StrongCoverageThreshold: 0.30,
		},
		InputAvailability: TradeGoodsInputAvailabilitySettings{
			MinWeight: 0.42,
			AvgWeight: 0.58,
		},
		CategoryAvailabilityFit: map[string]TradeGoodsAvailabilityFit{
			"processed": {Offset: 0.04, Scale: 0.96},
			"finished":  {Offset: 0.02, Scale: 0.92},
			"luxury":    {Offset: 0.00, Scale: 0.82},
			"strategic": {Offset: 0.00, Scale: 0.88},
		},
		CategoryScarcityPower: map[string]float64{
			"raw":       0.50,
			"processed": 0.60,
			"finished":  0.55,
			"luxury":    0.48,
			"strategic": 0.52,
			"default":   0.58,
		},
		DemandResponse: map[string]TradeGoodsResponseCurve{
			"raw":       {Base: 0.28, Slope: 0.42},
			"processed": {Base: 0.20, Slope: 0.54},
			"finished":  {Base: 0.18, Slope: 0.62},
			"luxury":    {Base: 0.12, Slope: 0.48},
			"strategic": {Base: 0.20, Slope: 0.72},
			"default":   {Base: 0.20, Slope: 0.50},
		},
		SupplyResponse: map[string]TradeGoodsResponseCurve{
			"raw":       {Base: 0.94, Slope: 0.10},
			"processed": {Base: 0.94, Slope: 0.18},
			"finished":  {Base: 0.96, Slope: 0.20},
			"luxury":    {Base: 0.92, Slope: 0.16},
			"strategic": {Base: 0.96, Slope: 0.22},
			"default":   {Base: 0.94, Slope: 0.16},
		},
		TradeValueResponse: map[string]TradeGoodsResponseCurve{
			"raw":       {Base: 0.86, Slope: 0.36},
			"processed": {Base: 0.90, Slope: 0.42},
			"finished":  {Base: 0.92, Slope: 0.46},
			"luxury":    {Base: 0.96, Slope: 0.34},
			"strategic": {Base: 0.96, Slope: 0.48},
			"default":   {Base: 0.90, Slope: 0.40},
		},
	}
}

func DefaultTradeGoodsMultimodalSettings() TradeGoodsMultimodalSettings {
	return TradeGoodsMultimodalSettings{
		CapacityScaleByMode: map[string]float64{
			"land":    17.0,
			"river":   23.0,
			"coastal": 29.0,
			"ocean":   38.0,
			"default": 20.0,
		},
		CapacityFactorByMode: map[string]float64{
			"land":    0.88,
			"river":   1.04,
			"coastal": 1.12,
			"ocean":   1.20,
			"default": 1.00,
		},
		VolumeBaseByMode: map[string]float64{
			"land":    16.0,
			"river":   22.0,
			"coastal": 26.0,
			"ocean":   32.0,
			"default": 20.0,
		},
		EndpointNeedShareByCategory: map[string]float64{
			"default": 0.0,
		},
		LocalNeedResponse: map[string]TradeGoodsResponseCurve{
			"raw":       {Base: 0.74, Slope: 0.48},
			"processed": {Base: 0.80, Slope: 0.56},
			"finished":  {Base: 0.84, Slope: 0.64},
			"luxury":    {Base: 0.70, Slope: 0.42},
			"strategic": {Base: 0.84, Slope: 0.70},
			"default":   {Base: 0.78, Slope: 0.50},
		},
		GlobalRarityResponse: map[string]TradeGoodsResponseCurve{
			"raw":       {Base: 0.92, Slope: 0.16},
			"processed": {Base: 0.92, Slope: 0.20},
			"finished":  {Base: 0.94, Slope: 0.24},
			"luxury":    {Base: 0.96, Slope: 0.28},
			"strategic": {Base: 0.96, Slope: 0.30},
			"default":   {Base: 0.93, Slope: 0.20},
		},
		SingleMarketWealthBase:     0.35,
		SingleMarketWealthScale:    0.70,
		DualMarketWealthBase:       0.35,
		DualMarketWealthScale:      0.85,
		SingleMarketFeederScale:    0.05,
		DualMarketFeederScale:      0.06,
		LowCapacityVolumeThreshold: 2.5,
	}
}

func DefaultTradeGoodsProductionSettings() TradeGoodsProductionSettings {
	return TradeGoodsProductionSettings{
		CategorySupplyScale: map[string]float64{
			"raw":       1.12,
			"processed": 1.06,
			"finished":  1.04,
			"luxury":    1.00,
			"strategic": 1.08,
			"default":   1.00,
		},
		CategorySpecializationScale: map[string]float64{
			"raw":       0.70,
			"processed": 0.42,
			"finished":  0.34,
			"luxury":    0.28,
			"strategic": 0.38,
			"default":   0.30,
		},
		ManufacturingBaseScale: map[string]float64{
			"raw":       0.30,
			"processed": 0.50,
			"finished":  0.46,
			"luxury":    0.38,
			"strategic": 0.48,
			"default":   0.45,
		},
		ManufacturingWorkshopScale: map[string]float64{
			"raw":       0.20,
			"processed": 0.72,
			"finished":  0.86,
			"luxury":    0.94,
			"strategic": 0.88,
			"default":   0.55,
		},
		MarketWorkshopBias: map[string]float64{
			"raw":       0.84,
			"processed": 1.10,
			"finished":  1.22,
			"luxury":    1.18,
			"strategic": 1.20,
			"default":   1.00,
		},
		MarketInputRichnessScale: map[string]float64{
			"raw":       0.04,
			"processed": 0.18,
			"finished":  0.28,
			"luxury":    0.24,
			"strategic": 0.26,
			"default":   0.14,
		},
		MarketConversionShare: map[string]float64{
			"raw":       0.08,
			"processed": 0.62,
			"finished":  0.54,
			"luxury":    0.42,
			"strategic": 0.50,
			"default":   0.40,
		},
		MarketDominancePenalty: map[string]float64{
			"raw":       0.00,
			"processed": 0.14,
			"finished":  0.08,
			"luxury":    0.06,
			"strategic": 0.10,
			"default":   0.10,
		},
		MarketImportedInputSupportScale: map[string]float64{
			"default": 0.0,
		},
		MarketExternalInputSupportScale: map[string]float64{
			"default": 0.0,
		},
		MarketInputReservationByInput: map[string]float64{
			"default": 0.0,
		},
		MarketInputReservationStrength:  0.24,
		MarketInputReservationCap:       0.30,
		RawCatchmentSpecializationScale: 0.62,
		RawCatchmentSpecializationFloor: 0.68,
		RawPotentialPivot:               0.36,
		ManufacturingPivot:              0.48,
	}
}

func DefaultTradeGoodsDemandSettings() TradeGoodsDemandSettings {
	return TradeGoodsDemandSettings{
		CategoryDemandScale: map[string]float64{
			"raw":       0.94,
			"processed": 1.00,
			"finished":  1.04,
			"luxury":    0.96,
			"strategic": 1.02,
			"default":   1.00,
		},
		LocalSupplyReliefByCategory: map[string]float64{
			"raw":       0.48,
			"processed": 0.26,
			"finished":  0.16,
			"luxury":    0.06,
			"strategic": 0.12,
			"default":   0.18,
		},
		DriverSpecializationScale: map[string]float64{
			"raw":       0.36,
			"processed": 0.44,
			"finished":  0.52,
			"luxury":    0.58,
			"strategic": 0.54,
			"default":   0.42,
		},
		MarketCategoryDemandScale: map[string]float64{
			"raw":       0.96,
			"processed": 1.04,
			"finished":  1.10,
			"luxury":    1.08,
			"strategic": 1.06,
			"default":   1.00,
		},
		MarketWealthPullScale: map[string]float64{
			"raw":       0.06,
			"processed": 0.12,
			"finished":  0.22,
			"luxury":    0.26,
			"strategic": 0.18,
			"default":   0.10,
		},
		MarketFeederPullScale: map[string]float64{
			"raw":       0.04,
			"processed": 0.10,
			"finished":  0.16,
			"luxury":    0.12,
			"strategic": 0.14,
			"default":   0.08,
		},
		DriverSpecializationPivot: 0.44,
	}
}
