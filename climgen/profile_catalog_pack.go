package climgen

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

const ProfileCatalogPackSchemaVersion = "profile-catalog-pack/v1"

type ProfileCatalogPack struct {
	SchemaVersion    string   `json:"schemaVersion"`
	AncestryFiles    []string `json:"ancestryFiles,omitempty"`
	CultureFiles     []string `json:"cultureFiles,omitempty"`
	CompositionFiles []string `json:"compositionFiles,omitempty"`
}

type profileCatalogHeader struct {
	SchemaVersion string `json:"schemaVersion"`
}

type profileCompositionFile struct {
	Compositions []ProfileCompositionSpec `json:"compositions,omitempty"`
}

func LoadProfileCatalog(path string) (*ProfileCatalog, error) {
	clean := filepath.Clean(path)
	if filepath.IsAbs(clean) {
		return LoadProfileCatalogFromFS(os.DirFS(filepath.Dir(clean)), filepath.Base(clean))
	}
	return LoadProfileCatalogFromFS(os.DirFS("."), filepath.ToSlash(clean))
}

func LoadProfileCatalogFromFS(fsys fs.FS, name string) (*ProfileCatalog, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}
	schema, err := detectProfileCatalogSchema(data)
	if err != nil {
		return nil, err
	}
	switch schema {
	case ProfileCatalogSchemaVersion:
		return loadProfileCatalogData(data)
	case ProfileCatalogPackSchemaVersion:
		return loadProfileCatalogPackFromFS(fsys, name, data)
	default:
		return nil, fmt.Errorf("unsupported profile catalog schemaVersion %q", schema)
	}
}

func detectProfileCatalogSchema(data []byte) (string, error) {
	var header profileCatalogHeader
	if err := json.Unmarshal(data, &header); err != nil {
		return "", fmt.Errorf("decode profile catalog header: %w", err)
	}
	if header.SchemaVersion == "" {
		return "", fmt.Errorf("profile catalog schemaVersion is required")
	}
	return header.SchemaVersion, nil
}

func loadProfileCatalogPackFromFS(fsys fs.FS, name string, data []byte) (*ProfileCatalog, error) {
	var pack ProfileCatalogPack
	if err := json.Unmarshal(data, &pack); err != nil {
		return nil, fmt.Errorf("decode profile catalog pack: %w", err)
	}
	if pack.SchemaVersion != ProfileCatalogPackSchemaVersion {
		return nil, fmt.Errorf("unsupported profile catalog pack schemaVersion %q", pack.SchemaVersion)
	}
	catalog := &ProfileCatalog{SchemaVersion: ProfileCatalogSchemaVersion}
	baseDir := path.Dir(name)
	for _, rel := range pack.AncestryFiles {
		resolved := resolveProfileCatalogPackPath(baseDir, rel)
		items, err := loadAncestryProfilesFromFS(fsys, resolved)
		if err != nil {
			return nil, err
		}
		catalog.Ancestries = append(catalog.Ancestries, items...)
	}
	for _, rel := range pack.CultureFiles {
		resolved := resolveProfileCatalogPackPath(baseDir, rel)
		items, err := loadCultureProfilesFromFS(fsys, resolved)
		if err != nil {
			return nil, err
		}
		catalog.Cultures = append(catalog.Cultures, items...)
	}
	for _, rel := range pack.CompositionFiles {
		resolved := resolveProfileCatalogPackPath(baseDir, rel)
		items, err := loadProfileCompositionSpecsFromFS(fsys, resolved)
		if err != nil {
			return nil, err
		}
		catalog.Compositions = append(catalog.Compositions, items...)
	}
	if err := ValidateProfileCatalog(catalog); err != nil {
		return nil, err
	}
	return catalog, nil
}

func resolveProfileCatalogPackPath(baseDir, rel string) string {
	return path.Clean(path.Join(baseDir, rel))
}

func loadAncestryProfilesFromFS(fsys fs.FS, name string) ([]AncestryProfile, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("read ancestry profile %q: %w", name, err)
	}
	var one AncestryProfile
	if err := json.Unmarshal(data, &one); err == nil && one.Name != "" {
		return []AncestryProfile{one}, nil
	}
	var many []AncestryProfile
	if err := json.Unmarshal(data, &many); err == nil && len(many) > 0 {
		return many, nil
	}
	return nil, fmt.Errorf("decode ancestry profile %q", name)
}

func loadCultureProfilesFromFS(fsys fs.FS, name string) ([]CultureProfile, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("read culture profile %q: %w", name, err)
	}
	var one CultureProfile
	if err := json.Unmarshal(data, &one); err == nil && one.Name != "" {
		return []CultureProfile{one}, nil
	}
	var many []CultureProfile
	if err := json.Unmarshal(data, &many); err == nil && len(many) > 0 {
		return many, nil
	}
	return nil, fmt.Errorf("decode culture profile %q", name)
}

func loadProfileCompositionSpecsFromFS(fsys fs.FS, name string) ([]ProfileCompositionSpec, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("read composition profile %q: %w", name, err)
	}
	var wrapped profileCompositionFile
	if err := json.Unmarshal(data, &wrapped); err == nil && len(wrapped.Compositions) > 0 {
		return wrapped.Compositions, nil
	}
	var many []ProfileCompositionSpec
	if err := json.Unmarshal(data, &many); err == nil && len(many) > 0 {
		return many, nil
	}
	var one ProfileCompositionSpec
	if err := json.Unmarshal(data, &one); err == nil && one.Name != "" {
		return []ProfileCompositionSpec{one}, nil
	}
	return nil, fmt.Errorf("decode composition profile %q", name)
}
