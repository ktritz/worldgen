package climgen

func (settings TradeGoodsSettings) withDefaults() TradeGoodsSettings {
	settings.Scarcity = settings.Scarcity.withDefaults()
	settings.Multimodal = settings.Multimodal.withDefaults()
	settings.Production = settings.Production.withDefaults()
	settings.Demand = settings.Demand.withDefaults()
	return settings
}

func (settings TradeGoodsScarcitySettings) withDefaults() TradeGoodsScarcitySettings {
	defaults := DefaultTradeGoodsScarcitySettings()
	if settings.RawAvailability.MeanWeight > 0 {
		defaults.RawAvailability.MeanWeight = settings.RawAvailability.MeanWeight
	}
	if settings.RawAvailability.CoverageWeight > 0 {
		defaults.RawAvailability.CoverageWeight = settings.RawAvailability.CoverageWeight
	}
	if settings.RawAvailability.StrongCoverageWeight > 0 {
		defaults.RawAvailability.StrongCoverageWeight = settings.RawAvailability.StrongCoverageWeight
	}
	if settings.RawAvailability.PeakWeight > 0 {
		defaults.RawAvailability.PeakWeight = settings.RawAvailability.PeakWeight
	}
	if settings.RawAvailability.CoverageThreshold > 0 {
		defaults.RawAvailability.CoverageThreshold = settings.RawAvailability.CoverageThreshold
	}
	if settings.RawAvailability.StrongCoverageThreshold > 0 {
		defaults.RawAvailability.StrongCoverageThreshold = settings.RawAvailability.StrongCoverageThreshold
	}
	if settings.InputAvailability.MinWeight > 0 {
		defaults.InputAvailability.MinWeight = settings.InputAvailability.MinWeight
	}
	if settings.InputAvailability.AvgWeight > 0 {
		defaults.InputAvailability.AvgWeight = settings.InputAvailability.AvgWeight
	}
	for category, fit := range settings.CategoryAvailabilityFit {
		if fit.Scale <= 0 && fit.Offset == 0 {
			continue
		}
		defaults.CategoryAvailabilityFit[category] = fit
	}
	for category, power := range settings.CategoryScarcityPower {
		if power <= 0 {
			continue
		}
		defaults.CategoryScarcityPower[category] = power
	}
	for category, curve := range settings.DemandResponse {
		if curve.Base == 0 && curve.Slope == 0 {
			continue
		}
		defaults.DemandResponse[category] = curve
	}
	for category, curve := range settings.SupplyResponse {
		if curve.Base == 0 && curve.Slope == 0 {
			continue
		}
		defaults.SupplyResponse[category] = curve
	}
	for category, curve := range settings.TradeValueResponse {
		if curve.Base == 0 && curve.Slope == 0 {
			continue
		}
		defaults.TradeValueResponse[category] = curve
	}
	return defaults
}

func (settings TradeGoodsMultimodalSettings) withDefaults() TradeGoodsMultimodalSettings {
	defaults := DefaultTradeGoodsMultimodalSettings()
	for mode, value := range settings.CapacityScaleByMode {
		if value > 0 {
			defaults.CapacityScaleByMode[mode] = value
		}
	}
	for mode, value := range settings.CapacityFactorByMode {
		if value > 0 {
			defaults.CapacityFactorByMode[mode] = value
		}
	}
	for mode, value := range settings.VolumeBaseByMode {
		if value > 0 {
			defaults.VolumeBaseByMode[mode] = value
		}
	}
	for category, curve := range settings.LocalNeedResponse {
		if curve.Base == 0 && curve.Slope == 0 {
			continue
		}
		defaults.LocalNeedResponse[category] = curve
	}
	for category, curve := range settings.GlobalRarityResponse {
		if curve.Base == 0 && curve.Slope == 0 {
			continue
		}
		defaults.GlobalRarityResponse[category] = curve
	}
	if settings.SingleMarketWealthBase > 0 {
		defaults.SingleMarketWealthBase = settings.SingleMarketWealthBase
	}
	if settings.SingleMarketWealthScale > 0 {
		defaults.SingleMarketWealthScale = settings.SingleMarketWealthScale
	}
	if settings.DualMarketWealthBase > 0 {
		defaults.DualMarketWealthBase = settings.DualMarketWealthBase
	}
	if settings.DualMarketWealthScale > 0 {
		defaults.DualMarketWealthScale = settings.DualMarketWealthScale
	}
	if settings.SingleMarketFeederScale > 0 {
		defaults.SingleMarketFeederScale = settings.SingleMarketFeederScale
	}
	if settings.DualMarketFeederScale > 0 {
		defaults.DualMarketFeederScale = settings.DualMarketFeederScale
	}
	if settings.LowCapacityVolumeThreshold > 0 {
		defaults.LowCapacityVolumeThreshold = settings.LowCapacityVolumeThreshold
	}
	return defaults
}

func (settings TradeGoodsProductionSettings) withDefaults() TradeGoodsProductionSettings {
	defaults := DefaultTradeGoodsProductionSettings()
	for category, value := range settings.CategorySupplyScale {
		if value > 0 {
			defaults.CategorySupplyScale[category] = value
		}
	}
	for category, value := range settings.CategorySpecializationScale {
		if value >= 0 {
			defaults.CategorySpecializationScale[category] = value
		}
	}
	for category, value := range settings.ManufacturingBaseScale {
		if value >= 0 {
			defaults.ManufacturingBaseScale[category] = value
		}
	}
	for category, value := range settings.ManufacturingWorkshopScale {
		if value >= 0 {
			defaults.ManufacturingWorkshopScale[category] = value
		}
	}
	for category, value := range settings.MarketWorkshopBias {
		if value >= 0 {
			defaults.MarketWorkshopBias[category] = value
		}
	}
	for category, value := range settings.MarketInputRichnessScale {
		if value >= 0 {
			defaults.MarketInputRichnessScale[category] = value
		}
	}
	for category, value := range settings.MarketConversionShare {
		if value >= 0 {
			defaults.MarketConversionShare[category] = value
		}
	}
	for category, value := range settings.MarketDominancePenalty {
		if value >= 0 {
			defaults.MarketDominancePenalty[category] = value
		}
	}
	for category, value := range settings.MarketImportedInputSupportScale {
		if value >= 0 {
			defaults.MarketImportedInputSupportScale[category] = value
		}
	}
	for category, value := range settings.MarketExternalInputSupportScale {
		if value >= 0 {
			defaults.MarketExternalInputSupportScale[category] = value
		}
	}
	for input, value := range settings.MarketInputReservationByInput {
		if value >= 0 {
			defaults.MarketInputReservationByInput[input] = value
		}
	}
	if settings.MarketInputReservationStrength > 0 {
		defaults.MarketInputReservationStrength = settings.MarketInputReservationStrength
	}
	if settings.MarketInputReservationCap > 0 {
		defaults.MarketInputReservationCap = settings.MarketInputReservationCap
	}
	if settings.RawCatchmentSpecializationScale > 0 {
		defaults.RawCatchmentSpecializationScale = settings.RawCatchmentSpecializationScale
	}
	if settings.RawCatchmentSpecializationFloor > 0 {
		defaults.RawCatchmentSpecializationFloor = settings.RawCatchmentSpecializationFloor
	}
	if settings.RawPotentialPivot > 0 {
		defaults.RawPotentialPivot = settings.RawPotentialPivot
	}
	if settings.ManufacturingPivot > 0 {
		defaults.ManufacturingPivot = settings.ManufacturingPivot
	}
	return defaults
}

func (settings TradeGoodsDemandSettings) withDefaults() TradeGoodsDemandSettings {
	defaults := DefaultTradeGoodsDemandSettings()
	for category, value := range settings.CategoryDemandScale {
		if value > 0 {
			defaults.CategoryDemandScale[category] = value
		}
	}
	for category, value := range settings.LocalSupplyReliefByCategory {
		if value >= 0 {
			defaults.LocalSupplyReliefByCategory[category] = value
		}
	}
	for category, value := range settings.DriverSpecializationScale {
		if value >= 0 {
			defaults.DriverSpecializationScale[category] = value
		}
	}
	for category, value := range settings.MarketCategoryDemandScale {
		if value > 0 {
			defaults.MarketCategoryDemandScale[category] = value
		}
	}
	for category, value := range settings.MarketWealthPullScale {
		if value >= 0 {
			defaults.MarketWealthPullScale[category] = value
		}
	}
	for category, value := range settings.MarketFeederPullScale {
		if value >= 0 {
			defaults.MarketFeederPullScale[category] = value
		}
	}
	if settings.DriverSpecializationPivot > 0 {
		defaults.DriverSpecializationPivot = settings.DriverSpecializationPivot
	}
	return defaults
}
