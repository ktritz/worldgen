package climgen

import "image/color"

func CoastalResourceColor(resource CoastalResourceType) color.RGBA {
	switch resource {
	case CoastalResourceOcean:
		return color.RGBA{24, 48, 90, 255}
	case CoastalResourceNone:
		return color.RGBA{220, 208, 186, 255}
	case CoastalResourceOpenFishery:
		return color.RGBA{63, 132, 214, 255}
	case CoastalResourceEstuarineFishery:
		return color.RGBA{48, 175, 166, 255}
	case CoastalResourceShellfish:
		return color.RGBA{171, 118, 197, 255}
	case CoastalResourceSaltworks:
		return color.RGBA{238, 228, 192, 255}
	default:
		return color.RGBA{255, 0, 255, 255}
	}
}
