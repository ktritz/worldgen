package climgen

import (
	"encoding/json"
	"fmt"
	"os"

	worldgen "worldgen"
)

func DefaultTradeGoodsSettings() TradeGoodsSettings {
	settings, err := loadTradeGoodsSettingsData(worldgen.EmbeddedTradeGoodsSettings())
	if err == nil {
		return *settings
	}
	return TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Scarcity:      DefaultTradeGoodsScarcitySettings(),
		Multimodal:    DefaultTradeGoodsMultimodalSettings(),
		Production:    DefaultTradeGoodsProductionSettings(),
		Demand:        DefaultTradeGoodsDemandSettings(),
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.45, Bulkiness: 0.80, Perishability: 0.42, SourceWeights: map[string]float64{"crop": 1}},
			{Name: "timber", Category: "raw", BaseValue: 0.42, Bulkiness: 0.90, Perishability: 0.08, SourceWeights: map[string]float64{"timber": 1}},
			{Name: "iron_ore", Category: "raw", BaseValue: 0.62, Bulkiness: 0.95, Perishability: 0.00, SourceWeights: map[string]float64{"iron_ore": 1}},
		},
	}
}

func LoadTradeGoodsSettings(path string) (TradeGoodsSettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TradeGoodsSettings{}, err
	}
	settings, err := loadTradeGoodsSettingsData(data)
	if err != nil {
		return TradeGoodsSettings{}, err
	}
	return *settings, nil
}

func loadTradeGoodsSettingsData(data []byte) (*TradeGoodsSettings, error) {
	var settings TradeGoodsSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("decode trade goods settings: %w", err)
	}
	settings = settings.withDefaults()
	if err := ValidateTradeGoodsSettings(settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

func (settings TradeGoodsSettings) GoodByName(name string) (TradeGoodSpec, bool) {
	for _, good := range settings.Goods {
		if good.Name == name {
			return good, true
		}
	}
	return TradeGoodSpec{}, false
}

func (settings TradeGoodsSettings) EffectiveScarcitySettings() TradeGoodsScarcitySettings {
	return settings.Scarcity.withDefaults()
}

func (settings TradeGoodsSettings) EffectiveMultimodalSettings() TradeGoodsMultimodalSettings {
	return settings.Multimodal.withDefaults()
}

func (settings TradeGoodsSettings) EffectiveProductionSettings() TradeGoodsProductionSettings {
	return settings.Production.withDefaults()
}

func (settings TradeGoodsSettings) EffectiveDemandSettings() TradeGoodsDemandSettings {
	return settings.Demand.withDefaults()
}
