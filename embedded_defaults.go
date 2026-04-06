package worldgen

import (
	"embed"
	"io/fs"
)

//go:embed config/profile_catalog_fantasy.json config/profiles/fantasy/ancestries/*.json config/profiles/fantasy/cultures/*.json config/profiles/fantasy/compositions/*.json config/settlement_profiles.json
var embeddedDefaultsFS embed.FS

func EmbeddedProfileCatalogFantasy() []byte {
	data, _ := fs.ReadFile(embeddedDefaultsFS, "config/profile_catalog_fantasy.json")
	return append([]byte(nil), data...)
}

func EmbeddedSettlementProfiles() []byte {
	data, _ := fs.ReadFile(embeddedDefaultsFS, "config/settlement_profiles.json")
	return append([]byte(nil), data...)
}

func EmbeddedDefaultsFS() fs.FS {
	return embeddedDefaultsFS
}
