package climgen

import "image/color"

func ResourceColor(r ResourceType) color.RGBA {
	switch r {
	case ResourceOcean:
		return color.RGBA{65, 105, 180, 255}
	case ResourceNone:
		return color.RGBA{196, 190, 174, 255}
	case ResourceIronOre:
		return color.RGBA{132, 92, 76, 255}
	case ResourceCopperOre:
		return color.RGBA{176, 104, 66, 255}
	case ResourceLeadSilverOre:
		return color.RGBA{122, 118, 136, 255}
	case ResourceGoldOre:
		return color.RGBA{198, 172, 78, 255}
	case ResourceGemstones:
		return color.RGBA{98, 146, 158, 255}
	case ResourcePlacerAlluvial:
		return color.RGBA{212, 182, 94, 255}
	case ResourceCoal:
		return color.RGBA{78, 74, 72, 255}
	case ResourceOilGas:
		return color.RGBA{102, 110, 84, 255}
	case ResourceEvaporite:
		return color.RGBA{216, 198, 164, 255}
	case ResourceClayAggregate:
		return color.RGBA{170, 138, 104, 255}
	case ResourceIndustrialStone:
		return color.RGBA{100, 102, 108, 255}
	default:
		return color.RGBA{255, 0, 255, 255}
	}
}

func ResourcePotential(diag *ResourceDiagnostics, resource ResourceType, idx int) float64 {
	if diag == nil || idx < 0 {
		return 0
	}
	switch resource {
	case ResourceIronOre:
		if idx < len(diag.IronAffinity) {
			return diag.IronAffinity[idx]
		}
	case ResourceCopperOre:
		if idx < len(diag.CopperAffinity) {
			return diag.CopperAffinity[idx]
		}
	case ResourceLeadSilverOre:
		if idx < len(diag.LeadSilverAffinity) {
			return diag.LeadSilverAffinity[idx]
		}
	case ResourceGoldOre:
		if idx < len(diag.GoldAffinity) {
			return diag.GoldAffinity[idx]
		}
	case ResourceGemstones:
		if idx < len(diag.GemAffinity) {
			return diag.GemAffinity[idx]
		}
	case ResourcePlacerAlluvial:
		if idx < len(diag.PlacerAffinity) {
			return diag.PlacerAffinity[idx]
		}
	case ResourceCoal:
		if idx < len(diag.CoalAffinity) {
			return diag.CoalAffinity[idx]
		}
	case ResourceOilGas:
		if idx < len(diag.OilGasAffinity) {
			return diag.OilGasAffinity[idx]
		}
	case ResourceEvaporite:
		if idx < len(diag.EvaporiteAffinity) {
			return diag.EvaporiteAffinity[idx]
		}
	case ResourceClayAggregate:
		if idx < len(diag.ClayAffinity) {
			return diag.ClayAffinity[idx]
		}
	case ResourceIndustrialStone:
		if idx < len(diag.StoneAffinity) {
			return diag.StoneAffinity[idx]
		}
	}
	return 0
}

func ResourcePotentialColor(resource ResourceType, potential float64) color.RGBA {
	base := ResourceColor(resource)
	t := clamp01(potential)
	low := color.RGBA{236, 234, 228, 255}
	return color.RGBA{
		R: uint8(float64(low.R)*(1-t) + float64(base.R)*t),
		G: uint8(float64(low.G)*(1-t) + float64(base.G)*t),
		B: uint8(float64(low.B)*(1-t) + float64(base.B)*t),
		A: 255,
	}
}
