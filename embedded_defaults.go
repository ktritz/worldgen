package worldgen

import (
	"embed"
	"io/fs"
)

//go:embed config/profile_catalog_fantasy.json config/profiles/fantasy/ancestries/*.json config/profiles/fantasy/cultures/*.json config/profiles/fantasy/compositions/*.json config/settlement_profiles.json config/maritime_vessels_earthlike.json config/maritime_ports_earthlike.json config/coastal_trade_earthlike.json config/ocean_trade_earthlike.json config/trade_goods_earthlike.json
var embeddedDefaultsFS embed.FS

func EmbeddedProfileCatalogFantasy() []byte {
	data, _ := fs.ReadFile(embeddedDefaultsFS, "config/profile_catalog_fantasy.json")
	return append([]byte(nil), data...)
}

func EmbeddedSettlementProfiles() []byte {
	data, _ := fs.ReadFile(embeddedDefaultsFS, "config/settlement_profiles.json")
	return append([]byte(nil), data...)
}

func EmbeddedMaritimeRouteSettings() []byte {
	data, _ := fs.ReadFile(embeddedDefaultsFS, "config/maritime_vessels_earthlike.json")
	return append([]byte(nil), data...)
}

func EmbeddedMaritimePortSettings() []byte {
	data, _ := fs.ReadFile(embeddedDefaultsFS, "config/maritime_ports_earthlike.json")
	return append([]byte(nil), data...)
}

func EmbeddedCoastalTradeSettings() []byte {
	data, _ := fs.ReadFile(embeddedDefaultsFS, "config/coastal_trade_earthlike.json")
	return append([]byte(nil), data...)
}

func EmbeddedOceanTradeSettings() []byte {
	data, _ := fs.ReadFile(embeddedDefaultsFS, "config/ocean_trade_earthlike.json")
	return append([]byte(nil), data...)
}

func EmbeddedTradeGoodsSettings() []byte {
	data, _ := fs.ReadFile(embeddedDefaultsFS, "config/trade_goods_earthlike.json")
	return append([]byte(nil), data...)
}

func EmbeddedDefaultsFS() fs.FS {
	return embeddedDefaultsFS
}
