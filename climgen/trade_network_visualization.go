package climgen

import "image/color"

func TradeCorridorTierColor(tier TradeCorridorTier) color.RGBA {
	switch tier {
	case TradeCorridorPrimary:
		return color.RGBA{R: 188, G: 104, B: 70, A: 255}
	case TradeCorridorRegional:
		return color.RGBA{R: 148, G: 112, B: 84, A: 255}
	default:
		return color.RGBA{R: 112, G: 96, B: 86, A: 255}
	}
}
