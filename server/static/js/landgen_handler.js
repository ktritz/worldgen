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

/**
 * Initializes the Land Generation Tab.
 * @param {object} commonViewerAPI - The common API for viewer interactions.
 * @returns {object|null} Handler API for the tab or null if initialization fails.
 */
function initializeLandgenTab(commonViewerAPI) {
    console.log("Initializing Land Generation Tab specific JavaScript (with Tectonic Params)...");
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

    /**
     * Gathers data from the land generation form.
     * @returns {object|null} An object containing form parameters, or null if the form is not found.
     */
    function getLandGenerationFormData() {
        if (!landGenerationFormRef) return null;
        const formData = new FormData(landGenerationFormRef);
        const params = {};
        for (let [key, value] of formData.entries()) {
            const inputElement = landGenerationFormRef.elements[key];
            if (!inputElement) continue;

            // Convert to appropriate types
            if (inputElement.type === 'checkbox') {
                params[key] = inputElement.checked;
            } else if (inputElement.type === 'number') {
                // Specific integer fields
                if (key === 'landSeed' || key === 'noiseOctaves' || key === 'numPlates') {
                    params[key] = parseInt(value, 10);
                } else { // Other numbers are floats
                    params[key] = parseFloat(value);
                }
            } else {
                params[key] = value;
            }
        }
        return params;
    }

    /**
     * Handles the land generation process by sending a request to the backend.
     */
    async function handleLandGeneration() {
        if (!commonViewerAPIInstance) {
            console.error("LandgenHandler: commonViewerAPIInstance is not available.");
            return;
        }

        const activeMeshData = commonViewerAPIInstance.getActiveMeshData();
        if (!activeMeshData || !activeMeshData.icosphereData || !activeMeshData.voronoiData ||
            !activeMeshData.icosphereData.vertices || !activeMeshData.voronoiData.cells ||
            activeMeshData.subdivisionLevel === undefined || activeMeshData.subdivisionLevel < 0) {
            commonViewerAPIInstance.showStatus("Error: No complete active Icosphere/Voronoi mesh data with subdivision level. Generate/load one first.", "error");
            updateBaseMeshInfoDisplay(null, null, -1); // Update UI to reflect missing base mesh
            if (mainGenerateBtnRef) mainGenerateBtnRef.disabled = true;
            return;
        }

        // Update the base mesh info display with current active mesh data
        updateBaseMeshInfoDisplay(activeMeshData.icosphereData, activeMeshData.voronoiData, activeMeshData.subdivisionLevel);

        const landParams = getLandGenerationFormData();
        if (!landParams) {
            commonViewerAPIInstance.showStatus("Error: Could not read land generation parameters.", "error");
            return;
        }

        // Construct the payload for the backend
        const requestPayload = {
            // General land parameters
            landSeed: landParams.landSeed, // This will be used as GlobalSeed

            // Tectonic Plate Parameters
            numPlates: landParams.numPlates,
            baseSpeed: landParams.baseSpeed,
            speedVariationFactor: landParams.speedVariationFactor,
            targetContinentalProportion: landParams.targetContinentalProportion,
            numInitialContinentalSeeds: landParams.numInitialContinentalSeeds,

            // Elevation Parameters
            noiseScale: landParams.noiseScale,
            noiseOctaves: landParams.noiseOctaves,
            noisePersistence: landParams.noisePersistence,
            noiseLacunarity: landParams.noiseLacunarity,
            elevationMultiplier: landParams.elevationMultiplier,
            
            // Tectonic Boundary Effects
            characteristicFalloffDistance: landParams.characteristicFalloffDistance,
            maxBoundaryEffectDistance: landParams.maxBoundaryEffectDistance,
            convergentBoundaryStrength: landParams.convergentBoundaryStrength,
            divergentBoundaryStrength: landParams.divergentBoundaryStrength,

            // Output and Base Mesh Data
            landOutputName: landParams.landOutputName,
            baseIcosphereData: activeMeshData.icosphereData,
            baseVoronoiData: activeMeshData.voronoiData,
            icosphereSubdivisions: activeMeshData.subdivisionLevel,
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

                        if (minElev === Infinity || maxElev === -Infinity) {
                            minElev = 0; maxElev = 0.1;
                            console.warn("LandgenHandler: minElev or maxElev was not updated, defaulting. Check elevation data.");
                        } else if (minElev === maxElev) {
                            maxElev = minElev + Math.abs(minElev * 0.1) + 0.01;
                            if (minElev === maxElev) maxElev = minElev + 0.1;
                        }
                        console.log(`LandgenHandler: Calculated minElev: ${minElev}, maxElev: ${maxElev}`);

                        const numIcosphereSites = activeMeshData.icosphereData.vertices.length / 3;
                        let texWidth = Math.ceil(Math.sqrt(numIcosphereSites));
                        const MAX_TEXTURE_DIM = 4096;
                        texWidth = Math.min(texWidth, MAX_TEXTURE_DIM);
                        let texHeight = Math.ceil(numIcosphereSites / texWidth);
                        texHeight = Math.min(texHeight, MAX_TEXTURE_DIM);
                        if (texWidth * texHeight < numIcosphereSites) {
                            texHeight = Math.ceil(numIcosphereSites / texWidth);
                        }
                        console.log(`LandgenHandler: numIcosphereSites: ${numIcosphereSites}, TexWidth: ${texWidth}, TexHeight: ${texHeight}`);

                        if (numIcosphereSites > 0 && texWidth > 0 && texHeight > 0) {
                            commonViewerAPIInstance.updateVoronoiElevationVisuals(
                                cellElevationsMap,
                                minElev,
                                maxElev,
                                texWidth,
                                texHeight,
                                activeMeshData.subdivisionLevel
                            );
                        } else {
                            console.error("LandgenHandler: Invalid dimensions for elevation texture.", { numIcosphereSites, texWidth, texHeight });
                            commonViewerAPIInstance.showStatus("Error preparing elevation visualization: invalid texture dimensions.", "error");
                        }
                    } else {
                        commonViewerAPIInstance.showStatus("No elevation data found in response to visualize.", "info");
                        if (commonViewerAPIInstance.updateVoronoiElevationVisuals) {
                            commonViewerAPIInstance.updateVoronoiElevationVisuals(new Map(), 0, 0.1, 1, 1, -1);
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
        /**
         * Updates the state and text of the main generate button based on current conditions.
         */
        updateGenerateButton: () => {
            if (!mainGenerateBtnRef) mainGenerateBtnRef = document.getElementById('mainGenerateBtn');

            const activeMeshData = commonViewerAPIInstance ? commonViewerAPIInstance.getActiveMeshData() : null;
            const canGenerateLand = updateBaseMeshInfoDisplay(
                activeMeshData ? activeMeshData.icosphereData : null,
                activeMeshData ? activeMeshData.voronoiData : null,
                activeMeshData ? activeMeshData.subdivisionLevel : -1
            );

            if (mainGenerateBtnRef) {
                if (canGenerateLand) {
                    mainGenerateBtnRef.textContent = 'Generate Land & Tectonics'; // Updated button text
                    mainGenerateBtnRef.classList.remove('btn-secondary');
                    mainGenerateBtnRef.classList.add('btn-primary');
                    mainGenerateBtnRef.disabled = false;
                } else {
                    mainGenerateBtnRef.textContent = 'Generate Land & Tectonics';
                    mainGenerateBtnRef.classList.remove('btn-primary');
                    mainGenerateBtnRef.classList.add('btn-secondary');
                    mainGenerateBtnRef.disabled = true;
                }
            }
        },
        /**
         * Cleans up resources specific to this tab handler.
         */
        dispose: () => {
            console.log("Disposing Land Generation Tab specific resources...");
            // Nullify references to DOM elements and API instances to prevent memory leaks
            commonViewerAPIInstance = null;
            landGenerationFormRef = null;
            mainGenerateBtnRef = null;
            baseMeshInfoDivRef = null;
            // Any other specific event listeners or objects created by this tab should be cleaned up here
        }
    };

    // Initial update of the generate button state when the tab is loaded
    if (typeof handlerAPI.updateGenerateButton === 'function') {
        handlerAPI.updateGenerateButton();
    }

    console.log("Land Generation Tab specific JavaScript Initialized (with Tectonic Params).");
    return handlerAPI;
}

// Expose the initializer to be called by the tab loader
window.initializeLandgenTab = initializeLandgenTab;
