package main

import (
	"fmt"

	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
)

func main() {
	fmt.Println("🔍 Debug: Direct Plate Type Assignment Test")
	fmt.Println("==========================================")

	// Generate a simple icosphere for testing
	fmt.Println("1. Generating icosphere...")
	vertices, faces := icosphere.GenerateIcosphere(3) // Subdivision level 3
	fmt.Printf("   Generated %d vertices, %d faces\n", len(vertices)/3, len(faces)/3)

	// Generate Voronoi cells
	fmt.Println("2. Generating Voronoi cells...")
	voronoiVertices, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	fmt.Printf("   Generated %d Voronoi vertices, %d cells\n", len(voronoiVertices)/3, len(voronoiCells))

	// Set up tectonic settings
	settings := tectonics.TectonicSettings{
		NumPlates:                   12,
		BaseSpeed:                   0.02,
		SpeedVariationFactor:        0.8,
		TargetContinentalProportion: 0.4, // 40% continental target
		NumInitialContinentalSeeds:  4,
		PlanetRadius:                6371000.0, // Earth radius
	}

	fmt.Printf("3. Generating tectonics with settings:\n")
	fmt.Printf("   - Num plates: %d\n", settings.NumPlates)
	fmt.Printf("   - Continental proportion target: %.1f%%\n", settings.TargetContinentalProportion*100)
	fmt.Printf("   - Continental seeds: %d\n", settings.NumInitialContinentalSeeds)

	// Convert icosphere data to format expected by tectonics
	icosphereSites := make([]icosphere.Vector3D, len(vertices)/3)
	for i := 0; i < len(vertices)/3; i++ {
		icosphereSites[i] = icosphere.Vector3D{
			X: vertices[i*3],
			Y: vertices[i*3+1],
			Z: vertices[i*3+2],
		}
	}

	// Generate tectonics data
	tectonicData, err := tectonics.GenerateTectonics(icosphereSites, voronoiCells, settings, 42)
	if err != nil {
		fmt.Printf("❌ Failed to generate tectonics: %v\n", err)
		return
	}

	fmt.Printf("\n4. Analyzing generated plates:\n")
	continentalPlates := 0
	oceanicPlates := 0
	continentalSites := 0
	oceanicSites := 0

	// Count plates by type
	for _, plate := range tectonicData.Plates {
		if plate.PlateType == tectonics.ContinentalPlate {
			continentalPlates++
		} else {
			oceanicPlates++
		}
	}

	// Count sites by plate type
	for siteID := int32(0); siteID < int32(len(icosphereSites)); siteID++ {
		plateID := tectonicData.SitePlateIDs[siteID]
		
		// Find plate
		var assignedPlate *tectonics.TectonicPlate
		for i := range tectonicData.Plates {
			if tectonicData.Plates[i].ID == plateID {
				assignedPlate = &tectonicData.Plates[i]
				break
			}
		}
		
		if assignedPlate != nil {
			if assignedPlate.PlateType == tectonics.ContinentalPlate {
				continentalSites++
			} else {
				oceanicSites++
			}
		}
	}

	fmt.Printf("   Plate counts:\n")
	fmt.Printf("   - Continental plates: %d (%.1f%%)\n", continentalPlates, float64(continentalPlates)/float64(continentalPlates+oceanicPlates)*100)
	fmt.Printf("   - Oceanic plates: %d (%.1f%%)\n", oceanicPlates, float64(oceanicPlates)/float64(continentalPlates+oceanicPlates)*100)
	
	fmt.Printf("   Site assignments:\n")
	fmt.Printf("   - Continental sites: %d (%.1f%%)\n", continentalSites, float64(continentalSites)/float64(len(icosphereSites))*100)
	fmt.Printf("   - Oceanic sites: %d (%.1f%%)\n", oceanicSites, float64(oceanicSites)/float64(len(icosphereSites))*100)

	// Show first few plates with details
	fmt.Printf("\n5. First 5 plates details:\n")
	for i := 0; i < 5 && i < len(tectonicData.Plates); i++ {
		plate := tectonicData.Plates[i]
		fmt.Printf("   Plate %d: Type=%s, Sites=%d, Area=%.1f\n", 
			plate.ID, plate.PlateType, len(plate.SiteIDs), plate.Area)
	}

	if continentalSites == 0 {
		fmt.Printf("\n❌ PROBLEM IDENTIFIED: No continental sites found!\n")
		fmt.Printf("   This explains why all elevations are oceanic.\n")
		
		if continentalPlates > 0 {
			fmt.Printf("   Continental plates exist but may have no sites assigned.\n")
		} else {
			fmt.Printf("   No continental plates were created during generation.\n")
		}
	} else {
		fmt.Printf("\n✅ Continental sites found: %d\n", continentalSites)
	}
}