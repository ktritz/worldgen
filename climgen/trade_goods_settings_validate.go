package climgen

import "fmt"

// ValidateTradeGoodsSettings checks everything that is *locally* decidable from a
// single settings document: required fields, value ranges, malformed entries.
//
// It deliberately does NOT check cross-good relationships (input names, cycles,
// declaration order). Trade goods settings are loaded per document and may be
// merged, so any one document can legitimately be a partial catalog whose
// consumers' inputs are declared elsewhere. Cross-good rules are checked by
// ValidateTradeGoodsCatalog on the fully assembled catalog instead.
func ValidateTradeGoodsSettings(settings TradeGoodsSettings) error {
	if settings.SchemaVersion == "" {
		return fmt.Errorf("trade goods schemaVersion is required")
	}
	if settings.SchemaVersion != TradeGoodsSchemaVersion {
		return fmt.Errorf("unsupported trade goods schemaVersion %q", settings.SchemaVersion)
	}
	if len(settings.Goods) == 0 {
		return fmt.Errorf("trade goods catalog requires at least one good")
	}
	seen := map[string]struct{}{}
	for i, good := range settings.Goods {
		if good.Name == "" {
			return fmt.Errorf("trade goods[%d].name is required", i)
		}
		if _, ok := seen[good.Name]; ok {
			return fmt.Errorf("duplicate trade good %q", good.Name)
		}
		seen[good.Name] = struct{}{}
		if good.Category == "" {
			return fmt.Errorf("trade good %q category is required", good.Name)
		}
		if good.BaseValue < 0 {
			return fmt.Errorf("trade good %q baseValue cannot be negative", good.Name)
		}
		if good.Bulkiness < 0 || good.Bulkiness > 1 {
			return fmt.Errorf("trade good %q bulkiness must be in [0,1]", good.Name)
		}
		if good.Perishability < 0 || good.Perishability > 1 {
			return fmt.Errorf("trade good %q perishability must be in [0,1]", good.Name)
		}
		if good.RawCatchmentSensitivity < 0 || good.RawCatchmentSensitivity > 3 {
			return fmt.Errorf("trade good %q rawCatchmentSensitivity must be in [0,3]", good.Name)
		}
		if good.MarketConversionScale < 0 || good.MarketConversionScale > 2 {
			return fmt.Errorf("trade good %q marketConversionScale must be in [0,2]", good.Name)
		}
		if good.MarketContextSensitivity < 0 || good.MarketContextSensitivity > 3 {
			return fmt.Errorf("trade good %q marketContextSensitivity must be in [0,3]", good.Name)
		}
		if good.MarketDominancePenalty < 0 || good.MarketDominancePenalty > 3 {
			return fmt.Errorf("trade good %q marketDominancePenalty must be in [0,3]", good.Name)
		}
		if good.MarketInputReservePriority < 0 || good.MarketInputReservePriority > 2 {
			return fmt.Errorf("trade good %q marketInputReservePriority must be in [0,2]", good.Name)
		}
		if good.MarketMinNodeKind != "" {
			if _, ok := tradeGoodsMarketMinNodeKind(good); !ok {
				return fmt.Errorf("trade good %q marketMinNodeKind %q is invalid", good.Name, good.MarketMinNodeKind)
			}
		}
		for input, floor := range good.LocalInputCapabilityFloor {
			if input == "" {
				return fmt.Errorf("trade good %q has empty local input capability floor key", good.Name)
			}
			if floor < 0 || floor > 1 {
				return fmt.Errorf("trade good %q localInputCapabilityFloor[%q] must be in [0,1]", good.Name, input)
			}
			if _, ok := good.Inputs[input]; !ok {
				return fmt.Errorf("trade good %q localInputCapabilityFloor[%q] requires matching input", good.Name, input)
			}
		}
		for input, floor := range good.MarketInputCapabilityFloor {
			if input == "" {
				return fmt.Errorf("trade good %q has empty market input capability floor key", good.Name)
			}
			if floor < 0 || floor > 1 {
				return fmt.Errorf("trade good %q marketInputCapabilityFloor[%q] must be in [0,1]", good.Name, input)
			}
			if _, ok := good.Inputs[input]; !ok {
				return fmt.Errorf("trade good %q marketInputCapabilityFloor[%q] requires matching input", good.Name, input)
			}
		}
		for key, value := range good.SourceWeights {
			if key == "" {
				return fmt.Errorf("trade good %q has empty source weight key", good.Name)
			}
			if value < 0 {
				return fmt.Errorf("trade good %q source weight %q cannot be negative", good.Name, key)
			}
		}
		for input, amount := range good.Inputs {
			if input == "" {
				return fmt.Errorf("trade good %q has empty input key", good.Name)
			}
			if amount <= 0 {
				return fmt.Errorf("trade good %q input %q must be positive", good.Name, input)
			}
		}
		for key, value := range good.ProductionDrivers {
			if key == "" {
				return fmt.Errorf("trade good %q has empty production driver key", good.Name)
			}
			if value < 0 {
				return fmt.Errorf("trade good %q production driver %q cannot be negative", good.Name, key)
			}
		}
	}
	if err := validateTradeGoodsScarcitySettings(settings.Scarcity); err != nil {
		return err
	}
	if err := validateTradeGoodsMultimodalSettings(settings.Multimodal); err != nil {
		return err
	}
	if err := validateTradeGoodsProductionSettings(settings.Production); err != nil {
		return err
	}
	if err := validateTradeGoodsDemandSettings(settings.Demand); err != nil {
		return err
	}
	return nil
}

func validateTradeGoodsScarcitySettings(settings TradeGoodsScarcitySettings) error {
	raw := settings.RawAvailability
	if raw.MeanWeight < 0 || raw.CoverageWeight < 0 || raw.StrongCoverageWeight < 0 || raw.PeakWeight < 0 {
		return fmt.Errorf("trade goods scarcity raw availability weights must be non-negative")
	}
	if raw.MeanWeight+raw.CoverageWeight+raw.StrongCoverageWeight+raw.PeakWeight <= 0 {
		return fmt.Errorf("trade goods scarcity raw availability weights must sum to > 0")
	}
	if raw.CoverageThreshold < 0 || raw.CoverageThreshold > 1 {
		return fmt.Errorf("trade goods scarcity raw coverageThreshold must be in [0,1]")
	}
	if raw.StrongCoverageThreshold < 0 || raw.StrongCoverageThreshold > 1 {
		return fmt.Errorf("trade goods scarcity raw strongCoverageThreshold must be in [0,1]")
	}
	if raw.StrongCoverageThreshold < raw.CoverageThreshold {
		return fmt.Errorf("trade goods scarcity strongCoverageThreshold must be >= coverageThreshold")
	}
	inputs := settings.InputAvailability
	if inputs.MinWeight < 0 || inputs.AvgWeight < 0 {
		return fmt.Errorf("trade goods scarcity input weights must be non-negative")
	}
	if inputs.MinWeight+inputs.AvgWeight <= 0 {
		return fmt.Errorf("trade goods scarcity input weights must sum to > 0")
	}
	for category, fit := range settings.CategoryAvailabilityFit {
		if fit.Scale < 0 || fit.Scale > 1 {
			return fmt.Errorf("trade goods scarcity categoryAvailabilityFit[%q].scale must be in [0,1]", category)
		}
		if fit.Offset < 0 || fit.Offset > 1 {
			return fmt.Errorf("trade goods scarcity categoryAvailabilityFit[%q].offset must be in [0,1]", category)
		}
	}
	for category, power := range settings.CategoryScarcityPower {
		if power <= 0 || power > 2 {
			return fmt.Errorf("trade goods scarcity categoryScarcityPower[%q] must be in (0,2]", category)
		}
	}
	if err := validateTradeGoodsResponseCurves(settings.DemandResponse, "demandResponse"); err != nil {
		return err
	}
	if err := validateTradeGoodsResponseCurves(settings.SupplyResponse, "supplyResponse"); err != nil {
		return err
	}
	if err := validateTradeGoodsResponseCurves(settings.TradeValueResponse, "tradeValueResponse"); err != nil {
		return err
	}
	return nil
}

func validateTradeGoodsResponseCurves(curves map[string]TradeGoodsResponseCurve, label string) error {
	for category, curve := range curves {
		if curve.Base < 0 || curve.Base > 2 {
			return fmt.Errorf("trade goods scarcity %s[%q].base must be in [0,2]", label, category)
		}
		if curve.Slope < 0 || curve.Slope > 2 {
			return fmt.Errorf("trade goods scarcity %s[%q].slope must be in [0,2]", label, category)
		}
	}
	return nil
}

func validateTradeGoodsMultimodalSettings(settings TradeGoodsMultimodalSettings) error {
	for mode, value := range settings.CapacityScaleByMode {
		if value <= 0 {
			return fmt.Errorf("trade goods multimodal capacityScaleByMode[%q] must be > 0", mode)
		}
	}
	for mode, value := range settings.CapacityFactorByMode {
		if value <= 0 {
			return fmt.Errorf("trade goods multimodal capacityFactorByMode[%q] must be > 0", mode)
		}
	}
	for mode, value := range settings.VolumeBaseByMode {
		if value <= 0 {
			return fmt.Errorf("trade goods multimodal volumeBaseByMode[%q] must be > 0", mode)
		}
	}
	for category, value := range settings.EndpointNeedShareByCategory {
		if value < 0 || value > 1 {
			return fmt.Errorf("trade goods multimodal endpointNeedShareByCategory[%q] must be in [0,1]", category)
		}
	}
	for category, value := range settings.EndpointSurplusReliefByCategory {
		if value < 0 {
			return fmt.Errorf("trade goods multimodal endpointSurplusReliefByCategory[%q] must be non-negative", category)
		}
	}
	if err := validateTradeGoodsResponseCurves(settings.LocalNeedResponse, "localNeedResponse"); err != nil {
		return err
	}
	if err := validateTradeGoodsResponseCurves(settings.GlobalRarityResponse, "globalRarityResponse"); err != nil {
		return err
	}
	if settings.SingleMarketWealthBase < 0 || settings.SingleMarketWealthScale < 0 ||
		settings.DualMarketWealthBase < 0 || settings.DualMarketWealthScale < 0 ||
		settings.SingleMarketFeederScale < 0 || settings.DualMarketFeederScale < 0 ||
		settings.LowCapacityVolumeThreshold < 0 {
		return fmt.Errorf("trade goods multimodal scalar settings must be non-negative")
	}
	return nil
}

func validateTradeGoodsProductionSettings(settings TradeGoodsProductionSettings) error {
	for category, value := range settings.CategorySupplyScale {
		if value <= 0 {
			return fmt.Errorf("trade goods production categorySupplyScale[%q] must be > 0", category)
		}
	}
	for category, value := range settings.CategorySpecializationScale {
		if value < 0 || value > 3 {
			return fmt.Errorf("trade goods production categorySpecializationScale[%q] must be in [0,3]", category)
		}
	}
	for category, value := range settings.ManufacturingBaseScale {
		if value < 0 || value > 2 {
			return fmt.Errorf("trade goods production manufacturingBaseScale[%q] must be in [0,2]", category)
		}
	}
	for category, value := range settings.ManufacturingWorkshopScale {
		if value < 0 || value > 2 {
			return fmt.Errorf("trade goods production manufacturingWorkshopScale[%q] must be in [0,2]", category)
		}
	}
	for category, value := range settings.MarketWorkshopBias {
		if value < 0 || value > 2 {
			return fmt.Errorf("trade goods production marketWorkshopBias[%q] must be in [0,2]", category)
		}
	}
	for category, value := range settings.MarketInputRichnessScale {
		if value < 0 || value > 2 {
			return fmt.Errorf("trade goods production marketInputRichnessScale[%q] must be in [0,2]", category)
		}
	}
	for category, value := range settings.MarketConversionShare {
		if value < 0 || value > 2 {
			return fmt.Errorf("trade goods production marketConversionShare[%q] must be in [0,2]", category)
		}
	}
	for category, value := range settings.MarketDominancePenalty {
		if value < 0 || value > 3 {
			return fmt.Errorf("trade goods production marketDominancePenalty[%q] must be in [0,3]", category)
		}
	}
	for category, value := range settings.MarketImportedInputSupportScale {
		if value < 0 || value > 2 {
			return fmt.Errorf("trade goods production marketImportedInputSupportScale[%q] must be in [0,2]", category)
		}
	}
	for category, value := range settings.MarketExternalInputSupportScale {
		if value < 0 || value > 2 {
			return fmt.Errorf("trade goods production marketExternalInputSupportScale[%q] must be in [0,2]", category)
		}
	}
	for input, value := range settings.MarketInputReservationByInput {
		if value < 0 || value > 2 {
			return fmt.Errorf("trade goods production marketInputReservationByInput[%q] must be in [0,2]", input)
		}
	}
	if settings.MarketInputReservationStrength < 0 || settings.MarketInputReservationStrength > 2 {
		return fmt.Errorf("trade goods production marketInputReservationStrength must be in [0,2]")
	}
	if settings.MarketInputReservationCap < 0 || settings.MarketInputReservationCap > 1 {
		return fmt.Errorf("trade goods production marketInputReservationCap must be in [0,1]")
	}
	if settings.RawCatchmentSpecializationScale < 0 || settings.RawCatchmentSpecializationScale > 2 {
		return fmt.Errorf("trade goods production rawCatchmentSpecializationScale must be in [0,2]")
	}
	if settings.RawCatchmentSpecializationFloor < 0 || settings.RawCatchmentSpecializationFloor > 2 {
		return fmt.Errorf("trade goods production rawCatchmentSpecializationFloor must be in [0,2]")
	}
	if settings.RawPotentialPivot < 0 || settings.RawPotentialPivot > 1 {
		return fmt.Errorf("trade goods production rawPotentialPivot must be in [0,1]")
	}
	if settings.ManufacturingPivot < 0 || settings.ManufacturingPivot > 1 {
		return fmt.Errorf("trade goods production manufacturingPivot must be in [0,1]")
	}
	return nil
}

func validateTradeGoodsDemandSettings(settings TradeGoodsDemandSettings) error {
	for category, value := range settings.CategoryDemandScale {
		if value <= 0 {
			return fmt.Errorf("trade goods demand categoryDemandScale[%q] must be > 0", category)
		}
	}
	for good, value := range settings.GoodDemandScale {
		if value <= 0 {
			return fmt.Errorf("trade goods demand goodDemandScale[%q] must be > 0", good)
		}
	}
	for category, value := range settings.LocalSupplyReliefByCategory {
		if value < 0 || value > 1.5 {
			return fmt.Errorf("trade goods demand localSupplyReliefByCategory[%q] must be in [0,1.5]", category)
		}
	}
	for good, value := range settings.LocalSupplyReliefByGood {
		if value < 0 || value > 1.5 {
			return fmt.Errorf("trade goods demand localSupplyReliefByGood[%q] must be in [0,1.5]", good)
		}
	}
	for category, value := range settings.DriverSpecializationScale {
		if value < 0 || value > 2 {
			return fmt.Errorf("trade goods demand driverSpecializationScale[%q] must be in [0,2]", category)
		}
	}
	for category, value := range settings.MarketCategoryDemandScale {
		if value <= 0 {
			return fmt.Errorf("trade goods demand marketCategoryDemandScale[%q] must be > 0", category)
		}
	}
	for good, value := range settings.MarketGoodDemandScale {
		if value <= 0 {
			return fmt.Errorf("trade goods demand marketGoodDemandScale[%q] must be > 0", good)
		}
	}
	for category, value := range settings.MarketWealthPullScale {
		if value < 0 || value > 2 {
			return fmt.Errorf("trade goods demand marketWealthPullScale[%q] must be in [0,2]", category)
		}
	}
	for category, value := range settings.MarketFeederPullScale {
		if value < 0 || value > 2 {
			return fmt.Errorf("trade goods demand marketFeederPullScale[%q] must be in [0,2]", category)
		}
	}
	if settings.DriverSpecializationPivot < 0 || settings.DriverSpecializationPivot > 1 {
		return fmt.Errorf("trade goods demand driverSpecializationPivot must be in [0,1]")
	}
	return nil
}
