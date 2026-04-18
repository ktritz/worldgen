package climgen

import "sort"

type TradeNodeMarketManufacturingCandidate struct {
	Good                string
	Priority            float64
	Potential           float64
	DemandGap           float64
	Bottleneck          string
	BottleneckNeed      float64
	BottleneckAvailable float64
	BottleneckEffective float64
}

type TradeNodeMarketManufacturingDiagnostics struct {
	CandidateCount  int
	ProducedCount   int
	NoInputCount    int
	NoCapacityCount int
	LowYieldCount   int
	Penalized       map[string]float64
	Latent          []TradeNodeMarketManufacturingCandidate
	Blocked         []TradeNodeMarketManufacturingCandidate
}

func analyzeTradeNodeMarketManufacturing(
	market *TradeNodeMarket,
	settings TradeGoodsSettings,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	assignment PolityProfileAssignment,
	production TradeGoodsProductionSettings,
) TradeNodeMarketManufacturingDiagnostics {
	diag := TradeNodeMarketManufacturingDiagnostics{
		Penalized: cloneFloatMap(market.Diagnostics.Penalized),
	}
	if market == nil || network == nil || market.NodeID < 0 || market.NodeID >= len(network.Nodes) {
		return diag
	}
	node := network.Nodes[market.NodeID]
	chainPressure := marketManufacturingChainPressure(market, settings, node, trade, assignment)
	for _, spec := range settings.Goods {
		if len(spec.Inputs) == 0 {
			continue
		}
		if !marketManufacturingEligibleForMarket(spec, node, market) {
			continue
		}
		diag.CandidateCount++
		inputAccess, inputRichness, capacity := marketEffectiveManufacturingInputs(spec, market, chainPressure, production)
		workshop := marketManufacturingWorkshop(node, market, trade, spec.Category, inputRichness, production)
		manufacturingSignal := clamp01(0.44*inputAccess + 0.24*inputRichness + 0.32*workshop)
		efficiency := tradeGoodsManufacturingOutput(spec.Category, inputAccess, workshop, production)
		contextFit := marketProductionContextMultiplier(spec, node, market, trade, assignment)
		profileAffinity := profileGoodAffinity(spec.ProfileProductionAffinity, assignment)
		conversionScale := tradeGoodsCategorySetting(production.MarketConversionShare, spec.Category, 0.40)
		if spec.MarketConversionScale > 0 {
			conversionScale *= spec.MarketConversionScale
		}
		outputScale := Clamp(conversionScale*efficiency*profileAffinity*contextFit*tradeGoodsProductionMultiplier(spec.Category, manufacturingSignal, production.ManufacturingPivot, production), 0, 1)
		potential := capacity * outputScale
		demandGap := maxFloat(market.Demand[spec.Name]-market.Supply[spec.Name], 0)
		if market.Manufactured[spec.Name] > 0.01 {
			diag.ProducedCount++
		}
		if inputAccess <= 0.01 {
			diag.NoInputCount++
		} else if capacity <= 0.01 {
			diag.NoCapacityCount++
		} else if potential <= 0.01 {
			diag.LowYieldCount++
		}
		bottleneck, bottleneckNeed, bottleneckAvailable, bottleneckEffective := marketManufacturingBottleneck(spec, market, chainPressure, production)
		priority := marketManufacturingPriority(spec, market, settings, node, trade, assignment, profileAffinity, contextFit, inputAccess, capacity, chainPressure)
		candidate := TradeNodeMarketManufacturingCandidate{
			Good:                spec.Name,
			Priority:            priority,
			Potential:           potential,
			DemandGap:           demandGap,
			Bottleneck:          bottleneck,
			BottleneckNeed:      bottleneckNeed,
			BottleneckAvailable: bottleneckAvailable,
			BottleneckEffective: bottleneckEffective,
		}
		if market.Manufactured[spec.Name] <= 0.01 && potential > 0.05 {
			diag.Latent = append(diag.Latent, candidate)
		} else if market.Manufactured[spec.Name] <= 0.01 && demandGap > 0.05 {
			diag.Blocked = append(diag.Blocked, candidate)
		}
	}
	sort.Slice(diag.Latent, func(i, j int) bool {
		if diag.Latent[i].Priority != diag.Latent[j].Priority {
			return diag.Latent[i].Priority > diag.Latent[j].Priority
		}
		if diag.Latent[i].Potential != diag.Latent[j].Potential {
			return diag.Latent[i].Potential > diag.Latent[j].Potential
		}
		return diag.Latent[i].Good < diag.Latent[j].Good
	})
	sort.Slice(diag.Blocked, func(i, j int) bool {
		if diag.Blocked[i].Priority != diag.Blocked[j].Priority {
			return diag.Blocked[i].Priority > diag.Blocked[j].Priority
		}
		if diag.Blocked[i].DemandGap != diag.Blocked[j].DemandGap {
			return diag.Blocked[i].DemandGap > diag.Blocked[j].DemandGap
		}
		return diag.Blocked[i].Good < diag.Blocked[j].Good
	})
	return diag
}

func marketManufacturingBottleneck(spec TradeGoodSpec, market *TradeNodeMarket, chainPressure map[string]float64, production TradeGoodsProductionSettings) (string, float64, float64, float64) {
	if market == nil || len(spec.Inputs) == 0 {
		return "none", 0, 0, 0
	}
	limitInput := ""
	limitRatio := 1.0
	limitNeed := 0.0
	limitAvailable := 0.0
	limitEffective := 0.0
	unlockSignal := clamp01(maxFloat(chainPressure[spec.Name]/1.25, spec.MarketInputReservePriority))
	for input, need := range spec.Inputs {
		if need <= 0 {
			continue
		}
		available := maxFloat(market.Supply[input]-market.Demand[input], 0)
		reserved := marketReservedInputAmount(input, available, unlockSignal, chainPressure, production)
		effective := maxFloat(available-reserved, 0)
		ratio := clamp01(effective / need)
		if limitInput == "" || ratio < limitRatio {
			limitInput = input
			limitRatio = ratio
			limitNeed = need
			limitAvailable = available
			limitEffective = effective
		}
	}
	if limitInput == "" {
		return "none", 0, 0, 0
	}
	return limitInput, limitNeed, limitAvailable, limitEffective
}
