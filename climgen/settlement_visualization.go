package climgen

import "image/color"

func SettlementColor(c SettlementClass) color.RGBA {
	switch c {
	case SettlementOcean:
		return color.RGBA{50, 90, 150, 255}
	case SettlementUnsuitable:
		return color.RGBA{92, 78, 68, 255}
	case SettlementMarginal:
		return color.RGBA{176, 152, 92, 255}
	case SettlementFavorable:
		return color.RGBA{118, 164, 94, 255}
	case SettlementPrime:
		return color.RGBA{66, 130, 88, 255}
	default:
		return color.RGBA{0, 0, 0, 255}
	}
}

func SettlementPreferenceColor(name string) color.RGBA {
	switch name {
	case "Human":
		return color.RGBA{174, 146, 82, 255}
	case "Elf":
		return color.RGBA{92, 150, 96, 255}
	case "Dwarf":
		return color.RGBA{120, 120, 132, 255}
	case "Halfling":
		return color.RGBA{168, 118, 76, 255}
	default:
		return color.RGBA{0, 0, 0, 255}
	}
}
