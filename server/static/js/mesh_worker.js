self.onmessage = function (e) {
    const { type, icosphereData, voronoiData, voronoiEnabled } = e.data;

    if (type === 'generate') {
        let processedIcosphereData = null;
        if (icosphereData) {
            processedIcosphereData = processIcosphereRawData(icosphereData);
        }

        let processedVoronoiData = null;
        if (voronoiEnabled && voronoiData) {
            processedVoronoiData = processVoronoiRawData(voronoiData);
        }

        self.postMessage({
            status: 'complete',
            icosphere: processedIcosphereData,
            voronoi: processedVoronoiData
        });
    }
};

function processIcosphereRawData(meshData) {
    // The raw data is already in a good format (flat arrays for vertices and faces)
    // We just ensure they are the correct types if necessary, or pass through
    // For simplicity, assuming data is { vertices: number[], faces: number[] }
    if (!meshData || !meshData.vertices || !meshData.faces) {
        console.error("Worker: Invalid icosphere data for processing.", meshData);
        return null;
    }
    if (meshData.faces.length === 0) {
        console.error("Worker: meshData.faces is empty for icosphere.");
        return {
            vertices: new Float32Array(meshData.vertices),
            faces: new Uint32Array(meshData.faces) // Or Int32Array
        };
    }
    if (!Array.isArray(meshData.faces) || !meshData.faces.every(num => typeof num === 'number')) {
        console.error("Worker: meshData.faces is not a valid array of numbers for icosphere.");
        return null;
    }

    return {
        vertices: new Float32Array(meshData.vertices),
        faces: new Uint32Array(meshData.faces) // Or Int32Array, ensure consistency with BufferGeometry.setIndex
    };
}

function processVoronoiRawData(voronoiRawData) {
    if (!voronoiRawData || !voronoiRawData.vertices || !voronoiRawData.cells) {
        console.error("Worker: Invalid Voronoi raw data for processing.", voronoiRawData);
        return null;
    }

    const allVoronoiVerticesArray = new Float32Array(voronoiRawData.vertices);
    const cells = voronoiRawData.cells;

    // --- Process Solid Mesh Data ---
    let totalTrianglesSolid = 0;
    cells.forEach(cell => {
        if (cell.vertexIndices && cell.vertexIndices.length >= 3) {
            totalTrianglesSolid += cell.vertexIndices.length - 2;
        }
    });

    let solidData = null;
    if (totalTrianglesSolid > 0) {
        const finalSolidPositions = new Float32Array(totalTrianglesSolid * 9); // 3 vertices per triangle, 3 coords per vertex
        const finalSolidCellIds = new Float32Array(totalTrianglesSolid * 3);   // 3 vertices per triangle, 1 cellId per vertex
        const finalSolidBarycentrics = new Float32Array(totalTrianglesSolid * 9); // 3 vertices per triangle, 3 barycentric coords per vertex
        let currentSolidVertexIndex = 0;

        cells.forEach((cell, cellIdx) => {
            const polyIndices = cell.vertexIndices;
            if (!polyIndices || polyIndices.length < 3) return;

            const v0Idx = polyIndices[0];
            for (let i = 1; i < polyIndices.length - 1; i++) {
                const v1Idx = polyIndices[i];
                const v2Idx = polyIndices[i + 1];

                [v0Idx, v1Idx, v2Idx].forEach((pvIdx, triVertIdx) => {
                    if ((pvIdx * 3 + 2) >= allVoronoiVerticesArray.length) {
                        console.error(`Worker Voronoi Solid: Vertex index out of bounds: ${pvIdx}`);
                        // Potentially skip this vertex or triangle
                        return;
                    }
                    // Positions
                    finalSolidPositions[currentSolidVertexIndex * 3 + 0] = allVoronoiVerticesArray[pvIdx * 3 + 0];
                    finalSolidPositions[currentSolidVertexIndex * 3 + 1] = allVoronoiVerticesArray[pvIdx * 3 + 1];
                    finalSolidPositions[currentSolidVertexIndex * 3 + 2] = allVoronoiVerticesArray[pvIdx * 3 + 2];

                    // Cell IDs
                    finalSolidCellIds[currentSolidVertexIndex] = cellIdx;

                    // Barycentric coordinates
                    finalSolidBarycentrics[currentSolidVertexIndex * 3 + 0] = (triVertIdx === 0) ? 1 : 0;
                    finalSolidBarycentrics[currentSolidVertexIndex * 3 + 1] = (triVertIdx === 1) ? 1 : 0;
                    finalSolidBarycentrics[currentSolidVertexIndex * 3 + 2] = (triVertIdx === 2) ? 1 : 0;

                    currentSolidVertexIndex++;
                });
            }
        });
        solidData = {
            positions: finalSolidPositions.slice(0, currentSolidVertexIndex * 3),
            cellIds: finalSolidCellIds.slice(0, currentSolidVertexIndex),
            barycentrics: finalSolidBarycentrics.slice(0, currentSolidVertexIndex * 3)
        };
    } else {
        console.warn("Worker: No triangles to create for Voronoi solid mesh.");
    }

    // --- Process Outline Mesh Data ---
    const outlinePointsList = [];
    cells.forEach(cell => {
        if (!cell.vertexIndices || cell.vertexIndices.length < 2) return;
        const polyIndices = cell.vertexIndices;
        for (let i = 0; i < polyIndices.length; i++) {
            const idx1 = polyIndices[i];
            const idx2 = polyIndices[(i + 1) % polyIndices.length];

            if ((idx1 * 3 + 2) < allVoronoiVerticesArray.length && (idx2 * 3 + 2) < allVoronoiVerticesArray.length) {
                outlinePointsList.push(allVoronoiVerticesArray[idx1 * 3 + 0]);
                outlinePointsList.push(allVoronoiVerticesArray[idx1 * 3 + 1]);
                outlinePointsList.push(allVoronoiVerticesArray[idx1 * 3 + 2]);
                outlinePointsList.push(allVoronoiVerticesArray[idx2 * 3 + 0]);
                outlinePointsList.push(allVoronoiVerticesArray[idx2 * 3 + 1]);
                outlinePointsList.push(allVoronoiVerticesArray[idx2 * 3 + 2]);
            } else {
                console.warn(`Worker Voronoi Outline: Vertex index out of bounds for cell edge. idx1: ${idx1}, idx2: ${idx2}`);
            }
        }
    });

    let outlineData = null;
    if (outlinePointsList.length > 0) {
        outlineData = {
            points: new Float32Array(outlinePointsList)
        };
    } else {
        console.warn("Worker: No outline points generated for Voronoi.");
    }

    return {
        solid: solidData,
        outline: outlineData
    };
} 