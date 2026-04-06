package climgen

import "sort"

func buildOceanTradeCorridor(fromNode, toNode, fromCiv, toCiv int, flow float64, path coastalEndpointPath) OceanTradeCorridor {
	return OceanTradeCorridor{
		FromNode:          fromNode,
		ToNode:            toNode,
		FromCivilization:  fromCiv,
		ToCivilization:    toCiv,
		TravelCost:        path.cost,
		Flow:              flow,
		MeanExposure:      path.meanExposure,
		MeanCurrentAssist: path.meanAssist,
		CellPath:          append([]int(nil), path.cells...),
		InterCivilization: fromCiv >= 0 && toCiv >= 0 && fromCiv != toCiv,
	}
}

func classifyOceanTradeCorridors(corridors []OceanTradeCorridor, settings OceanTradeSettings) {
	for i := range corridors {
		switch {
		case corridors[i].Flow >= settings.PrimaryFlow:
			corridors[i].Tier = OceanTradeCorridorPrimary
		case corridors[i].Flow >= settings.RegionalFlow || corridors[i].InterCivilization:
			corridors[i].Tier = OceanTradeCorridorRegional
		default:
			corridors[i].Tier = OceanTradeCorridorLocal
		}
	}
}

func applyOceanTradeDiagnostics(corridors []OceanTradeCorridor, diagnostics *OceanTradeDiagnostics) {
	if diagnostics == nil {
		return
	}
	for _, corridor := range corridors {
		for _, nodeIdx := range []int{corridor.FromNode, corridor.ToNode} {
			if nodeIdx >= 0 && nodeIdx < len(diagnostics.NodeCentrality) {
				diagnostics.NodeCentrality[nodeIdx] += corridor.Flow
			}
		}
		for _, cellIdx := range corridor.CellPath {
			if cellIdx >= 0 && cellIdx < len(diagnostics.RouteIntensity) {
				diagnostics.RouteIntensity[cellIdx] += corridor.Flow
				diagnostics.RouteExposure[cellIdx] += corridor.Flow * corridor.MeanExposure
			}
		}
	}
	for i := range diagnostics.RouteExposure {
		if diagnostics.RouteIntensity[i] > 0 {
			diagnostics.RouteExposure[i] /= diagnostics.RouteIntensity[i]
		}
	}
}

func identifyMajorOceanTradePorts(network *SettlementNetworkResult, ports *CoastalPortResult, diagnostics *OceanTradeDiagnostics) []int {
	if network == nil || ports == nil || ports.Diagnostics == nil || diagnostics == nil {
		return nil
	}
	maxCentrality := 0.0
	for _, v := range diagnostics.NodeCentrality {
		if v > maxCentrality {
			maxCentrality = v
		}
	}
	out := make([]int, 0, len(ports.MajorDeepwaterPorts))
	for _, nodeIdx := range ports.MajorDeepwaterPorts {
		if nodeIdx < 0 || nodeIdx >= len(network.Nodes) || nodeIdx >= len(ports.Diagnostics.NodeDeepwaterScore) {
			continue
		}
		if maxCentrality > 0 && diagnostics.NodeCentrality[nodeIdx]/maxCentrality > 0 {
			out = append(out, nodeIdx)
		}
	}
	if len(out) == 0 {
		out = append(out, ports.MajorDeepwaterPorts...)
	}
	sort.Slice(out, func(i, j int) bool {
		a := diagnostics.NodeCentrality[out[i]]
		b := diagnostics.NodeCentrality[out[j]]
		if a != b {
			return a > b
		}
		return ports.Diagnostics.NodeDeepwaterScore[out[i]] > ports.Diagnostics.NodeDeepwaterScore[out[j]]
	})
	return out
}
