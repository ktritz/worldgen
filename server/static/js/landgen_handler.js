// static/js/landgen_handler.js
import * as THREE from 'three';

let commonViewerAPIInstance = null;
let landGenerationFormRef = null;
let mainGenerateBtnRef = null;
let baseMeshInfoDivRef = null;

/**
 * Updates the display area with information about the base mesh.
 * @param {object|null} icosphereData Raw icosphere data from shell {vertices, faces}.
 * @param {object|null} voronoiData Raw voronoi data from shell {vertices, cells}.
 * @param {number} subdivisionLevel The subdivision level of the base mesh.
 * @returns {boolean} True if valid base mesh data is present, false otherwise.
 */
function updateBaseMeshInfoDisplay(icosphereData, voronoiData, subdivisionLevel) {
    if (!baseMeshInfoDivRef) {
        baseMeshInfoDivRef = document.getElementById('baseMeshInfo');
        if (!baseMeshInfoDivRef) return false;
    }

    const icosphereReady = icosphereData && icosphereData.vertices && icosphereData.faces;
    const voronoiReady = voronoiData && voronoiData.vertices && voronoiData.cells;

    if (icosphereReady && voronoiReady && subdivisionLevel !== undefined && subdivisionLevel >= 0) {
        const numSites = icosphereData.vertices.length / 3;
        const numVoronoiCells = voronoiData.cells.length;
        const numVoronoiVertices = voronoiData.vertices.length / 3;
        baseMeshInfoDivRef.innerHTML = `
            <p class="font-semibold">Using active base mesh (Subdivision: ${subdivisionLevel}):</p>
            <ul class="list-disc list-inside ml-2">
                <li>Icosphere Sites (Voronoi Generators): ${numSites}</li>
                <li>Voronoi Cells: ${numVoronoiCells}</li>
                <li>Voronoi Vertices: ${numVoronoiVertices}</li>
            </ul>
        `;
        return true;
    } else {
        baseMeshInfoDivRef.innerHTML = `
            <p class="text-red-500 font-semibold">No active Icosphere & Voronoi mesh found, or subdivision level is unknown.</p>
            <p>Please go to the 'Icosphere & Voronoi' tab to generate or load a base mesh first.</p>
        `;
        return false;
    }
}


function initializeLandgenTab(commonViewerAPI) {
    console.log("Initializing Land Generation Tab specific JavaScript (v1.4 - 2D Texture & Subdiv Logic)...");
    commonViewerAPIInstance = commonViewerAPI;

    landGenerationFormRef = document.getElementById('landGenerationForm');
    if (!landGenerationFormRef) {
        console.error("LandgenHandler: Land generation form (id='landGenerationForm') not found!");
        if (commonViewerAPIInstance && typeof commonViewerAPIInstance.showStatus === 'function') {
            commonViewerAPIInstance.showStatus("Error: Land Generation parameter form not found.", "error");
        }
        return null;
    }

    baseMeshInfoDivRef = document.getElementById('baseMeshInfo');
    // mainGenerateBtnRef is acquired by updateGenerateButton when tab is active

    function getLandGenerationFormData() {
        if (!landGenerationFormRef) return null;
        const formData = new FormData(landGenerationFormRef);
        const params = {};
        for (let [key, value] of formData.entries()) {
            const inputElement = landGenerationFormRef.elements[key];
            if (!inputElement) continue;

            if (inputElement.type === 'checkbox') {
                params[key] = inputElement.checked;
            } else if (inputElement.type === 'number') {
                if (key === 'landSeed' || key === 'noiseOctaves') {
                    params[key] = parseInt(value, 10);
                } else {
                    params[key] = parseFloat(value);
                }
            } else {
                params[key] = value;
            }
        }
        return params;
    }

    async function handleLandGeneration() {
        if (!commonViewerAPIInstance) {
            console.error("LandgenHandler: commonViewerAPIInstance is not available.");
            return;
        }

        const activeMeshData = commonViewerAPIInstance.getActiveMeshData();
        if (!activeMeshData || !activeMeshData.icosphereData || !activeMeshData.voronoiData ||
            !activeMeshData.icosphereData.vertices || !activeMeshData.voronoiData.cells ||
            activeMeshData.subdivisionLevel === undefined || activeMeshData.subdivisionLevel < 0) { // Check for valid subdivision level
            commonViewerAPIInstance.showStatus("Error: No complete active Icosphere/Voronoi mesh data with subdivision level. Generate/load one first.", "error");
            updateBaseMeshInfoDisplay(null, null, -1);
            if (mainGenerateBtnRef) mainGenerateBtnRef.disabled = true;
            return;
        }

        updateBaseMeshInfoDisplay(activeMeshData.icosphereData, activeMeshData.voronoiData, activeMeshData.subdivisionLevel);

        const landParams = getLandGenerationFormData();
        if (!landParams) {
            commonViewerAPIInstance.showStatus("Error: Could not read land generation parameters.", "error");
            return;
        }

        const requestPayload = {
            ...landParams,
            baseIcosphereData: activeMeshData.icosphereData,
            baseVoronoiData: activeMeshData.voronoiData,
            // No need to send subdivision separately if backend doesn't strictly need it for landgen logic itself,
            // but it's good for context if landgen.LandGenerationParams has it.
            // For now, assuming backend's landgen.LandGenerationParams still has IcosphereSubdivisions for reference.
            icosphereSubdivisions: activeMeshData.subdivisionLevel
        };

        commonViewerAPIInstance.showStatus('Requesting land generation... Please wait.', 'info');

        try {
            const response = await fetch('/api/generate_land', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(requestPayload),
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`HTTP error ${response.status}: ${errorText}`);
            }

            const result = await response.json();
            console.log("Land Generation Result from server:", result);

            if (result.status === 'success') {
                commonViewerAPIInstance.showStatus(`Land generation successful: ${result.message}`, 'success');

                if (result.elevationData && result.elevationData.cellElevations) {
                    const cellElevationsMap = new Map(Object.entries(result.elevationData.cellElevations).map(([k, v]) => [parseInt(k), parseFloat(v)]));
                    console.log(`LandgenHandler: Parsed cellElevationsMap.size: ${cellElevationsMap.size}`);

                    if (cellElevationsMap.size > 0) {
                        let minElev = Infinity;
                        let maxElev = -Infinity;

                        cellElevationsMap.forEach((elevation) => {
                            if (elevation < minElev) minElev = elevation;
                            if (elevation > maxElev) maxElev = elevation;
                        });

                        if (minElev === maxElev) {
                            maxElev = minElev + 0.1;
                            if (minElev === 0 && maxElev === 0.1) { /* Correct */ }
                            else if (minElev === 0) { maxElev = 0.1; }
                            else if (maxElev === minElev) { maxElev = minElev + Math.abs(minElev * 0.1) + 0.01; }
                            if (minElev === (maxElev - 0.1) && minElev < 0) {
                                minElev = maxElev - 0.2;
                            }
                        }
                        console.log(`LandgenHandler: Calculated minElev: ${minElev}, maxElev: ${maxElev}`);

                        const totalCellCount = activeMeshData.icosphereData.vertices.length / 3; // Number of sites

                        // --- MODIFIED: Calculate 2D texture dimensions ---
                        let texWidth = Math.ceil(Math.sqrt(totalCellCount));
                        const MAX_TEXTURE_DIM = 4096;
                        texWidth = Math.min(texWidth, MAX_TEXTURE_DIM);
                        let texHeight = Math.ceil(totalCellCount / texWidth);
                        texHeight = Math.min(texHeight, MAX_TEXTURE_DIM);

                        if (texWidth * texHeight < totalCellCount) {
                            texHeight = Math.ceil(totalCellCount / texWidth);
                        }
                        console.log(`LandgenHandler: totalCellCount: ${totalCellCount}, Calculated TexWidth: ${texWidth}, TexHeight: ${texHeight}`);
                        // --- END MODIFICATION ---

                        if (totalCellCount > 0 && texWidth > 0 && texHeight > 0) {
                            // Pass the subdivision level of the base mesh for which this elevation data is valid
                            commonViewerAPIInstance.updateVoronoiElevationVisuals(
                                cellElevationsMap,
                                minElev,
                                maxElev,
                                texWidth,
                                texHeight,
                                activeMeshData.subdivisionLevel // Pass the base mesh's subdivision level
                            );
                        } else {
                            console.error("LandgenHandler: Invalid dimensions for elevation texture.", { totalCellCount, texWidth, texHeight });
                            commonViewerAPIInstance.showStatus("Error preparing elevation visualization: invalid texture dimensions.", "error");
                        }
                    } else {
                        commonViewerAPIInstance.showStatus("No elevation data found in response to visualize.", "info");
                        if (commonViewerAPIInstance.updateVoronoiElevationVisuals) {
                            commonViewerAPIInstance.updateVoronoiElevationVisuals(new Map(), 0, 0, 1, 1, -1); // Reset with dummy, invalid subdivision
                        }
                    }
                } else if (result.heightmapUrl) {
                    commonViewerAPIInstance.showStatus(`Auxiliary heightmap image at ${result.heightmapUrl}. Per-cell elevation visualization not available.`, "info");
                } else {
                    commonViewerAPIInstance.showStatus("Land data generated, but no elevation data provided for visualization.", "info");
                }

            } else {
                commonViewerAPIInstance.showStatus(`Land generation failed: ${result.message || 'Unknown error'}`, 'error');
            }

        } catch (error) {
            console.error('Error during land generation:', error);
            commonViewerAPIInstance.showStatus(`Client-side error during land generation: ${error.message}`, 'error');
        }
    }

    const handlerAPI = {
        generate: handleLandGeneration,
        updateGenerateButton: () => {
            if (!mainGenerateBtnRef) mainGenerateBtnRef = document.getElementById('mainGenerateBtn');

            const activeMeshData = commonViewerAPIInstance ? commonViewerAPIInstance.getActiveMeshData() : null;
            const canGenerateLand = updateBaseMeshInfoDisplay(
                activeMeshData ? activeMeshData.icosphereData : null,
                activeMeshData ? activeMeshData.voronoiData : null,
                activeMeshData ? activeMeshData.subdivisionLevel : -1 // Pass subdivision to display
            );

            if (mainGenerateBtnRef) {
                if (canGenerateLand) {
                    mainGenerateBtnRef.textContent = 'Generate Land';
                    mainGenerateBtnRef.classList.remove('btn-secondary');
                    mainGenerateBtnRef.classList.add('btn-primary');
                    mainGenerateBtnRef.disabled = false;
                } else {
                    mainGenerateBtnRef.textContent = 'Generate Land';
                    mainGenerateBtnRef.classList.remove('btn-primary');
                    mainGenerateBtnRef.classList.add('btn-secondary');
                    mainGenerateBtnRef.disabled = true;
                }
            }
        },
        dispose: () => {
            console.log("Disposing Land Generation Tab specific resources...");
            commonViewerAPIInstance = null;
            landGenerationFormRef = null;
            mainGenerateBtnRef = null;
            baseMeshInfoDivRef = null;
        }
    };

    if (typeof handlerAPI.updateGenerateButton === 'function') {
        handlerAPI.updateGenerateButton();
    }

    console.log("Land Generation Tab specific JavaScript Initialized.");
    return handlerAPI;
}

window.initializeLandgenTab = initializeLandgenTab;
