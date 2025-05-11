package icosphere

import (
	"fmt"
	"runtime"
	"sync"
	// "math" // Not directly needed here if Vector3D methods are used
)

// RelaxMeshParameters holds all parameters for the relaxation process.
// This type is exported.
type RelaxMeshParameters struct {
	FixedInitial          bool
	K                     float64
	Damping               float64
	DtInitial             float64
	MaxIterations         int
	Tolerance             float64
	AdaptiveDt            bool
	DtMin                 float64
	DtMax                 float64
	DtIncreaseFactor      float64
	DtDecreaseFactor      float64
	MovementThresholdLow  float64
	MovementThresholdHigh float64
}

// RelaxMesh performs damped spring relaxation on the mesh.
// This function is exported.
func RelaxMesh(vertices []Vector3D, faces []Triangle, params RelaxMeshParameters) {
	numVertices := len(vertices)
	if numVertices == 0 {
		return
	}

	fixedCount := 0
	if params.FixedInitial {
		fixedCount = 12
		if numVertices < fixedCount {
			fixedCount = numVertices
		}
	}

	adj := make([][]int, numVertices)
	neighborMaps := make([]map[int]bool, numVertices)
	for i := range neighborMaps {
		neighborMaps[i] = make(map[int]bool)
	}
	var totalEdgeLength float64
	var edgeCount int
	uniqueEdges := make(map[EdgeKey]bool)

	for _, face := range faces {
		triVerts := [3]int{face.V1, face.V2, face.V3}
		for i := 0; i < 3; i++ {
			u := triVerts[i]
			v := triVerts[(i+1)%3]

			neighborMaps[u][v] = true
			neighborMaps[v][u] = true

			// Use the package-internal newEdgeKey
			edgeKey := newEdgeKey(u, v)
			if !uniqueEdges[edgeKey] {
				dist := vertices[u].Subtract(vertices[v]).Length()
				totalEdgeLength += dist
				edgeCount++
				uniqueEdges[edgeKey] = true
			}
		}
	}
	for i := 0; i < numVertices; i++ {
		for neighborIdx := range neighborMaps[i] {
			adj[i] = append(adj[i], neighborIdx)
		}
	}

	targetLength := 0.0
	if edgeCount > 0 {
		targetLength = totalEdgeLength / float64(edgeCount)
	} else if numVertices > 1 {
		fmt.Println("Warning: Target length was 0 from unique edges, falling back to an estimate.")
		foundEdge := false
		for i := 0; i < numVertices && !foundEdge; i++ {
			if len(adj[i]) > 0 {
				targetLength = vertices[i].Subtract(vertices[adj[i][0]]).Length()
				if targetLength > 1e-9 {
					foundEdge = true
				}
			}
		}
		if !foundEdge && numVertices > 1 {
			targetLength = vertices[0].Subtract(vertices[1]).Length()
		}
	}
	// This print statement might be better handled by the calling code (CLI or server)
	// if the library is to be truly general-purpose. For now, keeping it.
	fmt.Printf("  Relaxation: Target edge length (L0): %.6f\n", targetLength)

	velocities := make([]Vector3D, numVertices)
	forces := make([]Vector3D, numVertices)
	toleranceSq := params.Tolerance * params.Tolerance

	numWorkers := runtime.NumCPU()
	if numMovableVertices := numVertices - fixedCount; numMovableVertices > 0 {
		if numWorkers > numMovableVertices {
			numWorkers = numMovableVertices
		}
	} else {
		numWorkers = 1
	}
	if numWorkers <= 0 {
		numWorkers = 1
	}
	// fmt.Printf("  Relaxation: Using %d worker goroutines.\n", numWorkers) // Also potentially for calling code
	localMaxMovementsSq := make([]float64, numWorkers)
	currentDt := params.DtInitial

	for iter := 0; iter < params.MaxIterations; iter++ {
		var wg sync.WaitGroup

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				numMovable := numVertices - fixedCount
				if numMovable <= 0 {
					return
				}

				startIndex := fixedCount + workerID*(numMovable/numWorkers)
				endIndex := fixedCount + (workerID+1)*(numMovable/numWorkers)
				if workerID == numWorkers-1 {
					endIndex = numVertices
				}

				if startIndex >= endIndex {
					return
				}

				for i := startIndex; i < endIndex; i++ {
					p1 := vertices[i]
					totalForceOnI := Vector3D{0, 0, 0}
					for _, j := range adj[i] {
						p2 := vertices[j]
						vec := p2.Subtract(p1)
						dist := vec.Length()
						if dist > 1e-9 {
							direction := vec.Scale(1.0 / dist)
							displacement := dist - targetLength
							springForceMagnitude := params.K * displacement
							springForceVector := direction.Scale(springForceMagnitude)
							totalForceOnI = totalForceOnI.Add(springForceVector)
						}
					}
					forces[i] = totalForceOnI
				}
			}(w)
		}
		wg.Wait()

		for i := range localMaxMovementsSq {
			localMaxMovementsSq[i] = 0.0
		}

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				maxMovSqLocal := 0.0
				numMovable := numVertices - fixedCount
				if numMovable <= 0 {
					return
				}

				startIndex := fixedCount + workerID*(numMovable/numWorkers)
				endIndex := fixedCount + (workerID+1)*(numMovable/numWorkers)
				if workerID == numWorkers-1 {
					endIndex = numVertices
				}

				if startIndex >= endIndex {
					return
				}

				for i := startIndex; i < endIndex; i++ {
					dampingForce := velocities[i].Scale(-params.Damping)
					netForce := forces[i].Add(dampingForce)

					newVelocity := velocities[i].Add(netForce.Scale(currentDt))

					deltaP := newVelocity.Scale(currentDt)
					newPosition := vertices[i].Add(deltaP)

					velocities[i] = newVelocity
					vertices[i] = newPosition.Normalize()

					movementSq := deltaP.LengthSq()
					if movementSq > maxMovSqLocal {
						maxMovSqLocal = movementSq
					}
				}
				localMaxMovementsSq[workerID] = maxMovSqLocal
			}(w)
		}
		wg.Wait()

		maxMovementSqThisIteration := 0.0
		for _, localMax := range localMaxMovementsSq {
			if localMax > maxMovementSqThisIteration {
				maxMovementSqThisIteration = localMax
			}
		}

		// if (iter+1)%10 == 0 || iter == params.MaxIterations-1 {
		// 	fmt.Printf("  Relax Iter %d/%d, MaxMove^2: %.2e, dt: %.2e\n", iter+1, params.MaxIterations, maxMovementSqThisIteration, currentDt)
		// }

		if maxMovementSqThisIteration < toleranceSq && iter > 0 {
			// fmt.Printf("  Relaxation converged after %d iterations.\n", iter+1)
			break
		}
		// if iter == params.MaxIterations-1 {
		// 	fmt.Printf("  Relaxation reached max %d iterations.\n", params.MaxIterations)
		// }

		if params.AdaptiveDt {
			if maxMovementSqThisIteration < params.MovementThresholdLow && currentDt < params.DtMax {
				currentDt *= params.DtIncreaseFactor
				if currentDt > params.DtMax {
					currentDt = params.DtMax
				}
			} else if maxMovementSqThisIteration > params.MovementThresholdHigh && currentDt > params.DtMin {
				currentDt *= params.DtDecreaseFactor
				if currentDt < params.DtMin {
					currentDt = params.DtMin
				}
			}
		}
	}
}
