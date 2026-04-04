package climgen

import "image/color"

func SettlementNodeColor(kind SettlementNodeKind) color.RGBA {
	switch kind {
	case SettlementNodeHamlet:
		return color.RGBA{186, 160, 108, 255}
	case SettlementNodeVillage:
		return color.RGBA{134, 172, 106, 255}
	case SettlementNodeTown:
		return color.RGBA{92, 134, 100, 255}
	case SettlementNodeCity:
		return color.RGBA{146, 72, 62, 255}
	default:
		return color.RGBA{0, 0, 0, 255}
	}
}
