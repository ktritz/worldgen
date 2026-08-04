package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

func printTradeGoodSummary(result *climgen.TradeGoodResult) {
	if result == nil || len(result.Goods) == 0 {
		fmt.Println("    tradeGoods: goods=0")
		return
	}
	fmt.Printf("    tradeGoods: goods=%d\n", len(result.Goods))
	for _, line := range topTradeGoodEndowments(result, 8) {
		fmt.Printf("      goodPotential[%s]=%.2f category=%s\n", line.name, line.value, line.category)
	}
	for _, line := range topScarceTradeGoods(result, 5) {
		fmt.Printf("      goodScarcity[%s]=%.2f\n", line.name, line.value)
	}
}

func printPolityGoodsSummary(result *climgen.PolityGoodsResult, settings climgen.TradeGoodsSettings) {
	if result == nil || len(result.Balances) == 0 {
		fmt.Println("    polityGoods: polities=0")
		return
	}
	exportPolities := 0
	importPolities := 0
	for _, balance := range result.Balances {
		if len(balance.Exports) > 0 {
			exportPolities++
		}
		if len(balance.Imports) > 0 {
			importPolities++
		}
	}
	fmt.Printf("    polityGoods: polities=%d exporters=%d importers=%d\n", len(result.Balances), exportPolities, importPolities)
	for _, balance := range topPolityGoodBalances(result, 5) {
		expTop1, expTop3, expMix := tradeBalanceConcentration(balance.Surplus, settings, true)
		impTop1, impTop3, impMix := tradeBalanceConcentration(balance.Surplus, settings, false)
		fmt.Printf(
			"      polityGood[%d]: wealth=%.2f exports=%s imports=%s expTop1=%.2f expTop3=%.2f expMix=%s impTop1=%.2f impTop3=%.2f impMix=%s\n",
			balance.PolityID,
			balance.MarketWealth,
			formatGoodValues(balance.Exports, 3),
			formatGoodValues(balance.Imports, 3),
			expTop1,
			expTop3,
			expMix,
			impTop1,
			impTop3,
			impMix,
		)
	}
}

func printNodeGoodsSummary(result *climgen.NodeGoodsResult, network *climgen.SettlementNetworkResult) {
	if result == nil || len(result.Balances) == 0 {
		fmt.Println("    nodeGoods: nodes=0")
		return
	}
	exportNodes := 0
	importNodes := 0
	for _, balance := range result.Balances {
		if len(balance.Exports) > 0 {
			exportNodes++
		}
		if len(balance.Imports) > 0 {
			importNodes++
		}
	}
	fmt.Printf("    nodeGoods: nodes=%d exporters=%d importers=%d\n", len(result.Balances), exportNodes, importNodes)
	for _, balance := range topNodeGoodBalances(result, 3, true) {
		nodeLabel := fmt.Sprintf("%d", balance.NodeID)
		if network != nil && balance.NodeID >= 0 && balance.NodeID < len(network.Nodes) {
			nodeLabel = fmt.Sprintf("%d %s", balance.NodeID, climgen.SettlementNodeKindName(network.Nodes[balance.NodeID].Kind))
		}
		fmt.Printf("      nodeExport[%s]: wealth=%.2f exports=%s imports=%s\n", nodeLabel, balance.Wealth, formatGoodValues(balance.Exports, 3), formatGoodValues(balance.Imports, 3))
	}
	for _, balance := range topNodeGoodBalances(result, 3, false) {
		nodeLabel := fmt.Sprintf("%d", balance.NodeID)
		if network != nil && balance.NodeID >= 0 && balance.NodeID < len(network.Nodes) {
			nodeLabel = fmt.Sprintf("%d %s", balance.NodeID, climgen.SettlementNodeKindName(network.Nodes[balance.NodeID].Kind))
		}
		fmt.Printf("      nodeImport[%s]: wealth=%.2f imports=%s exports=%s\n", nodeLabel, balance.Wealth, formatGoodValues(balance.Imports, 3), formatGoodValues(balance.Exports, 3))
	}
}

func printTradeNodeMarketSummary(result *climgen.TradeNodeMarketResult, network *climgen.SettlementNetworkResult, settings climgen.TradeGoodsSettings) {
	if result == nil || len(result.Markets) == 0 {
		fmt.Println("    tradeMarkets: nodes=0")
		return
	}
	feederMarkets := 0
	for _, market := range result.Markets {
		if market.FeederNodes > 0 {
			feederMarkets++
		}
	}
	fmt.Printf("    tradeMarkets: nodes=%d feederNodes=%d\n", len(result.Markets), feederMarkets)
	printTradeNodeMarketAggregateDiagnostics(result, settings)
	for _, market := range topTradeNodeMarkets(result, 4) {
		nodeLabel := fmt.Sprintf("%d", market.NodeID)
		if network != nil && market.NodeID >= 0 && market.NodeID < len(network.Nodes) {
			nodeLabel = fmt.Sprintf("%d %s", market.NodeID, climgen.SettlementNodeKindName(network.Nodes[market.NodeID].Kind))
		}
		expTop1, expTop3, expMix := tradeBalanceConcentration(market.Surplus, settings, true)
		impTop1, impTop3, impMix := tradeBalanceConcentration(market.Surplus, settings, false)
		madeShare := manufacturedExportShare(market, settings)
		madeCat := manufacturingCategoryMix(market.Manufactured, settings)
		blockedCat := candidateCategoryMix(market.Diagnostics.Blocked, settings)
		penalizedCat := penalizedCategoryMix(market.Diagnostics.Penalized, settings)
		fmt.Printf(
			"      tradeMarket[%s]: wealth=%.2f feeders=%d feederWealth=%.2f exports=%s imports=%s made=%s used=%s makeDiag=%s madeCat=%s blockedCat=%s penalized=%s penalizedCat=%s latent=%s blocked=%s madeShare=%.2f expTop1=%.2f expTop3=%.2f expMix=%s impTop1=%.2f impTop3=%.2f impMix=%s\n",
			nodeLabel,
			market.Wealth,
			market.FeederNodes,
			market.FeederWealth,
			formatGoodValues(market.Exports, 3),
			formatGoodValues(market.Imports, 3),
			formatMetricValues(topMetricMap(market.Manufactured, 3), 3),
			formatMetricValues(topMetricMap(market.Consumed, 3), 3),
			formatMarketManufacturingDiagnostics(market.Diagnostics),
			madeCat,
			blockedCat,
			formatMetricValues(topMetricMap(market.Diagnostics.Penalized, 2), 2),
			penalizedCat,
			formatMarketManufacturingCandidates(market.Diagnostics.Latent, 2),
			formatMarketManufacturingCandidates(market.Diagnostics.Blocked, 2),
			madeShare,
			expTop1,
			expTop3,
			expMix,
			impTop1,
			impTop3,
			impMix,
		)
	}
	printTradeMarketCategorySummary(result, network, settings, "processed")
	printTradeMarketCategorySummary(result, network, settings, "finished")
	printTradeMarketCategorySummary(result, network, settings, "luxury")
	printTradeMarketCategorySummary(result, network, settings, "strategic")
}

func printMultimodalTradeSummary(result *climgen.MultimodalTradeResult, settings climgen.TradeGoodsSettings) {
	if result == nil {
		fmt.Println("    multimodalTrade: exchanges=0 pairs=0")
		return
	}
	if len(result.Exchanges) == 0 {
		fmt.Println("    multimodalTrade: exchanges=0 pairs=0")
		printMultimodalTradeDiagnostics(result, settings)
		return
	}
	modeTotals := map[string]float64{}
	modeVolumes := map[string]float64{}
	for _, exchange := range result.Exchanges {
		if exchange.Internal {
			continue
		}
		modeTotals[exchange.Mode] += exchange.Value
		modeVolumes[exchange.Mode] += exchange.Volume
	}
	fmt.Printf(
		"    multimodalTrade: exchanges=%d pairs=%d score=%.2f volume=%.2f matched=%.2f internalExchanges=%d internalScore=%.2f internalVolume=%.2f modes=%s modeVolume=%s\n",
		result.Diagnostics.ExternalExchanges,
		len(result.Pairs),
		result.Diagnostics.TotalScore,
		result.Diagnostics.TotalVolume,
		result.Diagnostics.TotalMatched,
		result.Diagnostics.InternalExchanges,
		result.Diagnostics.InternalScore,
		result.Diagnostics.InternalVolume,
		formatModeValues(topSummaryModeValues(modeTotals, 4), 4),
		formatModeValues(topSummaryModeValues(modeVolumes, 4), 4),
	)
	printMultimodalTradeDiagnostics(result, settings)
	printRouteExchangeDiagnostics(result, "coastal", 5)
	printRouteExchangeDiagnostics(result, "ocean", 3)
	limit := 5
	if len(result.Pairs) < limit {
		limit = len(result.Pairs)
	}
	for i := 0; i < limit; i++ {
		pair := result.Pairs[i]
		fmt.Printf(
			"      tradeFlow[%d->%d]: score=%.2f volume=%.2f matched=%.2f fit=%.2f transport=%.2f need=%.2f rarity=%.2f modes=%s modeVolume=%s goods=%s\n",
			pair.FromPolity,
			pair.ToPolity,
			pair.Value,
			pair.Volume,
			pair.Matched,
			leadingTradeFlowFactor(pair.Goods, func(flow climgen.TradeGoodFlowValue) float64 { return flow.MarketFit }),
			leadingTradeFlowFactor(pair.Goods, func(flow climgen.TradeGoodFlowValue) float64 { return flow.Transport }),
			leadingTradeFlowFactor(pair.Goods, func(flow climgen.TradeGoodFlowValue) float64 { return flow.LocalNeed }),
			leadingTradeFlowFactor(pair.Goods, func(flow climgen.TradeGoodFlowValue) float64 { return flow.Rarity }),
			formatModeValues(pair.Modes, 3),
			formatModeValues(pair.ModeVolume, 3),
			formatTradeFlowGoods(pair.Goods, 3),
		)
	}
}

func printRouteExchangeDiagnostics(result *climgen.MultimodalTradeResult, mode string, limit int) {
	if result == nil || len(result.Exchanges) == 0 || limit <= 0 {
		return
	}
	exchanges := make([]climgen.TradeGoodExchange, 0)
	for _, exchange := range result.Exchanges {
		if exchange.Mode == mode {
			exchanges = append(exchanges, exchange)
		}
	}
	if len(exchanges) == 0 {
		return
	}
	sort.Slice(exchanges, func(i, j int) bool {
		if exchanges[i].Value != exchanges[j].Value {
			return exchanges[i].Value > exchanges[j].Value
		}
		return exchanges[i].VolumeCapacity > exchanges[j].VolumeCapacity
	})
	if len(exchanges) < limit {
		limit = len(exchanges)
	}
	for i := 0; i < limit; i++ {
		exchange := exchanges[i]
		fmt.Printf(
			"      routeExchange[%s:%d]: polity=%d->%d civ=%d->%d internal=%v score=%.2f volume=%.2f matched=%.2f flow=%.2f cost=%.2f cap=%.3f volCap=%.2f fit=%.2f goods=%s\n",
			mode,
			exchange.RouteID,
			exchange.FromPolity,
			exchange.ToPolity,
			exchange.FromCivilization,
			exchange.ToCivilization,
			exchange.Internal,
			exchange.Value,
			exchange.Volume,
			exchange.Matched,
			exchange.RouteFlow,
			exchange.TravelCost,
			exchange.Capacity,
			exchange.VolumeCapacity,
			exchange.AvgMarketFit,
			formatTradeFlowGoods(exchange.Goods, 3),
		)
	}
}

func printMultimodalTradeDiagnostics(result *climgen.MultimodalTradeResult, settings climgen.TradeGoodsSettings) {
	fmt.Printf(
		"      tradeDiag: routes=%d/%d avgCapacity=%.2f avgVolumeCap=%.2f avgMarketFit=%.2f goods=%d/%d noSurplus=%d noNeed=%d noEndpoint=%d srcCap=%d needCap=%d lowCap=%d lowFit=%d lowScore=%d\n",
		result.Diagnostics.RouteActive,
		result.Diagnostics.RouteCandidates,
		result.Diagnostics.AvgCapacity,
		result.Diagnostics.AvgVolumeCapacity,
		result.Diagnostics.AvgMarketFit,
		result.Diagnostics.AcceptedGoods,
		result.Diagnostics.CandidateGoods,
		result.Diagnostics.NoSourceSurplus,
		result.Diagnostics.NoSinkNeed,
		result.Diagnostics.NoEndpointSupply,
		result.Diagnostics.SourceConstrained,
		result.Diagnostics.NeedConstrained,
		result.Diagnostics.LowCapacity,
		result.Diagnostics.LowMarketFit,
		result.Diagnostics.LowScoreFiltered,
	)
	fmt.Printf(
		"      tradeCapDiag: srcMatched=%.2f srcCap=%.2f srcScaledKeys=%d sinkMatched=%.2f sinkPolityCap=%.2f sinkEndpointCap=%.2f sinkEffectiveCap=%.2f sinkScaledKeys=%d sinkEndpointDominatedKeys=%d\n",
		result.Diagnostics.SourcePreCapMatched,
		result.Diagnostics.SourceCapacity,
		result.Diagnostics.SourceScaledKeys,
		result.Diagnostics.SinkPreCapMatched,
		result.Diagnostics.SinkPolityDeficitCapacity,
		result.Diagnostics.SinkEndpointCapacity,
		result.Diagnostics.SinkEffectiveCapacity,
		result.Diagnostics.SinkScaledKeys,
		result.Diagnostics.SinkEndpointDominatedKeys,
	)
	printRouteCivilizationDiagnostics(result)
	printTradeFlowModeDiagnostics(result)
	printTradeFlowCategoryDiagnostics(result)
	printTradeFlowCategoryMix(result, settings)
	for _, category := range []string{"processed", "finished", "luxury", "strategic"} {
		printTradeFlowCategorySummary(result, settings, category)
		printTradeFlowCategoryModeSummary(result, settings, category)
	}
}

func printTradeFlowModeDiagnostics(result *climgen.MultimodalTradeResult) {
	if result == nil || len(result.Diagnostics.ByMode) == 0 {
		return
	}
	modes := make([]climgen.MultimodalTradeModeDiagnostics, 0, len(result.Diagnostics.ByMode))
	for _, entry := range result.Diagnostics.ByMode {
		modes = append(modes, entry)
	}
	sort.Slice(modes, func(i, j int) bool {
		if modes[i].TotalScore != modes[j].TotalScore {
			return modes[i].TotalScore > modes[j].TotalScore
		}
		return modes[i].Mode < modes[j].Mode
	})
	for _, entry := range modes {
		exchanges := entry.ExternalExchanges + entry.InternalExchanges
		avgCap := meanFromTotal(entry.Capacity, exchanges)
		avgVolCap := meanFromTotal(entry.VolumeCapacity, exchanges)
		avgFit := meanFromTotal(entry.MarketFit, exchanges)
		fmt.Printf(
			"      tradeModeDiag[%s]: routes=%d/%d corridors=%d skipUnknown=%d skipSamePolity=%d external=%d internal=%d score=%.2f volume=%.2f matched=%.2f internalScore=%.2f avgCap=%.3f avgVolCap=%.2f avgFit=%.2f\n",
			entry.Mode,
			entry.RouteActive,
			entry.RouteCandidates,
			entry.RouteCorridors,
			entry.SkippedUnknown,
			entry.SkippedSamePolity,
			entry.ExternalExchanges,
			entry.InternalExchanges,
			entry.TotalScore,
			entry.TotalVolume,
			entry.TotalMatched,
			entry.InternalScore,
			avgCap,
			avgVolCap,
			avgFit,
		)
	}
}

func meanFromTotal(total float64, count int) float64 {
	if count <= 0 {
		return 0
	}
	return total / float64(count)
}

func printRouteCivilizationDiagnostics(result *climgen.MultimodalTradeResult) {
	if result == nil || len(result.Exchanges) == 0 {
		return
	}
	sameScore := 0.0
	sameVolume := 0.0
	sameCount := 0
	interScore := 0.0
	interVolume := 0.0
	interCount := 0
	unknownScore := 0.0
	unknownVolume := 0.0
	unknownCount := 0
	for _, exchange := range result.Exchanges {
		switch {
		case exchange.FromCivilization < 0 || exchange.ToCivilization < 0:
			unknownScore += exchange.Value
			unknownVolume += exchange.Volume
			unknownCount++
		case exchange.FromCivilization == exchange.ToCivilization:
			sameScore += exchange.Value
			sameVolume += exchange.Volume
			sameCount++
		default:
			interScore += exchange.Value
			interVolume += exchange.Volume
			interCount++
		}
	}
	fmt.Printf(
		"      routeCivDiag: same=%d/s%.2f/v%.2f inter=%d/s%.2f/v%.2f unknown=%d/s%.2f/v%.2f\n",
		sameCount,
		sameScore,
		sameVolume,
		interCount,
		interScore,
		interVolume,
		unknownCount,
		unknownScore,
		unknownVolume,
	)
}
