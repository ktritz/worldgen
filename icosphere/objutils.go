package icosphere

import (
	"bufio"
	"fmt"
	"os"
)

// SaveOBJTriangulated writes a mesh with triangular faces to an OBJ file.
// Used for saving the base icosphere/Delaunay mesh.
// This function is exported.
func SaveOBJTriangulated(filename string, vertices []Vector3D, faces []Triangle, comment string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	if _, err := writer.WriteString(fmt.Sprintf("# %s\n", comment)); err != nil {
		return err
	}
	if _, err := writer.WriteString(fmt.Sprintf("# Vertices: %d\n", len(vertices))); err != nil {
		return err
	}
	if _, err := writer.WriteString(fmt.Sprintf("# Faces: %d\n", len(faces))); err != nil {
		return err
	}

	for _, v := range vertices {
		if _, err := writer.WriteString(fmt.Sprintf("v %.6f %.6f %.6f\n", v.X, v.Y, v.Z)); err != nil {
			return err
		}
	}

	for _, f := range faces {
		if _, err := writer.WriteString(fmt.Sprintf("f %d %d %d\n", f.V1+1, f.V2+1, f.V3+1)); err != nil {
			return err
		}
	}

	return writer.Flush()
}

// SaveVoronoiOBJ_NGon saves a Voronoi diagram with N-gon faces.
// Each Voronoi cell is written as a separate group 'g'.
// This function is exported.
func SaveVoronoiOBJ_NGon(filename string, voronoiVertices []Vector3D, cells []VoronoiCell, comment string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	if _, err := writer.WriteString(fmt.Sprintf("# %s\n", comment)); err != nil {
		return err
	}
	if _, err := writer.WriteString(fmt.Sprintf("# Voronoi Vertices: %d\n", len(voronoiVertices))); err != nil {
		return err
	}
	if _, err := writer.WriteString(fmt.Sprintf("# N-gon Voronoi Cells: %d\n", len(cells))); err != nil {
		return err
	}

	for _, v := range voronoiVertices {
		if _, err := writer.WriteString(fmt.Sprintf("v %.6f %.6f %.6f\n", v.X, v.Y, v.Z)); err != nil {
			return err
		}
	}

	for _, cell := range cells {
		if len(cell.VertexIndices) < 3 {
			continue
		}
		if _, err := writer.WriteString(fmt.Sprintf("g voronoi_cell_%d\n", cell.SiteIndex)); err != nil {
			return err
		}

		line := "f"
		for _, idx := range cell.VertexIndices {
			line += fmt.Sprintf(" %d", idx+1)
		}
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

// SaveVoronoiOBJTriangulated saves a Voronoi diagram by triangulating its polygonal faces.
// Each original Voronoi cell's triangles are written under a separate group 'g'.
// This function is exported.
func SaveVoronoiOBJTriangulated(filename string, voronoiVertices []Vector3D, cells []VoronoiCell, comment string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	if _, err := writer.WriteString(fmt.Sprintf("# %s\n", comment)); err != nil {
		return err
	}
	if _, err := writer.WriteString(fmt.Sprintf("# Voronoi Vertices: %d\n", len(voronoiVertices))); err != nil {
		return err
	}

	numTriangles := 0
	for _, cell := range cells {
		if len(cell.VertexIndices) >= 3 {
			numTriangles += len(cell.VertexIndices) - 2
		}
	}
	if _, err := writer.WriteString(fmt.Sprintf("# Triangulated Voronoi Faces: %d\n", numTriangles)); err != nil {
		return err
	}

	for _, v := range voronoiVertices {
		if _, err := writer.WriteString(fmt.Sprintf("v %.6f %.6f %.6f\n", v.X, v.Y, v.Z)); err != nil {
			return err
		}
	}

	for _, cell := range cells {
		polyIndices := cell.VertexIndices
		if len(polyIndices) < 3 {
			continue
		}
		if _, err := writer.WriteString(fmt.Sprintf("g voronoi_cell_%d\n", cell.SiteIndex)); err != nil {
			return err
		}

		v0_obj := polyIndices[0] + 1
		for i := 1; i < len(polyIndices)-1; i++ {
			v1_obj := polyIndices[i] + 1
			v2_obj := polyIndices[i+1] + 1
			if _, err := writer.WriteString(fmt.Sprintf("f %d %d %d\n", v0_obj, v1_obj, v2_obj)); err != nil {
				return err
			}
		}
	}
	return writer.Flush()
}
