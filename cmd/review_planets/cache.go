package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

const (
	reviewTerrainCacheVersion      = "terrain-v11"
	reviewClimateCacheVersion      = "climate-v11"
	reviewDerivedCacheVersion      = "derived-v12"
	reviewTradeGoodsCacheVersion   = "tradegoods-v2"
	reviewCivilizationCacheVersion = "civilization-v72"
	reviewMaritimeCacheVersion     = "maritime-v48"
	reviewEconomyCacheVersion      = "economy-v50"
)

type reviewCacheStore struct {
	rootDir string
}

type cachedTerrainReview struct {
	Elevation   []float64                           `json:"elevation"`
	IsLand      []bool                              `json:"isLand"`
	Diagnostics terrain.PlanetGenerationDiagnostics `json:"diagnostics"`
}

type cachedDerivedReview struct {
	Biome            *climgen.BiomeResult           `json:"biome,omitempty"`
	Soils            *climgen.SoilResult            `json:"soils,omitempty"`
	Vegetation       *climgen.VegetationResult      `json:"vegetation,omitempty"`
	Agriculture      *climgen.AgricultureResult     `json:"agriculture,omitempty"`
	Wildlife         *climgen.WildlifeResult        `json:"wildlife,omitempty"`
	WaterResources   *climgen.WaterResourceResult   `json:"waterResources,omitempty"`
	CoastalResources *climgen.CoastalResourceResult `json:"coastalResources,omitempty"`
	Resources        *climgen.ResourceResult        `json:"resources,omitempty"`
	Settlement       *climgen.SettlementResult      `json:"settlement,omitempty"`
	Population       *climgen.PopulationResult      `json:"population,omitempty"`
}

type cachedCivilizationReview struct {
	Network     *climgen.SettlementNetworkResult `json:"network,omitempty"`
	Proto       *climgen.ProtoCivilizationResult `json:"proto,omitempty"`
	LandRoutes  *climgen.LandRouteResult         `json:"landRoutes,omitempty"`
	Trade       *climgen.TradeNetworkResult      `json:"trade,omitempty"`
	RiverRoutes *climgen.RiverRouteResult        `json:"riverRoutes,omitempty"`
	RiverTrade  *climgen.RiverTradeResult        `json:"riverTrade,omitempty"`
	Polities    *climgen.PolitySphereResult      `json:"polities,omitempty"`
	Profiles    *climgen.PolityProfileResult     `json:"profiles,omitempty"`
}

type cachedMaritimeReview struct {
	CoastalPorts *climgen.CoastalPortResult  `json:"coastalPorts,omitempty"`
	CoastalTrade *climgen.CoastalTradeResult `json:"coastalTrade,omitempty"`
	OceanTrade   *climgen.OceanTradeResult   `json:"oceanTrade,omitempty"`
}

type cachedEconomyReview struct {
	NodeGoods   *climgen.NodeGoodsResult       `json:"nodeGoods,omitempty"`
	PolityGoods *climgen.PolityGoodsResult     `json:"polityGoods,omitempty"`
	NodeMarkets *climgen.TradeNodeMarketResult `json:"nodeMarkets,omitempty"`
	Multimodal  *climgen.MultimodalTradeResult `json:"multimodal,omitempty"`
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

func (s *reviewCacheStore) derivedPath(key string) string {
	return filepath.Join(s.rootDir, "derived", key+".json")
}

func (s *reviewCacheStore) tradeGoodsPath(key string) string {
	return filepath.Join(s.rootDir, "trade_goods", key+".json")
}

func (s *reviewCacheStore) civilizationPath(key string) string {
	return filepath.Join(s.rootDir, "civilization", key+".json")
}

func (s *reviewCacheStore) maritimePath(key string) string {
	return filepath.Join(s.rootDir, "maritime", key+".json")
}

func (s *reviewCacheStore) economyPath(key string) string {
	return filepath.Join(s.rootDir, "economy", key+".json")
}

func terrainCacheKey(level, plates int, landFrac float64, seed int64) string {
	raw := fmt.Sprintf("%s|level=%d|plates=%d|land=%.6f|seed=%d", reviewTerrainCacheVersion, level, plates, landFrac, seed)
	return stableCacheKey(raw)
}

func climateCacheKey(terrainKey string, seed int64) string {
	raw := fmt.Sprintf("%s|terrain=%s|seed=%d|numSeasons=%d|numCycles=%d|refEq=%t", reviewClimateCacheVersion, terrainKey, seed, 4, 3, true)
	return stableCacheKey(raw)
}

func derivedCacheKey(terrainKey, climateKey string, climateHydrology bool, settingsDigest string) string {
	raw := fmt.Sprintf(
		"%s|terrain=%s|climate=%s|hydrology=%t|settings=%s",
		reviewDerivedCacheVersion,
		terrainKey,
		climateKey,
		climateHydrology,
		settingsDigest,
	)
	return stableCacheKey(raw)
}

func tradeGoodsCacheKey(derivedKey, settingsDigest string) string {
	raw := fmt.Sprintf("%s|derived=%s|settings=%s", reviewTradeGoodsCacheVersion, derivedKey, settingsDigest)
	return stableCacheKey(raw)
}

func civilizationCacheKey(derivedKey, settingsDigest string) string {
	raw := fmt.Sprintf("%s|derived=%s|settings=%s", reviewCivilizationCacheVersion, derivedKey, settingsDigest)
	return stableCacheKey(raw)
}

func maritimeCacheKey(civilizationKey, vesselName, settingsDigest string) string {
	raw := fmt.Sprintf("%s|civilization=%s|vessel=%s|settings=%s", reviewMaritimeCacheVersion, civilizationKey, vesselName, settingsDigest)
	return stableCacheKey(raw)
}

func economyCacheKey(civilizationKey, maritimeKey, settingsDigest string) string {
	raw := fmt.Sprintf("%s|civilization=%s|maritime=%s|settings=%s", reviewEconomyCacheVersion, civilizationKey, maritimeKey, settingsDigest)
	return stableCacheKey(raw)
}

func cacheSettingsDigest(values ...any) string {
	hash := sha256.New()
	enc := json.NewEncoder(hash)
	for _, value := range values {
		if err := enc.Encode(value); err != nil {
			panic(fmt.Sprintf("encode cache settings: %v", err))
		}
	}
	return hex.EncodeToString(hash.Sum(nil)[:8])
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

func (s *reviewCacheStore) LoadDerived(key string) (*cachedDerivedReview, bool, error) {
	var cached cachedDerivedReview
	ok, err := s.readJSON(s.derivedPath(key), &cached)
	if err != nil || !ok {
		return nil, ok, err
	}
	return &cached, true, nil
}

func (s *reviewCacheStore) SaveDerived(key string, cached *cachedDerivedReview) error {
	if cached == nil {
		return nil
	}
	return s.writeJSON(s.derivedPath(key), cached)
}

func (s *reviewCacheStore) LoadTradeGoods(key string) (*climgen.TradeGoodResult, bool, error) {
	var cached climgen.TradeGoodResult
	ok, err := s.readJSON(s.tradeGoodsPath(key), &cached)
	if err != nil || !ok {
		return nil, ok, err
	}
	return &cached, true, nil
}

func (s *reviewCacheStore) SaveTradeGoods(key string, cached *climgen.TradeGoodResult) error {
	if cached == nil {
		return nil
	}
	return s.writeJSON(s.tradeGoodsPath(key), cached)
}

func (s *reviewCacheStore) LoadCivilization(key string) (*cachedCivilizationReview, bool, error) {
	var cached cachedCivilizationReview
	ok, err := s.readJSON(s.civilizationPath(key), &cached)
	if err != nil || !ok {
		return nil, ok, err
	}
	return &cached, true, nil
}

func (s *reviewCacheStore) SaveCivilization(key string, cached *cachedCivilizationReview) error {
	if cached == nil {
		return nil
	}
	return s.writeJSON(s.civilizationPath(key), cached)
}

func (s *reviewCacheStore) LoadMaritime(key string) (*cachedMaritimeReview, bool, error) {
	var cached cachedMaritimeReview
	ok, err := s.readJSON(s.maritimePath(key), &cached)
	if err != nil || !ok {
		return nil, ok, err
	}
	return &cached, true, nil
}

func (s *reviewCacheStore) SaveMaritime(key string, cached *cachedMaritimeReview) error {
	if cached == nil {
		return nil
	}
	return s.writeJSON(s.maritimePath(key), cached)
}

func (s *reviewCacheStore) LoadEconomy(key string) (*cachedEconomyReview, bool, error) {
	var cached cachedEconomyReview
	ok, err := s.readJSON(s.economyPath(key), &cached)
	if err != nil || !ok {
		return nil, ok, err
	}
	return &cached, true, nil
}

func (s *reviewCacheStore) SaveEconomy(key string, cached *cachedEconomyReview) error {
	if cached == nil {
		return nil
	}
	return s.writeJSON(s.economyPath(key), cached)
}

func (s *reviewCacheStore) writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	sanitized := sanitizeCacheJSONValue(reflect.ValueOf(value))
	data, err := json.Marshal(sanitized.Interface())
	if err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
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

func sanitizeCacheJSONValue(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.New(v.Type().Elem())
		out.Elem().Set(sanitizeCacheJSONValue(v.Elem()))
		return out
	case reflect.Interface:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		sanitized := sanitizeCacheJSONValue(v.Elem())
		out := reflect.New(v.Type()).Elem()
		out.Set(sanitized)
		return out
	case reflect.Struct:
		out := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			field := out.Field(i)
			if !field.CanSet() {
				continue
			}
			// Skip fields excluded from JSON. Walking them is wasted work on
			// large per-cell arrays, and actively harmful: the float pass below
			// maps NaN to 0, so a field using NaN to mean "undefined" would cache
			// as a real zero. For a distance field that reads as "on the
			// boundary" -- the most misleading value available.
			if tag, ok := v.Type().Field(i).Tag.Lookup("json"); ok && strings.HasPrefix(tag, "-") {
				continue
			}
			field.Set(sanitizeCacheJSONValue(v.Field(i)))
		}
		return out
	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(sanitizeCacheJSONValue(v.Index(i)))
		}
		return out
	case reflect.Array:
		out := reflect.New(v.Type()).Elem()
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(sanitizeCacheJSONValue(v.Index(i)))
		}
		return out
	case reflect.Map:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			out.SetMapIndex(iter.Key(), sanitizeCacheJSONValue(iter.Value()))
		}
		return out
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if math.IsNaN(f) {
			return reflect.ValueOf(0.0).Convert(v.Type())
		}
		if math.IsInf(f, 1) {
			return reflect.ValueOf(1e30).Convert(v.Type())
		}
		if math.IsInf(f, -1) {
			return reflect.ValueOf(-1e30).Convert(v.Type())
		}
		return v
	default:
		return v
	}
}
