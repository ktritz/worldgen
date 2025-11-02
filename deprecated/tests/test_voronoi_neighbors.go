package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"worldgen/icosphere"
)

func main() {
	fmt.Println("=== Testing Voronoi Neighbor List Generation ===")
	
	// Parse command line arguments for subdivision level
	subdivisions := 2 // Default value
	if len(os.Args) > 1 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			fmt.Println("Usage: go run test_voronoi_neighbors.go [subdivisions]")
			fmt.Println("  subdivisions: 0-6 (default: 2)")
			fmt.Println("  Expected vertex counts:")
			fmt.Println("    0: 12 vertices")
			fmt.Println("    1: 42 vertices") 
			fmt.Println("    2: 162 vertices")
			fmt.Println("    3: 642 vertices")
			fmt.Println("    4: 2562 vertices")
			fmt.Println("    5: 10242 vertices")
			fmt.Println("    6: 40962 vertices")
			return
		}
		if sub, err := strconv.Atoi(os.Args[1]); err == nil && sub >= 0 && sub <= 6 {
			subdivisions = sub
		} else {
			fmt.Printf("Invalid subdivision level '%s'. Using default: %d\n", os.Args[1], subdivisions)
			fmt.Println("Valid range: 0-6 (use -h for help)")
		}
	}
	
	expectedVertices := 10*subdivisions*subdivisions + 2
	if subdivisions == 0 {
		expectedVertices = 12
	}
	fmt.Printf("Creating icosphere with %d subdivisions (expected: %d vertices)...\n", subdivisions, expectedVertices)
	
	sites, faces := icosphere.CreateIcosphere(subdivisions)
	fmt.Printf("Generated icosphere: %d sites, %d faces\n", len(sites), len(faces))
	
	// Generate Voronoi diagram
	fmt.Println("\nGenerating Voronoi diagram...")
	voronoiVertices, voronoiCells := icosphere.GenerateSphericalVoronoi(sites, faces)
	fmt.Printf("Generated Voronoi: %d vertices, %d cells\n", len(voronoiVertices), len(voronoiCells))
	
	// Test 1: Basic counts validation
	fmt.Println("\n=== Test 1: Basic Validation ===")
	if len(voronoiCells) != len(sites) {
		fmt.Printf("ERROR: Cell count mismatch! Sites: %d, Cells: %d\n", len(sites), len(voronoiCells))
		return
	}
	fmt.Println("✓ Cell count matches site count")
	
	// Test 2: Check if each cell has its correct site index
	fmt.Println("\n=== Test 2: Site Index Validation ===")
	siteIndexMap := make(map[int32]bool)
	for i, cell := range voronoiCells {
		if int(cell.SiteIndex) != i {
			fmt.Printf("ERROR: Cell %d has wrong SiteIndex %d\n", i, cell.SiteIndex)
		}
		siteIndexMap[cell.SiteIndex] = true
	}
	fmt.Printf("✓ All cells have correct site indices (0 to %d)\n", len(voronoiCells)-1)
	
	// Test 3: Neighbor list statistics
	fmt.Println("\n=== Test 3: Neighbor List Analysis ===")
	totalNeighbors := 0
	emptyCells := 0
	minNeighbors := 1000
	maxNeighbors := 0
	
	for i, cell := range voronoiCells {
		neighborCount := len(cell.NeighborSiteIndices)
		totalNeighbors += neighborCount
		
		if neighborCount == 0 {
			emptyCells++
		}
		if neighborCount < minNeighbors {
			minNeighbors = neighborCount
		}
		if neighborCount > maxNeighbors {
			maxNeighbors = neighborCount
		}
		
		// Show first few cells in detail
		if i < 5 {
			fmt.Printf("Cell %d (site %d): %d neighbors: %v\n", 
				i, cell.SiteIndex, neighborCount, cell.NeighborSiteIndices)
		}
	}
	
	avgNeighbors := float64(totalNeighbors) / float64(len(voronoiCells))
	fmt.Printf("\nNeighbor statistics:\n")
	fmt.Printf("  Total cells: %d\n", len(voronoiCells))
	fmt.Printf("  Cells with no neighbors: %d\n", emptyCells)
	fmt.Printf("  Min neighbors: %d\n", minNeighbors)
	fmt.Printf("  Max neighbors: %d\n", maxNeighbors)
	fmt.Printf("  Average neighbors: %.2f\n", avgNeighbors)
	
	if emptyCells > 0 {
		fmt.Printf("⚠️  WARNING: %d cells have no neighbors!\n", emptyCells)
	} else {
		fmt.Println("✓ All cells have neighbors")
	}
	
	// Test 4: Neighbor reciprocity check
	fmt.Println("\n=== Test 4: Neighbor Reciprocity Check ===")
	reciprocityErrors := 0
	invalidNeighbors := 0
	
	// Create lookup map for quick cell finding
	siteToCell := make(map[int32]int)
	for i, cell := range voronoiCells {
		siteToCell[cell.SiteIndex] = i
	}
	
	for i, cell := range voronoiCells {
		for _, neighborSiteIdx := range cell.NeighborSiteIndices {
			// Check if neighbor site index is valid
			if neighborSiteIdx < 0 || int(neighborSiteIdx) >= len(sites) {
				invalidNeighbors++
				if i < 3 {
					fmt.Printf("  Cell %d has invalid neighbor site index: %d\n", i, neighborSiteIdx)
				}
				continue
			}
			
			// Find the neighbor cell
			neighborCellIdx, exists := siteToCell[neighborSiteIdx]
			if !exists {
				invalidNeighbors++
				if i < 3 {
					fmt.Printf("  Cell %d references non-existent neighbor site: %d\n", i, neighborSiteIdx)
				}
				continue
			}
			
			// Check if the relationship is reciprocal
			neighborCell := voronoiCells[neighborCellIdx]
			isReciprocal := false
			for _, backNeighborSiteIdx := range neighborCell.NeighborSiteIndices {
				if backNeighborSiteIdx == cell.SiteIndex {
					isReciprocal = true
					break
				}
			}
			
			if !isReciprocal {
				reciprocityErrors++
				if reciprocityErrors <= 5 {
					fmt.Printf("  Non-reciprocal: Cell %d->%d, but Cell %d does not reference Cell %d\n", 
						i, neighborSiteIdx, neighborCellIdx, cell.SiteIndex)
				}
			}
		}
	}
	
	fmt.Printf("Reciprocity results:\n")
	fmt.Printf("  Invalid neighbor indices: %d\n", invalidNeighbors)
	fmt.Printf("  Non-reciprocal relationships: %d\n", reciprocityErrors)
	
	if invalidNeighbors == 0 && reciprocityErrors == 0 {
		fmt.Println("✓ All neighbor relationships are valid and reciprocal")
	}
	
	// Test 5: Input validation check
	fmt.Println("\n=== Test 5: Input Data Validation ===")
	fmt.Printf("Input faces count: %d\n", len(faces))
	if len(faces) == 0 {
		fmt.Println("ERROR: No Delaunay faces provided to Voronoi generation!")
		return
	}
	
	// Check if face indices are valid
	invalidFaces := 0
	for i, face := range faces {
		if face.V1 < 0 || int(face.V1) >= len(sites) ||
		   face.V2 < 0 || int(face.V2) >= len(sites) ||
		   face.V3 < 0 || int(face.V3) >= len(sites) {
			invalidFaces++
			if i < 3 {
				fmt.Printf("  Invalid face %d: vertices (%d, %d, %d), max site index: %d\n", 
					i, face.V1, face.V2, face.V3, len(sites)-1)
			}
		}
	}
	
	fmt.Printf("Face validation:\n")
	fmt.Printf("  Total faces: %d\n", len(faces))
	fmt.Printf("  Invalid faces: %d\n", invalidFaces)
	
	if invalidFaces == 0 {
		fmt.Println("✓ All face indices are valid")
	} else {
		fmt.Println("ERROR: Invalid face indices found!")
	}
	
	// Test 6: Geometric proximity check
	fmt.Println("\n=== Test 6: Geometric Proximity Check ===")
	fmt.Println("Checking if neighbors are actually geometrically close...")
	
	geometricErrors := 0
	maxValidDistance := 0.0
	totalDistances := 0.0
	distanceCount := 0
	
	// For first few cells, check distances to neighbors vs non-neighbors
	for i := 0; i < 5 && i < len(voronoiCells); i++ {
		cell := voronoiCells[i]
		cellPosition := sites[cell.SiteIndex]
		
		fmt.Printf("\nCell %d analysis:\n", i)
		
		// Calculate distances to declared neighbors
		neighborDistances := make([]float64, 0)
		for _, neighborSiteIdx := range cell.NeighborSiteIndices {
			neighborPosition := sites[neighborSiteIdx]
			// Use dot product to get angular distance on unit sphere
			dotProduct := cellPosition.Dot(neighborPosition)
			// Clamp to avoid numerical errors
			if dotProduct > 1.0 { dotProduct = 1.0 }
			if dotProduct < -1.0 { dotProduct = -1.0 }
			distance := math.Acos(dotProduct) // Angular distance in radians
			neighborDistances = append(neighborDistances, distance)
			totalDistances += distance
			distanceCount++
			
			fmt.Printf("  Neighbor %d: distance %.4f rad (%.1f deg)\n", 
				neighborSiteIdx, distance, distance*180.0/3.14159)
		}
		
		// Find max neighbor distance for this cell
		maxNeighborDist := 0.0
		for _, dist := range neighborDistances {
			if dist > maxNeighborDist {
				maxNeighborDist = dist
			}
		}
		
		// Check distances to some non-neighbors to see if any are closer than declared neighbors
		nonNeighborMap := make(map[int32]bool)
		for _, neighborSiteIdx := range cell.NeighborSiteIndices {
			nonNeighborMap[neighborSiteIdx] = true
		}
		nonNeighborMap[cell.SiteIndex] = true // Don't check self
		
		closerNonNeighbors := 0
		sampleCount := 0
		for j := 0; j < len(sites) && sampleCount < 20; j++ { // Sample 20 non-neighbors
			if !nonNeighborMap[int32(j)] {
				nonNeighborPosition := sites[j]
				dotProduct := cellPosition.Dot(nonNeighborPosition)
				if dotProduct > 1.0 { dotProduct = 1.0 }
				if dotProduct < -1.0 { dotProduct = -1.0 }
				distance := math.Acos(dotProduct)
				
				if distance < maxNeighborDist {
					closerNonNeighbors++
					if closerNonNeighbors <= 3 {
						fmt.Printf("  ⚠️  Non-neighbor %d is closer (%.4f rad) than furthest neighbor (%.4f rad)\n", 
							j, distance, maxNeighborDist)
					}
				}
				sampleCount++
			}
		}
		
		if closerNonNeighbors > 0 {
			geometricErrors++
			fmt.Printf("  Found %d non-neighbors closer than furthest declared neighbor\n", closerNonNeighbors)
		} else {
			fmt.Printf("  ✓ All neighbors are closer than sampled non-neighbors\n")
		}
		
		if maxNeighborDist > maxValidDistance {
			maxValidDistance = maxNeighborDist
		}
	}
	
	avgNeighborDistance := totalDistances / float64(distanceCount)
	fmt.Printf("\nGeometric analysis results:\n")
	fmt.Printf("  Cells with geometry errors: %d / 5\n", geometricErrors)
	fmt.Printf("  Average neighbor distance: %.4f rad (%.1f deg)\n", 
		avgNeighborDistance, avgNeighborDistance*180.0/3.14159)
	fmt.Printf("  Max neighbor distance observed: %.4f rad (%.1f deg)\n", 
		maxValidDistance, maxValidDistance*180.0/3.14159)
	
	// Expected distance for icosphere
	expectedDistance := 0.0
	if len(sites) > 12 {
		// Rough estimate: circumference / sqrt(num_vertices)
		expectedDistance = 2.0 * 3.14159 / (2.0 * math.Sqrt(float64(len(sites))))
		fmt.Printf("  Expected neighbor distance: ~%.4f rad (%.1f deg)\n", 
			expectedDistance, expectedDistance*180.0/3.14159)
	}
	
	if geometricErrors == 0 {
		fmt.Println("✓ Neighbor relationships appear geometrically correct")
	} else {
		fmt.Printf("⚠️  Found %d cells with questionable neighbor relationships\n", geometricErrors)
	}

	// Test 7: Clockwise ordering check
	fmt.Println("\n=== Test 7: Clockwise Ordering Check ===")
	fmt.Println("Checking if neighbors are sorted in clockwise order...")
	
	orderingErrors := 0
	
	for i := 0; i < 5 && i < len(voronoiCells); i++ {
		cell := voronoiCells[i]
		cellPosition := sites[cell.SiteIndex]
		
		fmt.Printf("\nCell %d clockwise ordering check:\n", i)
		
		if len(cell.NeighborSiteIndices) < 3 {
			fmt.Printf("  Skipping - only %d neighbors\n", len(cell.NeighborSiteIndices))
			continue
		}
		
		// Create a local coordinate system at the cell position
		// Similar to how it's done in voronoi.go:137-143
		upApprox := icosphere.Vector3D{X: 0, Y: 0, Z: 1}
		if math.Abs(cellPosition.Dot(upApprox)) > 0.99 {
			upApprox = icosphere.Vector3D{X: 0, Y: 1, Z: 0}
		}
		tangentX := cellPosition.Cross(upApprox).Normalize()
		tangentY := tangentX.Cross(cellPosition).Normalize()
		
		// Calculate angles for each neighbor
		neighborAngles := make([]float64, len(cell.NeighborSiteIndices))
		for j, neighborSiteIdx := range cell.NeighborSiteIndices {
			neighborPos := sites[neighborSiteIdx]
			localX := neighborPos.Dot(tangentX)
			localY := neighborPos.Dot(tangentY)
			angle := math.Atan2(localY, localX)
			neighborAngles[j] = angle
			
			fmt.Printf("  Neighbor %d (site %d): angle %.3f rad (%.1f deg)\n", 
				j, neighborSiteIdx, angle, angle*180.0/math.Pi)
		}
		
		// Check if angles are in clockwise order (decreasing)
		// Note: voronoi.go:170 sorts by angle1 > angle2 for clockwise
		clockwiseOrder := true
		for j := 0; j < len(neighborAngles)-1; j++ {
			currentAngle := neighborAngles[j]
			nextAngle := neighborAngles[j+1]
			
			// Handle angle wraparound (-π to π)
			angleDiff := currentAngle - nextAngle
			// Normalize to [-π, π]
			for angleDiff > math.Pi {
				angleDiff -= 2 * math.Pi
			}
			for angleDiff < -math.Pi {
				angleDiff += 2 * math.Pi
			}
			
			// For clockwise order, we expect current > next (positive difference)
			// But we need to handle wraparound carefully
			if angleDiff < 0 {
				// Check if this could be a valid wraparound case
				wraparoundDiff := currentAngle - nextAngle + 2*math.Pi
				if wraparoundDiff < 0 || wraparoundDiff > 2*math.Pi {
					clockwiseOrder = false
					fmt.Printf("  ❌ Order violation: neighbor %d (%.3f) -> neighbor %d (%.3f), diff=%.3f\n", 
						j, currentAngle, j+1, nextAngle, angleDiff)
					break
				}
			}
		}
		
		// Also check the wraparound from last to first
		if len(neighborAngles) > 2 {
			lastAngle := neighborAngles[len(neighborAngles)-1]
			firstAngle := neighborAngles[0]
			angleDiff := lastAngle - firstAngle
			
			// Normalize to [-π, π]
			for angleDiff > math.Pi {
				angleDiff -= 2 * math.Pi
			}
			for angleDiff < -math.Pi {
				angleDiff += 2 * math.Pi
			}
			
			if angleDiff < 0 {
				wraparoundDiff := lastAngle - firstAngle + 2*math.Pi
				if wraparoundDiff < 0 || wraparoundDiff > 2*math.Pi {
					clockwiseOrder = false
					fmt.Printf("  ❌ Wraparound violation: last (%.3f) -> first (%.3f), diff=%.3f\n", 
						lastAngle, firstAngle, angleDiff)
				}
			}
		}
		
		if clockwiseOrder {
			fmt.Printf("  ✓ Neighbors are in clockwise order\n")
		} else {
			fmt.Printf("  ❌ Neighbors are NOT in clockwise order\n")
			orderingErrors++
		}
		
		// Additional check: calculate total angular coverage
		// For neighbors in clockwise order, we want to measure the total coverage around the circle
		totalCoverage := 0.0
		if len(neighborAngles) > 2 {
			// Calculate the angular difference between each consecutive pair
			for j := 0; j < len(neighborAngles); j++ {
				currentAngle := neighborAngles[j]
				nextAngle := neighborAngles[(j+1)%len(neighborAngles)] // Wrap around for last->first
				
				// Calculate the clockwise angular distance from current to next
				angleDiff := currentAngle - nextAngle
				
				// Normalize to [0, 2π] for clockwise difference
				for angleDiff < 0 {
					angleDiff += 2 * math.Pi
				}
				for angleDiff >= 2 * math.Pi {
					angleDiff -= 2 * math.Pi
				}
				
				totalCoverage += angleDiff
			}
		}
		
		fmt.Printf("  Total angular coverage: %.3f rad (%.1f deg)\n", totalCoverage, totalCoverage*180.0/math.Pi)
		
		// For a well-distributed icosphere, neighbors should cover the full 360°
		expectedCoverage := 2 * math.Pi
		coverageError := math.Abs(totalCoverage - expectedCoverage)
		if coverageError > 0.1 { // Allow small numerical errors
			fmt.Printf("  ⚠️  Coverage error: %.1f deg (expected ~360.0 deg)\n", coverageError*180.0/math.Pi)
		} else {
			fmt.Printf("  ✓ Full angular coverage achieved\n")
		}
		
		// Also show the average spacing between neighbors
		if len(neighborAngles) > 0 {
			avgSpacing := totalCoverage / float64(len(neighborAngles))
			expectedSpacing := 2 * math.Pi / float64(len(neighborAngles))
			fmt.Printf("  Average neighbor spacing: %.1f deg (expected: %.1f deg)\n", 
				avgSpacing*180.0/math.Pi, expectedSpacing*180.0/math.Pi)
		}
	}
	
	fmt.Printf("\nClockwise ordering results:\n")
	fmt.Printf("  Cells with ordering errors: %d / 5\n", orderingErrors)
	
	if orderingErrors == 0 {
		fmt.Println("✓ Neighbor lists appear to be in correct clockwise order")
	} else {
		fmt.Printf("❌ Found %d cells with incorrect neighbor ordering\n", orderingErrors)
	}

	// Summary
	fmt.Println("\n=== Summary ===")
	if emptyCells == 0 && reciprocityErrors == 0 && invalidNeighbors == 0 && invalidFaces == 0 && geometricErrors == 0 && orderingErrors == 0 {
		fmt.Println("✅ Voronoi neighbor generation appears to be working correctly!")
	} else {
		fmt.Println("❌ Issues found with Voronoi neighbor generation")
		fmt.Printf("   - Empty neighbor lists: %d cells\n", emptyCells)
		fmt.Printf("   - Invalid neighbors: %d\n", invalidNeighbors)
		fmt.Printf("   - Reciprocity errors: %d\n", reciprocityErrors)
		fmt.Printf("   - Invalid faces: %d\n", invalidFaces)
		fmt.Printf("   - Geometric errors: %d\n", geometricErrors)
		fmt.Printf("   - Ordering errors: %d\n", orderingErrors)
	}
	
	// Expected values for reference
	fmt.Println("\n=== Expected Values (for reference) ===")
	expectedNeighbors := 6.0 // Most vertices in an icosphere have 6 neighbors (except 12 vertices with 5)
	fmt.Printf("Expected average neighbors: ~%.1f (most vertices have 6, twelve have 5)\n", expectedNeighbors)
	
	if len(sites) > 12 {
		expectedMin := 5
		expectedMax := 6
		fmt.Printf("Expected min neighbors: %d, max neighbors: %d\n", expectedMin, expectedMax)
		
		if minNeighbors != expectedMin || maxNeighbors != expectedMax {
			fmt.Printf("⚠️  Neighbor count outside expected range!\n")
		}
	}
}