package terrain

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
)

// LocalRefinementSettings controls the first-pass connected-cell local DEM
// prototype built from coarse terrain and hydrology scaffolds.
type LocalRefinementSettings struct {
	Resolution         int
	Radius             int
	NoiseFrequency     float64
	NoiseOctaves       int
	NoiseAmplitude     float64
	ChannelDepthScale  float64
	MarginScale        float64
	LakeFlattening     float64
	SideflowSplitScale float64
}

// DefaultLocalRefinementSettings returns conservative defaults for the local
// refinement prototype.
func DefaultLocalRefinementSettings() LocalRefinementSettings {
	return LocalRefinementSettings{
		Resolution:         320,
		Radius:             1,
		NoiseFrequency:     140.0,
		NoiseOctaves:       4,
		NoiseAmplitude:     1.0,
		ChannelDepthScale:  18.0,
		MarginScale:        0.75,
		LakeFlattening:     0.75,
		SideflowSplitScale: 0.45,
	}
}

// LocalRefinementDebug summarizes whether the refined patch is obeying the
// coarse terrain/hydrology contracts closely enough to be useful.
type LocalRefinementDebug struct {
	CenterCell            int
	SelectedCells         []int
	SupportCells          []int
	MeanBoundaryMismatch  float64
	MaxBoundaryMismatch   float64
	NumBoundarySamples    int
	MeanChannelCarve      float64
	MaxChannelCarve       float64
	NumChannelEdges       int
	LakeCoveragePct       float64
	ChannelCoveragePct    float64
	BoundaryCrossings     []LocalBoundaryCrossing
	CellBoundaryCrossings []LocalCellBoundaryCrossing
}

type LocalBoundaryCrossing struct {
	Side       string
	BearingDeg float64
	WidthPx    int
	Strength   float64
	Kind       string
}

type LocalCellBoundaryCrossing struct {
	Neighbor   int
	BearingDeg float64
	Offset     float64
	WidthPx    int
	Strength   float64
	Kind       string
}

// LocalRefinementPatch is a locally refined DEM patch in a tangent-plane frame.
type LocalRefinementPatch struct {
	Width       int
	Height      int
	XMin        float64
	XMax        float64
	YMin        float64
	YMax        float64
	Elevation   []float64
	WaterMask   []bool
	Hydrology   []float64
	WaterSignal []float64
	LakeSignal  []float64
	ChannelMask []float64
	NearestCell []int
	Center      Vector3D
	East        Vector3D
	North       Vector3D
	Debug       LocalRefinementDebug
}

type localPlanePoint struct {
	idx int
	x   float64
	y   float64
}

type localChannelEdge struct {
	ax, ay   float64
	bx, by   float64
	strength float64
}

// BuildLocalRefinementPatch builds a small connected-cell DEM patch around the
// requested center cell using the coarse terrain and hydrology scaffolds.
func BuildLocalRefinementPatch(
	sites []Vector3D,
	cells []VoronoiCell,
	elevation []float64,
	hydro *HydrologyScaffold,
	refine *TerrainRefinementScaffold,
	centerCell int,
	seed int64,
	settings LocalRefinementSettings,
	contract *SharedBoundaryContract,
) (*LocalRefinementPatch, error) {
	if hydro == nil || refine == nil {
		return nil, fmt.Errorf("missing hydrology or terrain refinement scaffold")
	}
	if centerCell < 0 || centerCell >= len(elevation) {
		return nil, fmt.Errorf("center cell %d out of range", centerCell)
	}
	if settings.Resolution < 32 {
		settings.Resolution = 32
	}
	if settings.NoiseOctaves < 1 {
		settings.NoiseOctaves = 1
	}
	if settings.Radius < 0 {
		settings.Radius = 0
	}
	if settings.MarginScale <= 0 {
		settings.MarginScale = 0.75
	}

	selected := bfsCells(cells, centerCell, settings.Radius)
	supportSet := make(map[int]struct{}, len(selected)*2)
	for _, idx := range selected {
		supportSet[idx] = struct{}{}
		for _, nidx := range cells[idx].NeighborSiteIndices {
			neighbor := int(nidx)
			if neighbor >= 0 && neighbor < len(cells) {
				supportSet[neighbor] = struct{}{}
			}
		}
	}
	support := make([]int, 0, len(supportSet))
	for idx := range supportSet {
		support = append(support, idx)
	}
	sort.Ints(support)

	center := patchCenterVector(sites, selected)
	east, north := tangentBasis(center)
	points := make([]localPlanePoint, 0, len(support))
	xMin, yMin := math.Inf(1), math.Inf(1)
	xMax, yMax := math.Inf(-1), math.Inf(-1)
	for _, idx := range support {
		x, y := projectToLocalPlane(center, east, north, sites[idx])
		points = append(points, localPlanePoint{idx: idx, x: x, y: y})
		if x < xMin {
			xMin = x
		}
		if x > xMax {
			xMax = x
		}
		if y < yMin {
			yMin = y
		}
		if y > yMax {
			yMax = y
		}
	}
	cellSpacing := estimateLocalSpacing(points)
	margin := settings.MarginScale * cellSpacing
	if math.IsNaN(margin) || math.IsInf(margin, 0) || margin <= 0 {
		margin = 0.03
	}
	xMin -= margin
	xMax += margin
	yMin -= margin
	yMax += margin

	width := settings.Resolution
	height := settings.Resolution
	patch := &LocalRefinementPatch{
		Width:       width,
		Height:      height,
		XMin:        xMin,
		XMax:        xMax,
		YMin:        yMin,
		YMax:        yMax,
		Elevation:   make([]float64, width*height),
		WaterMask:   make([]bool, width*height),
		Hydrology:   make([]float64, width*height),
		WaterSignal: make([]float64, width*height),
		LakeSignal:  make([]float64, width*height),
		ChannelMask: make([]float64, width*height),
		NearestCell: make([]int, width*height),
		Center:      center,
		East:        east,
		North:       north,
		Debug: LocalRefinementDebug{
			CenterCell:    centerCell,
			SelectedCells: append([]int(nil), selected...),
			SupportCells:  append([]int(nil), support...),
		},
	}

	channelEdges := collectChannelEdges(points, hydro, elevation, settings.SideflowSplitScale)
	channelEdges = append(channelEdges, collectLakeSpillEdges(points, hydro, elevation, cellSpacing)...)
	channelEdges = append(channelEdges, collectContractChannelEdges(points, refine, centerCell, contract, cellSpacing)...)
	lakeFields := buildLakeFields(points, elevation, hydro, refine, cellSpacing)
	totalCarve := 0.0
	maxCarve := 0.0
	carveSamples := 0
	lakePixels := 0
	channelPixels := 0
	for py := 0; py < height; py++ {
		tY := 0.0
		if height > 1 {
			tY = float64(py) / float64(height-1)
		}
		y := Lerp(yMax, yMin, tY)
		for px := 0; px < width; px++ {
			tX := 0.0
			if width > 1 {
				tX = float64(px) / float64(width-1)
			}
			x := Lerp(xMin, xMax, tX)
			pos := localPlaneToSphere(center, east, north, x, y)
			nearest, base, slopeTerm, reliefAmp := sampleCoarseTerrain(x, y, pos, points, elevation, refine)
			noise := reliefAmp * settings.NoiseAmplitude * FBMNoiseWithFreq(pos, seed+313131, settings.NoiseFrequency, settings.NoiseOctaves)
			refined := base + noise - slopeTerm
			channelSignal := channelInfluence(x, y, channelEdges, cellSpacing)
			carve := channelCarveDepth(x, y, channelEdges, cellSpacing, settings.ChannelDepthScale)
			lakeMask, lakeSurface := sampleLakeField(x, y, lakeFields)
			if lakeMask > 0 {
				refined = Lerp(refined, lakeSurface, Clamp(settings.LakeFlattening*lakeMask, 0, 1))
			}
			if carve > 0 {
				refined -= carve
				totalCarve += carve
				carveSamples++
				if carve > maxCarve {
					maxCarve = carve
				}
			}
			idx := py*width + px
			patch.Elevation[idx] = refined
			patch.Hydrology[idx] = carve
			patch.ChannelMask[idx] = channelSignal
			patch.LakeSignal[idx] = lakeMask
			patch.NearestCell[idx] = nearest
			lakeWater := 0.0
			if lakeMask > 0 {
				lakeWater = lakeMask * (1.0 - SmoothStep(lakeSurface-3, lakeSurface+9, refined))
			}
			oceanWater := 0.0
			if refined <= 0 {
				oceanWater = 1.0
			}
			waterSignal := math.Max(oceanWater, lakeWater)
			patch.WaterSignal[idx] = waterSignal
			patch.WaterMask[idx] = waterSignal > 0.25
			if lakeWater > 0.25 {
				lakePixels++
			}
			if channelSignal > 0.30 {
				channelPixels++
			}
		}
	}
	if carveSamples > 0 {
		patch.Debug.MeanChannelCarve = totalCarve / float64(carveSamples)
	}
	patch.Debug.MaxChannelCarve = maxCarve
	patch.Debug.NumChannelEdges = len(channelEdges)
	patch.Debug.LakeCoveragePct = 100 * float64(lakePixels) / float64(width*height)
	patch.Debug.ChannelCoveragePct = 100 * float64(channelPixels) / float64(width*height)
	patch.Debug.BoundaryCrossings = extractBoundaryCrossings(patch)
	patch.Debug.CellBoundaryCrossings = extractCellBoundaryCrossings(patch, points, refine, centerCell)

	meanMismatch, maxMismatch, samples := sampleBoundaryMismatch(patch, points, refine)
	patch.Debug.MeanBoundaryMismatch = meanMismatch
	patch.Debug.MaxBoundaryMismatch = maxMismatch
	patch.Debug.NumBoundarySamples = samples
	return patch, nil
}

// RenderLocalRefinementPatch writes a PNG of the refined patch.
func RenderLocalRefinementPatch(patch *LocalRefinementPatch, filename string) error {
	img := image.NewRGBA(image.Rect(0, 0, patch.Width, patch.Height))
	for py := 0; py < patch.Height; py++ {
		for px := 0; px < patch.Width; px++ {
			idx := py*patch.Width + px
			elev := patch.Elevation[idx]
			c := HypsometricColor(elev)
			if patch.WaterMask[idx] {
				if elev > 0 {
					c = color.RGBA{70, 130, 190, 255}
				}
			}
			img.Set(px, py, c)
		}
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	return nil
}

// RenderLocalHydrologyPatch writes a PNG emphasizing local water features,
// channel carving, and coarse water masks for review.
func RenderLocalHydrologyPatch(patch *LocalRefinementPatch, filename string) error {
	img := image.NewRGBA(image.Rect(0, 0, patch.Width, patch.Height))
	maxHydro := 0.0
	for _, v := range patch.Hydrology {
		if v > maxHydro {
			maxHydro = v
		}
	}
	if maxHydro < 1e-6 {
		maxHydro = 1
	}
	for py := 0; py < patch.Height; py++ {
		for px := 0; px < patch.Width; px++ {
			idx := py*patch.Width + px
			elev := patch.Elevation[idx]
			h := patch.Hydrology[idx] / maxHydro
			lake := patch.LakeSignal[idx]
			channel := patch.ChannelMask[idx]
			water := patch.WaterSignal[idx]
			var c color.RGBA
			switch {
			case water > 0.55 && elev <= 0:
				c = color.RGBA{25, 80, 170, 255}
			case lake > 0.22 || water > 0.35:
				c = color.RGBA{70, 145, 215, 255}
			case channel > 0.18 || h > 0.10:
				intensity := uint8(Clamp(95+160*math.Max(channel, h), 0, 255))
				c = color.RGBA{30, intensity, 230, 255}
			default:
				base := uint8(Clamp(210-0.02*math.Max(elev, 0), 120, 220))
				c = color.RGBA{base, base, base, 255}
			}
			img.Set(px, py, c)
		}
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	return nil
}

// SampleLocalPatchAtSphere samples the refined patch at a global spherical
// position using the patch's tangent-plane frame.
func SampleLocalPatchAtSphere(patch *LocalRefinementPatch, pos Vector3D) float64 {
	x, y := projectToLocalPlane(patch.Center, patch.East, patch.North, pos)
	return samplePatchNearest(patch, x, y)
}

// CompareLocalRefinementOverlap compares two refined patches at the shared
// coarse cell centers in `cellsToCheck`.
func CompareLocalRefinementOverlap(
	a, b *LocalRefinementPatch,
	sites []Vector3D,
	cellsToCheck []int,
) (mean, max float64, count int) {
	total := 0.0
	for _, idx := range cellsToCheck {
		if idx < 0 || idx >= len(sites) {
			continue
		}
		va := SampleLocalPatchAtSphere(a, sites[idx])
		vb := SampleLocalPatchAtSphere(b, sites[idx])
		err := math.Abs(va - vb)
		total += err
		count++
		if err > max {
			max = err
		}
	}
	if count > 0 {
		mean = total / float64(count)
	}
	return mean, max, count
}

func bfsCells(cells []VoronoiCell, start int, radius int) []int {
	type item struct {
		idx   int
		depth int
	}
	queue := []item{{idx: start, depth: 0}}
	seen := map[int]bool{start: true}
	out := make([]int, 0)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		out = append(out, cur.idx)
		if cur.depth >= radius {
			continue
		}
		for _, nidx := range cells[cur.idx].NeighborSiteIndices {
			neighbor := int(nidx)
			if neighbor < 0 || neighbor >= len(cells) || seen[neighbor] {
				continue
			}
			seen[neighbor] = true
			queue = append(queue, item{idx: neighbor, depth: cur.depth + 1})
		}
	}
	sort.Ints(out)
	return out
}

func patchCenterVector(sites []Vector3D, cells []int) Vector3D {
	sum := Vector3D{}
	for _, idx := range cells {
		sum = sum.Add(sites[idx])
	}
	return sum.Normalize()
}

func tangentBasis(center Vector3D) (east, north Vector3D) {
	up := Vector3D{X: 0, Y: 1, Z: 0}
	east = center.Cross(up)
	if east.LengthSq() < 1e-12 {
		east = Vector3D{X: 1, Y: 0, Z: 0}
	}
	east = east.Normalize()
	north = east.Cross(center).Normalize()
	return east, north
}

func projectToLocalPlane(center, east, north, pos Vector3D) (float64, float64) {
	tangent := pos.Subtract(center.Scale(pos.Dot(center)))
	return tangent.Dot(east), tangent.Dot(north)
}

func localPlaneToSphere(center, east, north Vector3D, x, y float64) Vector3D {
	return center.Add(east.Scale(x)).Add(north.Scale(y)).Normalize()
}

func estimateLocalSpacing(points []localPlanePoint) float64 {
	if len(points) < 2 {
		return 0.04
	}
	sum := 0.0
	count := 0
	for i := range points {
		best := math.Inf(1)
		for j := range points {
			if i == j {
				continue
			}
			dx := points[i].x - points[j].x
			dy := points[i].y - points[j].y
			d := math.Hypot(dx, dy)
			if d < best {
				best = d
			}
		}
		if !math.IsNaN(best) && !math.IsInf(best, 0) {
			sum += best
			count++
		}
	}
	if count == 0 {
		return 0.04
	}
	return sum / float64(count)
}

func sampleCoarseTerrain(
	x, y float64,
	pos Vector3D,
	points []localPlanePoint,
	elevation []float64,
	refine *TerrainRefinementScaffold,
) (nearest int, base, slopeTerm, reliefAmp float64) {
	type neighbor struct {
		idx  int
		dist float64
	}
	nearest = -1
	best := math.Inf(1)
	all := make([]neighbor, 0, len(points))
	for _, p := range points {
		d := math.Hypot(x-p.x, y-p.y)
		all = append(all, neighbor{idx: p.idx, dist: d})
		if d < best {
			best = d
			nearest = p.idx
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].dist < all[j].dist })
	limit := 4
	if len(all) < limit {
		limit = len(all)
	}
	weightSum := 0.0
	for i := 0; i < limit; i++ {
		w := 1.0 / math.Max(1e-4, all[i].dist*all[i].dist)
		base += w * elevation[all[i].idx]
		weightSum += w
	}
	if weightSum > 0 {
		base /= weightSum
	}
	if nearest >= 0 && nearest < len(refine.Cells) {
		c := refine.Cells[nearest]
		reliefAmp = Clamp(0.12*c.LocalRelief, 20, 220)
		dirX := math.Sin(c.DownslopeBearingDeg * math.Pi / 180)
		dirY := math.Cos(c.DownslopeBearingDeg * math.Pi / 180)
		var nx, ny float64
		for _, p := range points {
			if p.idx == nearest {
				nx, ny = p.x, p.y
				break
			}
		}
		proj := (x-nx)*dirX + (y-ny)*dirY
		slopeTerm = Clamp(0.18*c.DownslopeStrength*proj, -180, 180)
	}
	_ = pos
	return nearest, base, slopeTerm, reliefAmp
}

func collectChannelEdges(
	points []localPlanePoint,
	hydro *HydrologyScaffold,
	elevation []float64,
	sideflowSplitScale float64,
) []localChannelEdge {
	loc := make(map[int][2]float64, len(points))
	pointSet := make(map[int]bool, len(points))
	for _, p := range points {
		loc[p.idx] = [2]float64{p.x, p.y}
		pointSet[p.idx] = true
	}
	out := make([]localChannelEdge, 0)
	for idx, receiver := range hydro.Receivers {
		if idx < 0 || idx >= len(elevation) || elevation[idx] <= 0 {
			continue
		}
		if receiver < 0 || receiver >= len(elevation) {
			continue
		}
		aok := pointSet[idx]
		bok := pointSet[receiver]
		if !aok && !bok {
			continue
		}
		a := loc[idx]
		b := loc[receiver]
		strength := 0.0
		if idx < len(hydro.ChannelStrength) {
			strength = hydro.ChannelStrength[idx]
		}
		if strength <= 0 {
			continue
		}
		out = append(out, localChannelEdge{
			ax: a[0], ay: a[1],
			bx: b[0], by: b[1],
			strength: strength,
		})
		if idx < len(hydro.MaxOutflows) && hydro.MaxOutflows[idx] > 1 && idx < len(hydro.BoundarySideFlow) {
			origin := a
			for _, side := range hydro.BoundarySideFlow[idx] {
				if side.OutflowStrength <= 0 {
					continue
				}
				dirRad := side.BearingCenterDeg * math.Pi / 180
				length := 0.9 * estimatePointRadius(points, idx)
				if length <= 0 {
					length = 0.04
				}
				bx := origin[0] + length*math.Sin(dirRad)
				by := origin[1] + length*math.Cos(dirRad)
				branchStrength := strength * side.OutflowStrength / math.Max(hydro.Accumulation[idx], 1e-6)
				branchStrength *= Clamp(sideflowSplitScale, 0.1, 1.0)
				out = append(out, localChannelEdge{
					ax: origin[0], ay: origin[1],
					bx: bx, by: by,
					strength: branchStrength,
				})
			}
		}
	}
	return out
}

func collectLakeSpillEdges(
	points []localPlanePoint,
	hydro *HydrologyScaffold,
	elevation []float64,
	cellSpacing float64,
) []localChannelEdge {
	loc := make(map[int][2]float64, len(points))
	pointSet := make(map[int]bool, len(points))
	for _, p := range points {
		loc[p.idx] = [2]float64{p.x, p.y}
		pointSet[p.idx] = true
	}
	out := make([]localChannelEdge, 0)
	for _, p := range points {
		idx := p.idx
		if idx < 0 || idx >= len(elevation) || idx >= len(hydro.CellClass) || idx >= len(hydro.OutletMode) {
			continue
		}
		class := hydro.CellClass[idx]
		actualWater := elevation[idx] <= 0
		if !actualWater && class != "endorheic_basin" && class != "lake_complex" {
			continue
		}
		if hydro.OutletMode[idx] == "none" {
			continue
		}
		strength := 0.0
		if idx < len(hydro.ChannelStrength) {
			strength = math.Max(strength, hydro.ChannelStrength[idx])
		}
		if idx < len(hydro.Accumulation) {
			strength = math.Max(strength, 0.04*hydro.Accumulation[idx])
		}
		strength = Clamp(strength, 0.35, 2.4)
		if idx < len(hydro.BoundaryFlow) {
			flow := hydro.BoundaryFlow[idx]
			if flow.OutflowNeighbor >= 0 {
				if end, ok := loc[flow.OutflowNeighbor]; ok {
					out = append(out, localChannelEdge{
						ax: p.x, ay: p.y,
						bx: end[0], by: end[1],
						strength: 1.15 * strength,
					})
					continue
				}
			}
			if flow.OutflowStrength > 0 {
				dirRad := flow.OutflowBearingDeg * math.Pi / 180
				length := 1.2 * math.Max(estimatePointRadius(points, idx), 0.65*cellSpacing)
				out = append(out, localChannelEdge{
					ax: p.x, ay: p.y,
					bx:       p.x + length*math.Sin(dirRad),
					by:       p.y + length*math.Cos(dirRad),
					strength: strength,
				})
				continue
			}
		}
		if idx < len(hydro.BoundarySideFlow) {
			for _, side := range hydro.BoundarySideFlow[idx] {
				if side.OutflowStrength <= 0 {
					continue
				}
				dirRad := side.BearingCenterDeg * math.Pi / 180
				length := 1.1 * math.Max(estimatePointRadius(points, idx), 0.6*cellSpacing)
				out = append(out, localChannelEdge{
					ax: p.x, ay: p.y,
					bx:       p.x + length*math.Sin(dirRad),
					by:       p.y + length*math.Cos(dirRad),
					strength: 0.85 * strength,
				})
			}
		}
	}
	return out
}

func collectContractChannelEdges(
	points []localPlanePoint,
	refine *TerrainRefinementScaffold,
	centerCell int,
	contract *SharedBoundaryContract,
	cellSpacing float64,
) []localChannelEdge {
	if contract == nil || centerCell < 0 || centerCell >= len(refine.Cells) {
		return nil
	}
	loc := make(map[int][2]float64, len(points))
	for _, p := range points {
		loc[p.idx] = [2]float64{p.x, p.y}
	}
	centerLoc, ok := loc[centerCell]
	if !ok {
		return nil
	}
	boundaryByNeighbor := make(map[int]TerrainBoundaryConstraint, len(refine.Cells[centerCell].Boundary))
	for _, boundary := range refine.Cells[centerCell].Boundary {
		boundaryByNeighbor[boundary.Neighbor] = boundary
	}
	out := make([]localChannelEdge, 0)
	for _, crossing := range contract.Crossings {
		neighbor := -1
		offset := 0.0
		if crossing.CellA == centerCell {
			neighbor = crossing.CellB
			offset = crossing.OffsetA
		} else if crossing.CellB == centerCell {
			neighbor = crossing.CellA
			offset = crossing.OffsetB
		} else {
			continue
		}
		neighborLoc, ok := loc[neighbor]
		if !ok {
			continue
		}
		boundary, ok := boundaryByNeighbor[neighbor]
		if !ok {
			continue
		}
		mx := 0.5 * (centerLoc[0] + neighborLoc[0])
		my := 0.5 * (centerLoc[1] + neighborLoc[1])
		dx := neighborLoc[0] - centerLoc[0]
		dy := neighborLoc[1] - centerLoc[1]
		dist := math.Hypot(dx, dy)
		if dist <= 1e-6 {
			continue
		}
		tx := -dy / dist
		ty := dx / dist
		nx := dx / dist
		ny := dy / dist
		cx := mx + tx*offset
		cy := my + ty*offset
		inward := math.Max(0.16*dist, 0.35*cellSpacing)
		outward := math.Max(0.08*dist, 0.20*cellSpacing)
		strength := Clamp(1.15*crossing.Strength, 0.20, 2.50)
		if boundary.CrossingStrength > 0 {
			strength = math.Max(strength, boundary.CrossingStrength)
		}
		out = append(out, localChannelEdge{
			ax:       cx - nx*inward,
			ay:       cy - ny*inward,
			bx:       cx + nx*outward,
			by:       cy + ny*outward,
			strength: strength,
		})
	}
	return out
}

func channelInfluence(x, y float64, edges []localChannelEdge, spacing float64) float64 {
	if spacing <= 0 {
		spacing = 0.03
	}
	best := 0.0
	for _, edge := range edges {
		d := distanceToSegment(x, y, edge.ax, edge.ay, edge.bx, edge.by)
		width := spacing * (0.07 + 0.025*math.Min(edge.strength, 6.0))
		v := Clamp(0.28*edge.strength, 0, 1.0) * math.Exp(-0.5*(d*d)/(width*width))
		if v > best {
			best = v
		}
	}
	return best
}

func channelCarveDepth(x, y float64, edges []localChannelEdge, spacing, depthScale float64) float64 {
	if spacing <= 0 {
		spacing = 0.03
	}
	maxDepth := 0.0
	for _, edge := range edges {
		d := distanceToSegment(x, y, edge.ax, edge.ay, edge.bx, edge.by)
		width := spacing * (0.14 + 0.06*math.Min(edge.strength, 6.0))
		depth := depthScale * math.Min(edge.strength, 4.0) * math.Exp(-0.5*(d*d)/(width*width))
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth
}

func distanceToSegment(px, py, ax, ay, bx, by float64) float64 {
	dx := bx - ax
	dy := by - ay
	den := dx*dx + dy*dy
	if den <= 1e-12 {
		return math.Hypot(px-ax, py-ay)
	}
	t := ((px-ax)*dx + (py-ay)*dy) / den
	t = Clamp(t, 0, 1)
	cx := ax + t*dx
	cy := ay + t*dy
	return math.Hypot(px-cx, py-cy)
}

func sampleBoundaryMismatch(
	patch *LocalRefinementPatch,
	points []localPlanePoint,
	refine *TerrainRefinementScaffold,
) (mean, max float64, count int) {
	loc := make(map[int][2]float64, len(points))
	for _, p := range points {
		loc[p.idx] = [2]float64{p.x, p.y}
	}
	total := 0.0
	for _, p := range points {
		if p.idx < 0 || p.idx >= len(refine.Cells) {
			continue
		}
		for _, b := range refine.Cells[p.idx].Boundary {
			neighborLoc, ok := loc[b.Neighbor]
			if !ok {
				continue
			}
			mx := 0.5 * (p.x + neighborLoc[0])
			my := 0.5 * (p.y + neighborLoc[1])
			refined := samplePatchNearest(patch, mx, my)
			err := math.Abs(refined - b.BoundaryElevation)
			total += err
			count++
			if err > max {
				max = err
			}
		}
	}
	if count > 0 {
		mean = total / float64(count)
	}
	return mean, max, count
}

func samplePatchNearest(patch *LocalRefinementPatch, x, y float64) float64 {
	tx := 0.0
	ty := 0.0
	if patch.XMax > patch.XMin {
		tx = (x - patch.XMin) / (patch.XMax - patch.XMin)
	}
	if patch.YMax > patch.YMin {
		ty = (patch.YMax - y) / (patch.YMax - patch.YMin)
	}
	px := int(Clamp(math.Round(tx*float64(patch.Width-1)), 0, float64(patch.Width-1)))
	py := int(Clamp(math.Round(ty*float64(patch.Height-1)), 0, float64(patch.Height-1)))
	return patch.Elevation[py*patch.Width+px]
}

type localLakeField struct {
	x, y     float64
	radius   float64
	surface  float64
	strength float64
}

func buildLakeFields(
	points []localPlanePoint,
	elevation []float64,
	hydro *HydrologyScaffold,
	refine *TerrainRefinementScaffold,
	cellSpacing float64,
) []localLakeField {
	fields := make([]localLakeField, 0)
	for _, p := range points {
		if p.idx < 0 || p.idx >= len(elevation) || p.idx >= len(hydro.CellClass) {
			continue
		}
		class := hydro.CellClass[p.idx]
		actualWater := elevation[p.idx] <= 0
		if !actualWater && class != "endorheic_basin" && class != "lake_complex" {
			continue
		}
		radius := 0.55 * cellSpacing
		strength := 1.0
		surface := elevation[p.idx]
		if p.idx < len(refine.Cells) {
			surface = math.Min(surface, refine.Cells[p.idx].BaseElevation)
		}
		switch {
		case actualWater && class == "lake_complex":
			radius = 0.70 * cellSpacing
			strength = 0.85
		case actualWater:
			radius = 0.60 * cellSpacing
			strength = 1.0
		case class == "lake_complex":
			radius = 0.48 * cellSpacing
			strength = 0.45
			surface -= 8
		case class == "endorheic_basin":
			radius = 0.42 * cellSpacing
			strength = 0.55
			surface -= 12
		}
		fields = append(fields, localLakeField{
			x: p.x, y: p.y,
			radius:   radius,
			surface:  surface,
			strength: strength,
		})
	}
	return fields
}

func sampleLakeField(x, y float64, fields []localLakeField) (mask, surface float64) {
	bestMask := 0.0
	bestSurface := 0.0
	for _, field := range fields {
		d := math.Hypot(x-field.x, y-field.y)
		m := 1.0 - SmoothStep(0.55*field.radius, field.radius, d)
		m *= field.strength
		if m > bestMask {
			bestMask = m
			bestSurface = field.surface
		}
	}
	return bestMask, bestSurface
}

func estimatePointRadius(points []localPlanePoint, idx int) float64 {
	best := math.Inf(1)
	var px, py float64
	found := false
	for _, p := range points {
		if p.idx == idx {
			px, py = p.x, p.y
			found = true
			break
		}
	}
	if !found {
		return 0.04
	}
	for _, p := range points {
		if p.idx == idx {
			continue
		}
		d := math.Hypot(px-p.x, py-p.y)
		if d < best {
			best = d
		}
	}
	if !math.IsInf(best, 0) && !math.IsNaN(best) {
		return 0.5 * best
	}
	return 0.04
}

func extractBoundaryCrossings(patch *LocalRefinementPatch) []LocalBoundaryCrossing {
	if patch == nil || patch.Width < 2 || patch.Height < 2 {
		return nil
	}
	type boundarySample struct {
		side     string
		k        int
		width    int
		strength float64
		kind     string
	}
	samples := make([]boundarySample, 0)
	addSide := func(side string, fixed int, horizontal bool) {
		runStart := -1
		totalStrength := 0.0
		runKind := ""
		count := 0
		flush := func(end int) {
			if runStart < 0 || count == 0 {
				runStart = -1
				totalStrength = 0
				runKind = ""
				count = 0
				return
			}
			center := (runStart + end - 1) / 2
			samples = append(samples, boundarySample{
				side:     side,
				k:        center,
				width:    count,
				strength: totalStrength / float64(count),
				kind:     runKind,
			})
			runStart = -1
			totalStrength = 0
			runKind = ""
			count = 0
		}
		limit := patch.Width
		if !horizontal {
			limit = patch.Height
		}
		for i := 0; i < limit; i++ {
			px, py := i, fixed
			if !horizontal {
				px, py = fixed, i
			}
			idx := py*patch.Width + px
			channel := patch.ChannelMask[idx]
			lake := patch.LakeSignal[idx]
			qualifies := channel > 0.22 || (patch.WaterSignal[idx] > 0.35 && channel > 0.10)
			if !qualifies {
				flush(i)
				continue
			}
			if runStart < 0 {
				runStart = i
			}
			totalStrength += math.Max(channel, 0.5*lake)
			if lake > channel {
				runKind = "lake_outlet"
			} else if runKind == "" {
				runKind = "channel"
			}
			count++
		}
		flush(limit)
	}
	addSide("north", 0, true)
	addSide("south", patch.Height-1, true)
	addSide("west", 0, false)
	addSide("east", patch.Width-1, false)

	out := make([]LocalBoundaryCrossing, 0, len(samples))
	for _, sample := range samples {
		var px, py int
		switch sample.side {
		case "north":
			px, py = sample.k, 0
		case "south":
			px, py = sample.k, patch.Height-1
		case "west":
			px, py = 0, sample.k
		case "east":
			px, py = patch.Width-1, sample.k
		}
		x, y := patchPixelToLocal(patch, px, py)
		bearing := math.Atan2(x, y) * 180 / math.Pi
		if bearing < 0 {
			bearing += 360
		}
		out = append(out, LocalBoundaryCrossing{
			Side:       sample.side,
			BearingDeg: bearing,
			WidthPx:    sample.width,
			Strength:   sample.strength,
			Kind:       sample.kind,
		})
	}
	return out
}

func patchPixelToLocal(patch *LocalRefinementPatch, px, py int) (x, y float64) {
	tx := 0.0
	ty := 0.0
	if patch.Width > 1 {
		tx = float64(px) / float64(patch.Width-1)
	}
	if patch.Height > 1 {
		ty = float64(py) / float64(patch.Height-1)
	}
	x = Lerp(patch.XMin, patch.XMax, tx)
	y = Lerp(patch.YMax, patch.YMin, ty)
	return x, y
}

func extractCellBoundaryCrossings(
	patch *LocalRefinementPatch,
	points []localPlanePoint,
	refine *TerrainRefinementScaffold,
	centerCell int,
) []LocalCellBoundaryCrossing {
	if centerCell < 0 || centerCell >= len(refine.Cells) {
		return nil
	}
	loc := make(map[int][2]float64, len(points))
	for _, p := range points {
		loc[p.idx] = [2]float64{p.x, p.y}
	}
	centerLoc, ok := loc[centerCell]
	if !ok {
		return nil
	}
	out := make([]LocalCellBoundaryCrossing, 0)
	for _, boundary := range refine.Cells[centerCell].Boundary {
		neighborLoc, ok := loc[boundary.Neighbor]
		if !ok {
			continue
		}
		mx := 0.5 * (centerLoc[0] + neighborLoc[0])
		my := 0.5 * (centerLoc[1] + neighborLoc[1])
		dx := neighborLoc[0] - centerLoc[0]
		dy := neighborLoc[1] - centerLoc[1]
		dist := math.Hypot(dx, dy)
		if dist <= 1e-6 {
			continue
		}
		tx := -dy / dist
		ty := dx / dist
		halfSpan := 0.30 * dist
		const samples = 25
		runStart := -1
		runStrength := 0.0
		runKind := ""
		runCount := 0
		flush := func(end int) {
			if runStart < 0 || runCount == 0 {
				runStart = -1
				runStrength = 0
				runKind = ""
				runCount = 0
				return
			}
			centerSample := 0.5 * float64(runStart+end-1)
			alpha := 0.0
			if samples > 1 {
				alpha = centerSample / float64(samples-1)
			}
			offset := Lerp(-halfSpan, halfSpan, alpha)
			out = append(out, LocalCellBoundaryCrossing{
				Neighbor:   boundary.Neighbor,
				BearingDeg: boundary.BearingDeg,
				Offset:     offset,
				WidthPx:    runCount,
				Strength:   runStrength / float64(runCount),
				Kind:       runKind,
			})
			runStart = -1
			runStrength = 0
			runKind = ""
			runCount = 0
		}
		for i := 0; i < samples; i++ {
			alpha := 0.0
			if samples > 1 {
				alpha = float64(i) / float64(samples-1)
			}
			offset := Lerp(-halfSpan, halfSpan, alpha)
			x := mx + tx*offset
			y := my + ty*offset
			channel := samplePatchFieldNearest(patch.ChannelMask, patch, x, y)
			lake := samplePatchFieldNearest(patch.LakeSignal, patch, x, y)
			water := samplePatchFieldNearest(patch.WaterSignal, patch, x, y)
			qualifies := channel > 0.16 || (water > 0.20 && channel > 0.08)
			if !qualifies {
				flush(i)
				continue
			}
			if runStart < 0 {
				runStart = i
			}
			runStrength += math.Max(channel, 0.5*lake)
			if lake > channel {
				runKind = "lake_outlet"
			} else if runKind == "" {
				runKind = "channel"
			}
			runCount++
		}
		flush(samples)
	}
	return out
}

func samplePatchFieldNearest(field []float64, patch *LocalRefinementPatch, x, y float64) float64 {
	tx := 0.0
	ty := 0.0
	if patch.XMax > patch.XMin {
		tx = (x - patch.XMin) / (patch.XMax - patch.XMin)
	}
	if patch.YMax > patch.YMin {
		ty = (patch.YMax - y) / (patch.YMax - patch.YMin)
	}
	px := int(Clamp(math.Round(tx*float64(patch.Width-1)), 0, float64(patch.Width-1)))
	py := int(Clamp(math.Round(ty*float64(patch.Height-1)), 0, float64(patch.Height-1)))
	return field[py*patch.Width+px]
}
