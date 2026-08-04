package terrain

import (
	"math"
	"math/rand"
)

const terrainStageSeedStride int64 = 0x5851f42d4c957f2d

func terrainStageSeed(seed int64, stage int64) int64 {
	return seed + stage*terrainStageSeedStride
}

func terrainStageRNG(seed int64, stage int64) *rand.Rand {
	return rand.New(rand.NewSource(terrainStageSeed(seed, stage)))
}

func terrainFeatureRNG(seed int64, stage int64, parts ...int64) *rand.Rand {
	mixed := terrainStageSeed(seed, stage)
	for _, part := range parts {
		mixed ^= mixTerrainSeed(part + terrainStageSeedStride)
		mixed = mixTerrainSeed(mixed)
	}
	return rand.New(rand.NewSource(mixed))
}

func terrainVectorSeedPart(v Vector3D) int64 {
	v = v.Normalize()
	x := int64(math.Round(v.X * 1_000_000))
	y := int64(math.Round(v.Y * 1_000_000))
	z := int64(math.Round(v.Z * 1_000_000))
	return mixTerrainSeed(x) ^ mixTerrainSeed(y+0x1000003) ^ mixTerrainSeed(z+0x2000003)
}

func mixTerrainSeed(x int64) int64 {
	u := uint64(x)
	u ^= u >> 30
	u *= 0xbf58476d1ce4e5b9
	u ^= u >> 27
	u *= 0x94d049bb133111eb
	u ^= u >> 31
	return int64(u)
}
