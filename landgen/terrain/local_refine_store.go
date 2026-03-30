package terrain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// LocalRefinementStore persists refined cell artifacts and shared-boundary
// contracts for one world seed under a reusable directory layout.
type LocalRefinementStore struct {
	RootDir string
	Seed    int64
}

// NewLocalRefinementStore creates a store rooted at `rootDir` for one world
// seed. Files are grouped by seed so multiple worlds can share the same base.
func NewLocalRefinementStore(rootDir string, seed int64) *LocalRefinementStore {
	return &LocalRefinementStore{
		RootDir: rootDir,
		Seed:    seed,
	}
}

func (s *LocalRefinementStore) seedDir() string {
	return filepath.Join(s.RootDir, fmt.Sprintf("seed_%d", s.Seed))
}

func (s *LocalRefinementStore) artifactsDir() string {
	return filepath.Join(s.seedDir(), "artifacts")
}

func (s *LocalRefinementStore) contractsDir() string {
	return filepath.Join(s.seedDir(), "contracts")
}

func (s *LocalRefinementStore) ArtifactPath(cell int) string {
	return filepath.Join(s.artifactsDir(), fmt.Sprintf("cell_%d.json", cell))
}

func (s *LocalRefinementStore) ContractPath(cellA, cellB int) string {
	a, b := canonicalCellPair(cellA, cellB)
	return filepath.Join(s.contractsDir(), fmt.Sprintf("cells_%d_%d.json", a, b))
}

func (s *LocalRefinementStore) SaveArtifact(artifact *LocalRefinementArtifact) error {
	if artifact == nil {
		return nil
	}
	return s.writeJSON(s.ArtifactPath(artifact.Cell), artifact)
}

func (s *LocalRefinementStore) LoadArtifact(cell int) (*LocalRefinementArtifact, error) {
	var artifact LocalRefinementArtifact
	ok, err := s.readJSON(s.ArtifactPath(cell), &artifact)
	if err != nil || !ok {
		return nil, err
	}
	return &artifact, nil
}

func (s *LocalRefinementStore) SaveContract(contract *SharedBoundaryContract) error {
	if contract == nil {
		return nil
	}
	return s.writeJSON(s.ContractPath(contract.CellA, contract.CellB), contract)
}

func (s *LocalRefinementStore) LoadContract(cellA, cellB int) (*SharedBoundaryContract, error) {
	var contract SharedBoundaryContract
	ok, err := s.readJSON(s.ContractPath(cellA, cellB), &contract)
	if err != nil || !ok {
		return nil, err
	}
	return &contract, nil
}

func (s *LocalRefinementStore) LoadContractsForCell(cell int) ([]*SharedBoundaryContract, error) {
	entries, err := os.ReadDir(s.contractsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]*SharedBoundaryContract, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var contract SharedBoundaryContract
		ok, err := s.readJSON(filepath.Join(s.contractsDir(), entry.Name()), &contract)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if contract.CellA == cell || contract.CellB == cell {
			out = append(out, &contract)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CellA == out[j].CellA {
			return out[i].CellB < out[j].CellB
		}
		return out[i].CellA < out[j].CellA
	})
	return out, nil
}

func (s *LocalRefinementStore) QueuedDirtyNeighbors(cell int) ([]int, error) {
	contracts, err := s.LoadContractsForCell(cell)
	if err != nil {
		return nil, err
	}
	neighbors := make([]int, 0)
	for _, contract := range contracts {
		for _, dirty := range BuildRefinementRerunQueue(contract) {
			if dirty != cell {
				neighbors = appendUniqueInt(neighbors, dirty)
			}
		}
	}
	sort.Ints(neighbors)
	return neighbors, nil
}

func (s *LocalRefinementStore) writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *LocalRefinementStore) readJSON(path string, value any) (bool, error) {
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
