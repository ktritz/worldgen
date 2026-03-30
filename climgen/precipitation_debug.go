package climgen

import "fmt"

func newPrecipitationDebugFields(n int) *PrecipitationDebugFields {
	return &PrecipitationDebugFields{
		OceanFetch:            make([]float64, n),
		CoastalOnshore:        make([]float64, n),
		EffectiveFetch:        make([]float64, n),
		EffectiveOnshore:      make([]float64, n),
		FootprintOceanSupport: make([]float64, n),
		NeighborOceanFraction: make([]float64, n),
		MaritimeSignal:        make([]float64, n),
		MaritimeGeomSupport:   make([]float64, n),
		OceanAtmosphere:       make([]float64, n),
		OceanDownwindLand:     make([]float64, n),
		MarineEntryScale:      make([]float64, n),
		MarineDonor:           make([]float64, n),
		MarineDonorStrength:   make([]float64, n),
		MarineDonorOutgoing:   make([]float64, n),
		MarineDonorOceanAtm:   make([]float64, n),
		MarineDonorDownwind:   make([]float64, n),
		MarineRootSource:      make([]float64, n),
		MarineRootStrength:    make([]float64, n),
		MarineRootOceanAtm:    make([]float64, n),
		MarineRootDownwind:    make([]float64, n),
		MarineRootOceanSource: make([]float64, n),
		MarineRootRetention:   make([]float64, n),
		MarineRootPathSteps:   make([]float64, n),
		UpwindParent:          make([]float64, n),
		UpwindParentStrength:  make([]float64, n),
		LandTravel:            make([]float64, n),
		LandInterior:          make([]float64, n),
		OrographicLift:        make([]float64, n),
		OrographicLocalRise:   make([]float64, n),
		OrographicFootprint:   make([]float64, n),
		OrographicBarrier:     make([]float64, n),
		OrographicWindFactor:  make([]float64, n),
		Convergence:           make([]float64, n),
		MoistureCapacity:      make([]float64, n),
		LandSource:            make([]float64, n),
		TropicalSource:        make([]float64, n),
		FrontalSource:         make([]float64, n),
		MarineIncoming:        make([]float64, n),
		LandIncoming:          make([]float64, n),
		FrontalIncoming:       make([]float64, n),
		MarineToLand:          make([]float64, n),
		MarineToFrontal:       make([]float64, n),
		CondensedTotal:        make([]float64, n),
		CondensedBase:         make([]float64, n),
		CondensedSupersat:     make([]float64, n),
		CondensedSupersatSupport: make([]float64, n),
		CondensedTropicalCoast: make([]float64, n),
		CondensedCoastalPenalty: make([]float64, n),
		CondensedAscent:       make([]float64, n),
		CondensedConvective:   make([]float64, n),
		CondensedMixing:       make([]float64, n),
		CondensedEffCapacity:  make([]float64, n),
		CondensedSupersatHum:  make([]float64, n),
		RetainedHumidity:      make([]float64, n),
		CondensationScale:     make([]float64, n),
		LandRetentionScale:    make([]float64, n),
		FrontalSourceScale:    make([]float64, n),
		FrontalRetentionScale: make([]float64, n),
		TropicalSourceScale:   make([]float64, n),
	}
}

func CloneScaledPrecipitationDebugFields(src *PrecipitationDebugFields, scale float64) *PrecipitationDebugFields {
	if src == nil {
		return nil
	}
	dst := newPrecipitationDebugFields(len(src.OceanFetch))
	copy(dst.OceanFetch, src.OceanFetch)
	copy(dst.CoastalOnshore, src.CoastalOnshore)
	copy(dst.EffectiveFetch, src.EffectiveFetch)
	copy(dst.EffectiveOnshore, src.EffectiveOnshore)
	copy(dst.FootprintOceanSupport, src.FootprintOceanSupport)
	copy(dst.NeighborOceanFraction, src.NeighborOceanFraction)
	copy(dst.MaritimeSignal, src.MaritimeSignal)
	copy(dst.MaritimeGeomSupport, src.MaritimeGeomSupport)
	copy(dst.OceanAtmosphere, src.OceanAtmosphere)
	copy(dst.OceanDownwindLand, src.OceanDownwindLand)
	copy(dst.MarineEntryScale, src.MarineEntryScale)
	copy(dst.MarineDonor, src.MarineDonor)
	copy(dst.MarineDonorStrength, src.MarineDonorStrength)
	copy(dst.MarineDonorOutgoing, src.MarineDonorOutgoing)
	copy(dst.MarineDonorOceanAtm, src.MarineDonorOceanAtm)
	copy(dst.MarineDonorDownwind, src.MarineDonorDownwind)
	copy(dst.MarineRootSource, src.MarineRootSource)
	copy(dst.MarineRootStrength, src.MarineRootStrength)
	copy(dst.MarineRootOceanAtm, src.MarineRootOceanAtm)
	copy(dst.MarineRootDownwind, src.MarineRootDownwind)
	copy(dst.MarineRootOceanSource, src.MarineRootOceanSource)
	copy(dst.MarineRootRetention, src.MarineRootRetention)
	copy(dst.MarineRootPathSteps, src.MarineRootPathSteps)
	copy(dst.UpwindParent, src.UpwindParent)
	copy(dst.UpwindParentStrength, src.UpwindParentStrength)
	copy(dst.LandTravel, src.LandTravel)
	copy(dst.LandInterior, src.LandInterior)
	copy(dst.OrographicLift, src.OrographicLift)
	copy(dst.OrographicLocalRise, src.OrographicLocalRise)
	copy(dst.OrographicFootprint, src.OrographicFootprint)
	copy(dst.OrographicBarrier, src.OrographicBarrier)
	copy(dst.OrographicWindFactor, src.OrographicWindFactor)
	copy(dst.Convergence, src.Convergence)
	copy(dst.MoistureCapacity, src.MoistureCapacity)
	copy(dst.CondensationScale, src.CondensationScale)
	copy(dst.LandRetentionScale, src.LandRetentionScale)
	copy(dst.FrontalSourceScale, src.FrontalSourceScale)
	copy(dst.FrontalRetentionScale, src.FrontalRetentionScale)
	copy(dst.TropicalSourceScale, src.TropicalSourceScale)
	copy(dst.LandSource, src.LandSource)
	copy(dst.TropicalSource, src.TropicalSource)
	copy(dst.FrontalSource, src.FrontalSource)
	copy(dst.MarineIncoming, src.MarineIncoming)
	copy(dst.LandIncoming, src.LandIncoming)
	copy(dst.FrontalIncoming, src.FrontalIncoming)
	copy(dst.MarineToLand, src.MarineToLand)
	copy(dst.MarineToFrontal, src.MarineToFrontal)
	copy(dst.CondensedTotal, src.CondensedTotal)
	copy(dst.CondensedBase, src.CondensedBase)
	copy(dst.CondensedSupersat, src.CondensedSupersat)
	copy(dst.CondensedSupersatSupport, src.CondensedSupersatSupport)
	copy(dst.CondensedTropicalCoast, src.CondensedTropicalCoast)
	copy(dst.CondensedCoastalPenalty, src.CondensedCoastalPenalty)
	copy(dst.CondensedAscent, src.CondensedAscent)
	copy(dst.CondensedConvective, src.CondensedConvective)
	copy(dst.CondensedMixing, src.CondensedMixing)
	copy(dst.CondensedEffCapacity, src.CondensedEffCapacity)
	copy(dst.CondensedSupersatHum, src.CondensedSupersatHum)
	scaleSlice(dst.RetainedHumidity, src.RetainedHumidity, scale)
	return dst
}

func scaleSlice(dst []float64, src []float64, scale float64) {
	for i := range dst {
		if i < len(src) {
			dst[i] = src[i] * scale
		}
	}
}

func FormatPrecipitationDebugCell(
	debug *PrecipitationDebugFields,
	result *PrecipitationResult,
	idx int,
) string {
	if debug == nil || idx < 0 || idx >= len(debug.OceanFetch) {
		return ""
	}
	precip := 0.0
	marinePrecip := 0.0
	landPrecip := 0.0
	frontalPrecip := 0.0
	hasComponents := false
	if result != nil {
		if idx < len(result.Precipitation) {
			precip = result.Precipitation[idx]
		}
		if idx < len(result.MarinePrecipitation) {
			marinePrecip = result.MarinePrecipitation[idx]
			hasComponents = true
		}
		if idx < len(result.LandPrecipitation) {
			landPrecip = result.LandPrecipitation[idx]
			hasComponents = true
		}
		if idx < len(result.FrontalPrecipitation) {
			frontalPrecip = result.FrontalPrecipitation[idx]
			hasComponents = true
		}
	}
	precipLine := fmt.Sprintf("    precip total=%.1f", precip)
	if hasComponents {
		precipLine += fmt.Sprintf(" marine=%.1f land=%.1f frontal=%.1f", marinePrecip, landPrecip, frontalPrecip)
	}
	return fmt.Sprintf(
		"fetch=%.2f->%.2f onshore=%.2f->%.2f foot=%.2f nbrOcean=%.2f mSig=%.2f mGeom=%.2f oAtm=%.2f oLand=%.2f mEntry=%.2f donor=%d@%.2f dOut=%.2f dAtm=%.2f dLand=%.2f root=%d@%.2f rAtm=%.2f rLand=%.2f rSrc=%.2f rRet=%.2f rSteps=%.0f parent=%d@%.2f travel=%.2f interior=%.2f uplift=%.2f localRise=%.0f fpRise=%.0f barrier=%.2f uWind=%.2f conv=%.2f cap=%.2f\n"+
			"    src land=%.2f tropical=%.2f frontal=%.2f scales cond=%.2f retain=%.2f trop=%.2f frontSrc=%.2f frontRet=%.2f\n"+
			"    incoming marine=%.2f land=%.2f frontal=%.2f transfer m->l=%.2f m->f=%.2f retained=%.2f condensed=%.2f [base=%.2f ascent=%.2f supersat=%.2f ssup=%.2f tcoast=%.2f cpen=%.2f conv=%.2f mix=%.2f ecap=%.2f shum=%.2f]\n"+
			"%s",
		debug.OceanFetch[idx],
		debug.EffectiveFetch[idx],
		debug.CoastalOnshore[idx],
		debug.EffectiveOnshore[idx],
		debug.FootprintOceanSupport[idx],
		debug.NeighborOceanFraction[idx],
		debug.MaritimeSignal[idx],
		debug.MaritimeGeomSupport[idx],
		debug.OceanAtmosphere[idx],
		debug.OceanDownwindLand[idx],
		debug.MarineEntryScale[idx],
		int(debug.MarineDonor[idx]),
		debug.MarineDonorStrength[idx],
		debug.MarineDonorOutgoing[idx],
		debug.MarineDonorOceanAtm[idx],
		debug.MarineDonorDownwind[idx],
		int(debug.MarineRootSource[idx]),
		debug.MarineRootStrength[idx],
		debug.MarineRootOceanAtm[idx],
		debug.MarineRootDownwind[idx],
		debug.MarineRootOceanSource[idx],
		debug.MarineRootRetention[idx],
		debug.MarineRootPathSteps[idx],
		int(debug.UpwindParent[idx]),
		debug.UpwindParentStrength[idx],
		debug.LandTravel[idx],
		debug.LandInterior[idx],
		debug.OrographicLift[idx],
		debug.OrographicLocalRise[idx],
		debug.OrographicFootprint[idx],
		debug.OrographicBarrier[idx],
		debug.OrographicWindFactor[idx],
		debug.Convergence[idx],
		debug.MoistureCapacity[idx],
		debug.LandSource[idx],
		debug.TropicalSource[idx],
		debug.FrontalSource[idx],
		debug.CondensationScale[idx],
		debug.LandRetentionScale[idx],
		debug.TropicalSourceScale[idx],
		debug.FrontalSourceScale[idx],
		debug.FrontalRetentionScale[idx],
		debug.MarineIncoming[idx],
		debug.LandIncoming[idx],
		debug.FrontalIncoming[idx],
		debug.MarineToLand[idx],
		debug.MarineToFrontal[idx],
		debug.RetainedHumidity[idx],
		debug.CondensedTotal[idx],
		debug.CondensedBase[idx],
		debug.CondensedAscent[idx],
		debug.CondensedSupersat[idx],
		debug.CondensedSupersatSupport[idx],
		debug.CondensedTropicalCoast[idx],
		debug.CondensedCoastalPenalty[idx],
		debug.CondensedConvective[idx],
		debug.CondensedMixing[idx],
		debug.CondensedEffCapacity[idx],
		debug.CondensedSupersatHum[idx],
		precipLine,
	)
}
