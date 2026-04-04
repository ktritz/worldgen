package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

const (
	reviewTerrainCacheVersion = "terrain-v1"
	reviewClimateCacheVersion = "climate-v3"
)

type reviewCacheStore struct {
	rootDir string
}

type cachedTerrainReview struct {
	Elevation   []float64                           `json:"elevation"`
	IsLand      []bool                              `json:"isLand"`
	Diagnostics terrain.PlanetGenerationDiagnostics `json:"diagnostics"`
}

func newReviewCacheStore(outputDir string) *reviewCacheStore {
	return &reviewCacheStore{rootDir: filepath.Join(outputDir, "cache")}
}

func (s *reviewCacheStore) terrainPath(key string) string {
	return filepath.Join(s.rootDir, "terrain", key+".json")
}

func (s *reviewCacheStore) climatePath(key string) string {
	return filepath.Join(s.rootDir, "climate", key+".json")
}

func terrainCacheKey(level, plates int, landFrac float64, seed int64) string {
	raw := fmt.Sprintf("%s|level=%d|plates=%d|land=%.6f|seed=%d", reviewTerrainCacheVersion, level, plates, landFrac, seed)
	return stableCacheKey(raw)
}

func climateCacheKey(terrainKey string, seed int64) string {
	raw := fmt.Sprintf("%s|terrain=%s|seed=%d|numSeasons=%d|numCycles=%d|refEq=%t", reviewClimateCacheVersion, terrainKey, seed, 4, 3, true)
	return stableCacheKey(raw)
}

func stableCacheKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:8])
}

func (s *reviewCacheStore) LoadTerrain(key string) (*cachedTerrainReview, bool, error) {
	var cached cachedTerrainReview
	ok, err := s.readJSON(s.terrainPath(key), &cached)
	if err != nil || !ok {
		return nil, ok, err
	}
	return &cached, true, nil
}

func (s *reviewCacheStore) SaveTerrain(key string, cached *cachedTerrainReview) error {
	if cached == nil {
		return nil
	}
	return s.writeJSON(s.terrainPath(key), cached)
}

func (s *reviewCacheStore) LoadClimate(key string) (*climgen.SeasonalClimateResult, bool, error) {
	var cached climgen.SeasonalClimateResult
	ok, err := s.readJSON(s.climatePath(key), &cached)
	if err != nil || !ok {
		return nil, ok, err
	}
	return &cached, true, nil
}

func (s *reviewCacheStore) SaveClimate(key string, cached *climgen.SeasonalClimateResult) error {
	if cached == nil {
		return nil
	}
	return s.writeJSON(s.climatePath(key), cached)
}

func (s *reviewCacheStore) writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *reviewCacheStore) readJSON(path string, value any) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal(data, value); err != nil {
		return false, err
	}
	return true, nil
}
