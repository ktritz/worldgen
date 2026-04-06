package climgen

import "sort"

func classifyCoastalTradeCorridors(corridors []CoastalTradeCorridor, settings CoastalTradeSettings) {
	for i := range corridors {
		switch {
		case corridors[i].Flow >= settings.PrimaryFlow:
			corridors[i].Tier = CoastalTradeCorridorPrimary
		case corridors[i].Flow >= settings.RegionalFlow || corridors[i].InterCivilization:
			corridors[i].Tier = CoastalTradeCorridorRegional
		default:
			corridors[i].Tier = CoastalTradeCorridorLocal
		}
	}
}

func applyCoastalTradeDiagnostics(corridors []CoastalTradeCorridor, diagnostics *CoastalTradeDiagnostics) {
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

func identifyMajorCoastalTradePorts(network *SettlementNetworkResult, ports *CoastalPortResult, diagnostics *CoastalTradeDiagnostics) []int {
	if network == nil || ports == nil || diagnostics == nil {
		return nil
	}
	maxCentrality := 0.0
	for _, v := range diagnostics.NodeCentrality {
		if v > maxCentrality {
			maxCentrality = v
		}
	}
	out := make([]int, 0)
	for _, nodeIdx := range ports.MajorPorts {
		if nodeIdx < 0 || nodeIdx >= len(network.Nodes) || nodeIdx >= len(ports.Diagnostics.NodePortScore) {
			continue
		}
		centralityNorm := 0.0
		if maxCentrality > 0 {
			centralityNorm = diagnostics.NodeCentrality[nodeIdx] / maxCentrality
		}
		if centralityNorm <= 0 && ports.Diagnostics.NodePortScore[nodeIdx] < 0.70 {
			continue
		}
		out = append(out, nodeIdx)
	}
	if len(out) == 0 {
		for _, nodeIdx := range ports.MajorPorts {
			if nodeIdx < 0 || nodeIdx >= len(network.Nodes) || nodeIdx >= len(ports.Diagnostics.NodePortScore) {
				continue
			}
			if ports.Diagnostics.NodePortScore[nodeIdx] >= 0.64 {
				out = append(out, nodeIdx)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a := diagnostics.NodeCentrality[out[i]]
		b := diagnostics.NodeCentrality[out[j]]
		if a != b {
			return a > b
		}
		return ports.Diagnostics.NodePortScore[out[i]] > ports.Diagnostics.NodePortScore[out[j]]
	})
	return out
}
