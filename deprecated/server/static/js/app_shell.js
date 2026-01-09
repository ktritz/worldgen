// static/js/app_shell.js (New leaner version)
import * as THREE from 'three'; // CORRECTED: Direct import of THREE
import { initializeViewer } from './three_viewer.js';
import { initializeShellLayout } from './shell_layout_manager.js';
import { initializeViewerUIControls } from './viewer_ui_controls.js';
import { initializeMeshInteraction } from './mesh_interaction_controls.js';
import { initializeTabLoader, loadTabAndUpdate } from './tab_loader.js';

// Global state variables 
let activeIcosphereData = null;
let activeVoronoiData = null;
let landgenTabButtonElement = null;
let defaultVoronoiUniforms = null;
let showElevationBtnElement = null;
let activeBaseMeshSubdivision = -1;
let currentElevationDataSubdivision = -1;

let currentTabHandlerAPI = null;
let currentActiveTabId = null;

// Display mesh references (managed by commonViewerAPI)
let icosphereDisplayMesh = null;
let voronoiDisplayData = null; // { solidMesh, outlineMesh }
let landDisplayMesh = null;
let currentVisibleMeshType = null;

let threeViewerAPI = null;
let tabLoaderAPI = null;

document.addEventListener('DOMContentLoaded', () => {
    console.log("DOM Content Loaded. Initializing app_shell_main.js (v5.9.15 - Texture Mapping Fix)...");

    // --- Get DOM Elements ---
    const viewerContainer = document.getElementById('viewerContainer');
    const statusMessageDiv = document.getElementById('statusMessage');

    const hamburgerBtn = document.getElementById('hamburgerBtn');
    const parametersColumn = document.getElementById('parametersColumn');
    const mainContentArea = document.getElementById('mainContentArea');
    const toggleFullscreenBtn = document.getElementById('toggleFullscreenBtn');

    const showIcosphereBtn = document.getElementById('showIcosphereBtn');
    const showVoronoiBtn = document.getElementById('showVoronoiBtn');
    showElevationBtnElement = document.getElementById('showElevationBtn');
    const showLandBtn = document.getElementById('showLandBtn');
    const viewerWireframeCheckbox = document.getElementById('viewerWireframeCheckbox');
    const viewerAutoRotateCheckbox = document.getElementById('viewerAutoRotateCheckbox');

    landgenTabButtonElement = document.getElementById('landgenTabButton');
    const tabButtons = document.querySelectorAll('.tab-button');
    const parametersTitle = document.getElementById('parametersTitle');
    const parametersFormContainer = document.getElementById('parametersFormContainer');
    const mainGenerateBtn = document.getElementById('mainGenerateBtn');

    // --- Initialize Core 3D Viewer ---
    if (viewerContainer) {
        threeViewerAPI = initializeViewer(viewerContainer); // initializeViewer is from three_viewer.js
        if (!threeViewerAPI) {
            if (statusMessageDiv) {
                statusMessageDiv.textContent = "Critical Error: Could not initialize 3D viewer.";
                statusMessageDiv.className = 'mb-3 p-3 text-sm rounded-lg bg-red-100 border-red-200 text-red-700';
                statusMessageDiv.classList.remove('hidden');
            }
            console.error("app_shell_main: Failed to initialize ThreeViewer.");
            return;
        }
    } else {
        console.error("app_shell_main: viewerContainer DOM element not found!");
        return;
    }

    // --- Define commonViewerAPI ---
    const commonViewerAPI = {
        showStatus: (message, type = 'info') => {
            if (!statusMessageDiv) { console.warn("commonViewerAPI: statusMessageDiv not found."); return; }
            statusMessageDiv.textContent = message;
            statusMessageDiv.className = 'mb-3 p-3 text-sm rounded-lg';
            if (type === 'success') statusMessageDiv.classList.add('bg-green-100', 'border-green-200', 'text-green-700');
            else if (type === 'error') statusMessageDiv.classList.add('bg-red-100', 'border-red-200', 'text-red-700');
            else statusMessageDiv.classList.add('bg-blue-50', 'border-blue-200', 'text-blue-700');
            statusMessageDiv.classList.remove('hidden');
        },
        clearScene: () => {
            console.log("commonViewerAPI: clearScene called.");
            const scene = threeViewerAPI.getScene(); // Use getter from threeViewerAPI
            if (!scene) { return; }

            if (icosphereDisplayMesh) {
                if (icosphereDisplayMesh.geometry) icosphereDisplayMesh.geometry.dispose();
                if (icosphereDisplayMesh.material) {
                    if (Array.isArray(icosphereDisplayMesh.material)) icosphereDisplayMesh.material.forEach(m => m.dispose());
                    else icosphereDisplayMesh.material.dispose();
                }
                scene.remove(icosphereDisplayMesh); icosphereDisplayMesh = null;
            }
            if (voronoiDisplayData) {
                if (voronoiDisplayData.solidMesh) {
                    if (voronoiDisplayData.solidMesh.geometry) voronoiDisplayData.solidMesh.geometry.dispose();
                    if (voronoiDisplayData.solidMesh.material) {
                        // Restore default uniforms before disposing if they were captured
                        if (defaultVoronoiUniforms && voronoiDisplayData.solidMesh.material.uniforms) {
                            const uniforms = voronoiDisplayData.solidMesh.material.uniforms;
                            uniforms.u_useElevationColoring.value = defaultVoronoiUniforms.u_useElevationColoring.value;
                            if (uniforms.u_elevationDataTexture.value && uniforms.u_elevationDataTexture.value !== defaultVoronoiUniforms.u_elevationDataTexture.value) {
                                uniforms.u_elevationDataTexture.value.dispose();
                            }
                            uniforms.u_elevationDataTexture.value = defaultVoronoiUniforms.u_elevationDataTexture.value;
                            uniforms.u_elevationTextureDim.value.copy(defaultVoronoiUniforms.u_elevationTextureDim.value);
                            uniforms.u_minElevation.value = defaultVoronoiUniforms.u_minElevation.value;
                            uniforms.u_maxElevation.value = defaultVoronoiUniforms.u_maxElevation.value;
                        }
                        voronoiDisplayData.solidMesh.material.dispose();
                    }
                    scene.remove(voronoiDisplayData.solidMesh);
                }
                if (voronoiDisplayData.outlineMesh) {
                    if (voronoiDisplayData.outlineMesh.geometry) voronoiDisplayData.outlineMesh.geometry.dispose();
                    if (voronoiDisplayData.outlineMesh.material) {
                        if (Array.isArray(voronoiDisplayData.outlineMesh.material)) voronoiDisplayData.outlineMesh.material.forEach(m => m.dispose());
                        else voronoiDisplayData.outlineMesh.material.dispose();
                    }
                    scene.remove(voronoiDisplayData.outlineMesh);
                }
                voronoiDisplayData = null; defaultVoronoiUniforms = null; // Reset captured defaults
            }
            if (landDisplayMesh) {
                if (landDisplayMesh.geometry) landDisplayMesh.geometry.dispose();
                if (landDisplayMesh.material) {
                    if (Array.isArray(landDisplayMesh.material)) landDisplayMesh.material.forEach(m => m.dispose());
                    else landDisplayMesh.material.dispose();
                }
                scene.remove(landDisplayMesh); landDisplayMesh = null;
            }

            currentVisibleMeshType = null;
            activeIcosphereData = null; activeVoronoiData = null;
            activeBaseMeshSubdivision = -1; currentElevationDataSubdivision = -1;

            if (typeof updateLandgenTabState === 'function') updateLandgenTabState();
            if (typeof updateShowElevationButtonState === 'function') updateShowElevationButtonState();
            if (showLandBtn) showLandBtn.classList.add('hidden'); // Hide land button if scene is cleared
        },
        setIcosphereMesh: (mesh) => { icosphereDisplayMesh = mesh; },
        setVoronoiData: (data) => {
            voronoiDisplayData = data;
            // Capture default uniforms when Voronoi data is first set
            if (voronoiDisplayData && voronoiDisplayData.solidMesh && voronoiDisplayData.solidMesh.material && !defaultVoronoiUniforms) {
                const uniforms = voronoiDisplayData.solidMesh.material.uniforms;
                if (uniforms && uniforms.u_useElevationColoring && uniforms.u_elevationDataTexture && uniforms.u_elevationTextureDim && uniforms.u_minElevation && uniforms.u_maxElevation) {
                    defaultVoronoiUniforms = {
                        u_useElevationColoring: { value: uniforms.u_useElevationColoring.value },
                        u_elevationDataTexture: { value: uniforms.u_elevationDataTexture.value }, // This might be a dummy texture initially
                        u_elevationTextureDim: { value: new THREE.Vector2().copy(uniforms.u_elevationTextureDim.value) },
                        u_minElevation: { value: uniforms.u_minElevation.value },
                        u_maxElevation: { value: uniforms.u_maxElevation.value },
                    };
                    console.log("commonViewerAPI: Default Voronoi uniforms captured:", defaultVoronoiUniforms);
                } else {
                    console.warn("commonViewerAPI: Voronoi shader uniforms not fully available to capture defaults at setVoronoiData time.");
                }
            }
            if (typeof updateShowElevationButtonState === 'function') updateShowElevationButtonState();
        },
        setLandMesh: (mesh) => {
            landDisplayMesh = mesh;
            if (landDisplayMesh && showLandBtn) showLandBtn.classList.remove('hidden');
            else if (showLandBtn) showLandBtn.classList.add('hidden');
        },
        setActiveMeshData: (icoData, voroData, subdivisionLevel) => {
            console.log(`commonViewerAPI: setActiveMeshData called for subdivision: ${subdivisionLevel}`);
            activeIcosphereData = icoData; activeVoronoiData = voroData;
            activeBaseMeshSubdivision = subdivisionLevel !== undefined ? subdivisionLevel : -1;

            // If the base mesh changes, invalidate current elevation data
            if (currentElevationDataSubdivision !== -1 && currentElevationDataSubdivision !== activeBaseMeshSubdivision) {
                console.log("commonViewerAPI: Base mesh changed, resetting current elevation data subdivision and Voronoi coloring.");
                currentElevationDataSubdivision = -1; // Mark elevation data as stale
                if (voronoiDisplayData && voronoiDisplayData.solidMesh && voronoiDisplayData.solidMesh.material.uniforms && defaultVoronoiUniforms) {
                    const uniforms = voronoiDisplayData.solidMesh.material.uniforms;
                    uniforms.u_useElevationColoring.value = defaultVoronoiUniforms.u_useElevationColoring.value; // false
                    // Dispose old texture if it's not the default dummy
                    if (uniforms.u_elevationDataTexture.value && uniforms.u_elevationDataTexture.value !== defaultVoronoiUniforms.u_elevationDataTexture.value) {
                        uniforms.u_elevationDataTexture.value.dispose();
                    }
                    uniforms.u_elevationDataTexture.value = defaultVoronoiUniforms.u_elevationDataTexture.value; // Reset to dummy
                    uniforms.u_elevationTextureDim.value.copy(defaultVoronoiUniforms.u_elevationTextureDim.value);
                    uniforms.u_minElevation.value = defaultVoronoiUniforms.u_minElevation.value;
                    uniforms.u_maxElevation.value = defaultVoronoiUniforms.u_maxElevation.value;
                    voronoiDisplayData.solidMesh.material.needsUpdate = true;
                }
            }
            if (typeof updateLandgenTabState === 'function') updateLandgenTabState();
            if (typeof updateShowElevationButtonState === 'function') updateShowElevationButtonState();
        },
        getActiveMeshData: () => ({
            icosphereData: activeIcosphereData,
            voronoiData: activeVoronoiData,
            subdivisionLevel: activeBaseMeshSubdivision
        }),
        updateVoronoiElevationVisuals: (elevationMap, minElev, maxElev, texWidth, texHeight, baseSubdivisionLevel) => {
            if (!voronoiDisplayData || !voronoiDisplayData.solidMesh || !voronoiDisplayData.solidMesh.material || !voronoiDisplayData.solidMesh.material.uniforms) {
                commonViewerAPI.showStatus("Voronoi mesh not ready for elevation display.", "warning"); return;
            }
            console.log(`commonViewerAPI: Updating Voronoi elevation for base mesh sub ${baseSubdivisionLevel}. MinElev: ${minElev}, MaxElev: ${maxElev}, TexDims: ${texWidth}x${texHeight}`);

            const uniforms = voronoiDisplayData.solidMesh.material.uniforms;

            // Total number of cells in the texture (texWidth * texHeight)
            // This might be larger than numIcosphereSites if texWidth * texHeight was rounded up.
            const textureArraySize = texWidth * texHeight;
            const textureData = new Float32Array(textureArraySize);
            textureData.fill(minElev); // Initialize with a default value (e.g., minElev)

            const numIcosphereSites = activeIcosphereData ? activeIcosphereData.vertices.length / 3 : 0;

            // Iterate through the actual cell IDs (0 to numIcosphereSites - 1)
            // and place their elevation data at the correct 2D texture coordinate.
            for (let cellId = 0; cellId < numIcosphereSites; cellId++) {
                const elevation = elevationMap.has(cellId) ? elevationMap.get(cellId) : minElev;

                // Calculate the 2D texture coordinates for this cellId
                const texX = cellId % texWidth;
                const texY = Math.floor(cellId / texWidth);

                // Calculate the 1D index in the textureData array
                const textureIndex = texY * texWidth + texX;

                if (textureIndex < textureData.length) { // Boundary check
                    textureData[textureIndex] = elevation;
                } else {
                    // This should ideally not happen if texWidth * texHeight >= numIcosphereSites
                    console.warn(`commonViewerAPI.updateVoronoiElevationVisuals: Calculated textureIndex ${textureIndex} is out of bounds for textureData (length ${textureData.length}) for cellId ${cellId}.`);
                }
            }

            // Dispose previous texture if it's not the default one
            if (uniforms.u_elevationDataTexture.value && uniforms.u_elevationDataTexture.value !== (defaultVoronoiUniforms ? defaultVoronoiUniforms.u_elevationDataTexture.value : null)) {
                uniforms.u_elevationDataTexture.value.dispose();
            }

            const elevationTexture = new THREE.DataTexture(textureData, texWidth, texHeight, THREE.RedFormat, THREE.FloatType);
            elevationTexture.needsUpdate = true;

            uniforms.u_elevationDataTexture.value = elevationTexture;
            uniforms.u_elevationTextureDim.value.set(parseFloat(texWidth), parseFloat(texHeight));
            uniforms.u_minElevation.value = parseFloat(minElev);
            uniforms.u_maxElevation.value = parseFloat(maxElev);
            uniforms.u_useElevationColoring.value = true; // Enable elevation coloring
            voronoiDisplayData.solidMesh.material.needsUpdate = true;

            currentElevationDataSubdivision = baseSubdivisionLevel; // Store the subdivision level this data is for
            if (typeof updateShowElevationButtonState === 'function') updateShowElevationButtonState(); // Update button state

            commonViewerAPI.showStatus("Elevation map applied to Voronoi cells.", "success");
            // If Voronoi is not the current mesh, or if elevation button was not active, switch to it
            if (currentVisibleMeshType !== 'voronoi' || (showElevationBtnElement && !showElevationBtnElement.classList.contains('btn-active'))) {
                commonViewerAPI.displayMesh('voronoi');
            }
        },
        displayMesh: (meshType) => {
            const scene = threeViewerAPI.getScene(); // Use getter
            if (!scene) { console.error("commonViewerAPI.displayMesh: Scene not available."); return; }

            // Remove existing meshes from the scene
            if (icosphereDisplayMesh) scene.remove(icosphereDisplayMesh);
            if (voronoiDisplayData) {
                if (voronoiDisplayData.solidMesh) scene.remove(voronoiDisplayData.solidMesh);
                if (voronoiDisplayData.outlineMesh) scene.remove(voronoiDisplayData.outlineMesh);
            }
            if (landDisplayMesh) scene.remove(landDisplayMesh);

            currentVisibleMeshType = meshType;

            // Reset active state for all viewer control buttons
            if (showIcosphereBtn) showIcosphereBtn.classList.remove('btn-active');
            if (showVoronoiBtn) showVoronoiBtn.classList.remove('btn-active');
            if (showElevationBtnElement) showElevationBtnElement.classList.remove('btn-active');
            if (showLandBtn) showLandBtn.classList.remove('btn-active');

            // Add the selected mesh type to the scene
            if (meshType === 'icosphere' && icosphereDisplayMesh) {
                if (!icosphereDisplayMesh.material) { // Ensure material exists
                    icosphereDisplayMesh.material = new THREE.MeshStandardMaterial({ color: 0x00dd00, side: THREE.DoubleSide, metalness: 0.1, roughness: 0.6 });
                }
                scene.add(icosphereDisplayMesh);
                if (showIcosphereBtn) showIcosphereBtn.classList.add('btn-active');
            } else if (meshType === 'voronoi' && voronoiDisplayData) {
                if (voronoiDisplayData.solidMesh) {
                    scene.add(voronoiDisplayData.solidMesh);
                    // Check if elevation coloring is active and valid for this base mesh
                    const useElevColor = voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring.value;
                    const elevDataValid = currentElevationDataSubdivision === activeBaseMeshSubdivision && currentElevationDataSubdivision !== -1;

                    if (useElevColor && elevDataValid) {
                        if (showElevationBtnElement) showElevationBtnElement.classList.add('btn-active');
                    } else {
                        // If elevation coloring is not active or not valid, ensure Voronoi button is active
                        voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring.value = false; // Ensure it's off if data is invalid
                        if (showVoronoiBtn) showVoronoiBtn.classList.add('btn-active');
                    }
                }
                if (voronoiDisplayData.outlineMesh) {
                    scene.add(voronoiDisplayData.outlineMesh);
                }
            } else if (meshType === 'land' && landDisplayMesh) {
                scene.add(landDisplayMesh);
                if (showLandBtn) showLandBtn.classList.add('btn-active');
            } else {
                console.warn(`commonViewerAPI.displayMesh: Mesh type "${meshType}" or its data not available.`);
            }
            // Apply current wireframe state to the newly displayed mesh
            if (viewerWireframeCheckbox) commonViewerAPI.setWireframe(viewerWireframeCheckbox.checked);
        },
        getViewerControlsState: () => ({ wireframe: viewerWireframeCheckbox ? viewerWireframeCheckbox.checked : true }),
        onWindowResize: () => {
            if (threeViewerAPI && viewerContainer) {
                threeViewerAPI.resize(viewerContainer.clientWidth, viewerContainer.clientHeight);
            }
        },
        getCurrentMeshType: () => currentVisibleMeshType,
        getRendererDomElement: () => threeViewerAPI ? threeViewerAPI.getDomElement() : null,
        getOrbitControls: () => threeViewerAPI ? threeViewerAPI.getControls() : null,
        getCamera: () => threeViewerAPI ? threeViewerAPI.getCamera() : null,
        setAutoRotate: (enabled) => {
            if (threeViewerAPI) threeViewerAPI.setAutoRotate(enabled);
        },
        getVoronoiDisplayData: () => voronoiDisplayData,
        getIcosphereDisplayMesh: () => icosphereDisplayMesh,
        getLandDisplayMesh: () => landDisplayMesh,
        getCurrentElevationDataSubdivision: () => currentElevationDataSubdivision,
        setWireframe: (isWireframe) => {
            console.log(`commonViewerAPI: Setting wireframe to ${isWireframe}`);
            const mesh = currentVisibleMeshType === 'icosphere' ? icosphereDisplayMesh :
                currentVisibleMeshType === 'land' ? landDisplayMesh : null;

            if (mesh && mesh.material) {
                if (mesh.material.hasOwnProperty('wireframe')) { // Standard materials
                    mesh.material.wireframe = isWireframe;
                }
            } else if (currentVisibleMeshType === 'voronoi' && voronoiDisplayData) {
                if (voronoiDisplayData.solidMesh) {
                    // Solid mesh is always visible if Voronoi is the current type.
                    // Its appearance (solid vs. internal wireframe) is controlled by u_edgeFeather.
                    voronoiDisplayData.solidMesh.visible = true;
                    if (voronoiDisplayData.solidMesh.material && voronoiDisplayData.solidMesh.material.uniforms) {
                        // When wireframe is ON, we want solid faces *plus* the N-gon outline.
                        // So, u_edgeFeather should be -1.0 to show solid faces.
                        // When wireframe is OFF, we also want solid faces.
                        voronoiDisplayData.solidMesh.material.uniforms.u_edgeFeather.value = -1.0;
                    }
                }
                if (voronoiDisplayData.outlineMesh) {
                    // Show the N-gon outline mesh only when wireframe is ON.
                    voronoiDisplayData.outlineMesh.visible = isWireframe;
                }
            }
        }
    };
    window.worldgenViewerAPI = commonViewerAPI;

    // --- Initialize Shell Layout Manager ---
    initializeShellLayout(commonViewerAPI, {
        hamburgerBtn, parametersColumn, mainContentArea, toggleFullscreenBtn, viewerContainer
    });

    // --- Initialize Viewer UI Controls ---
    initializeViewerUIControls(commonViewerAPI, {
        showIcosphereBtn, showVoronoiBtn, showElevationBtnElement,
        showLandBtn, viewerWireframeCheckbox, viewerAutoRotateCheckbox
    });

    // --- Initialize Mesh Interaction Controls ---
    initializeMeshInteraction(commonViewerAPI);

    // --- Initialize Tab Loader ---
    const tabLoaderConfig = {
        parametersTitle, parametersFormContainer, allTabButtons: tabButtons,
        mainGenerateBtn, landgenTabButtonElement, commonViewerAPI,
        setCurrentTabHandlerAPI: (handler) => { currentTabHandlerAPI = handler; },
        setCurrentActiveTabId: (tabId) => { currentActiveTabId = tabId; },
        getCurrentTabHandlerAPI: () => currentTabHandlerAPI
    };
    tabLoaderAPI = initializeTabLoader(tabLoaderConfig);

    // --- Full definitions for UI state update functions ---
    function updateLandgenTabState() {
        if (landgenTabButtonElement) {
            const icosphereReady = activeIcosphereData && activeIcosphereData.vertices && activeIcosphereData.faces;
            const voronoiReady = activeVoronoiData && activeVoronoiData.vertices && activeVoronoiData.cells;
            const canEnableLandgen = icosphereReady && voronoiReady;
            const previousDisabledState = landgenTabButtonElement.disabled;

            landgenTabButtonElement.disabled = !canEnableLandgen;
            landgenTabButtonElement.title = canEnableLandgen ?
                "Generate land based on the current Icosphere/Voronoi mesh." :
                "Generate or load an Icosphere & Voronoi mesh first to enable land generation.";

            if (!canEnableLandgen && currentActiveTabId === 'landgen') {
                const firstAvailableTab = document.querySelector('.tab-button:not([disabled]):not(#landgenTabButton)');
                if (firstAvailableTab) { if (tabLoaderAPI && tabLoaderAPI.loadTab) tabLoaderAPI.loadTab(firstAvailableTab); }
                else { const firstOverallTab = document.querySelector('.tab-button'); if (firstOverallTab && firstOverallTab !== landgenTabButtonElement && tabLoaderAPI && tabLoaderAPI.loadTab) tabLoaderAPI.loadTab(firstOverallTab); }
            }
            // Manage click listener based on disabled state
            if (canEnableLandgen && previousDisabledState === true) { // Tab was disabled, now enabled
                if (!landgenTabButtonElement._hasClickListener) {
                    landgenTabButtonElement.addEventListener('click', handleLandgenTabClick);
                    landgenTabButtonElement._hasClickListener = true;
                }
            } else if (!canEnableLandgen && landgenTabButtonElement._hasClickListener) { // Tab was enabled, now disabled
                landgenTabButtonElement.removeEventListener('click', handleLandgenTabClick);
                landgenTabButtonElement._hasClickListener = false;
            }
        }
    }

    function handleLandgenTabClick() {
        if (tabLoaderAPI && tabLoaderAPI.loadTab) {
            tabLoaderAPI.loadTab(landgenTabButtonElement);
        } else {
            // Fallback or error if tabLoaderAPI.loadTab isn't ready
            console.warn("TabLoader API not ready for handleLandgenTabClick, attempting direct call.");
            loadTabAndUpdate(landgenTabButtonElement, tabLoaderConfig);
        }
    }

    function updateShowElevationButtonState() {
        if (showElevationBtnElement) {
            const hasVoronoiMeshAndMaterial = voronoiDisplayData &&
                voronoiDisplayData.solidMesh &&
                voronoiDisplayData.solidMesh.material &&
                voronoiDisplayData.solidMesh.material.uniforms &&
                voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring;

            const elevationDataMatchesBase = currentElevationDataSubdivision !== -1 &&
                activeBaseMeshSubdivision !== -1 &&
                currentElevationDataSubdivision === activeBaseMeshSubdivision;

            const canEnableAndShowElevation = hasVoronoiMeshAndMaterial && elevationDataMatchesBase;

            showElevationBtnElement.disabled = !canEnableAndShowElevation;
            showElevationBtnElement.title = canEnableAndShowElevation ?
                "Show elevation data on Voronoi mesh" :
                "Generate/Load Voronoi mesh and then generate Land data for that specific mesh to enable elevation view.";

            // If elevation cannot be shown but was previously active, reset Voronoi view
            if (!canEnableAndShowElevation && hasVoronoiMeshAndMaterial &&
                voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring.value) {

                voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring.value = false; // Turn off elevation coloring
                if (currentVisibleMeshType === 'voronoi') { // If Voronoi is currently visible
                    if (showVoronoiBtn) showVoronoiBtn.classList.add('btn-active'); // Make Voronoi button active
                    if (showElevationBtnElement) showElevationBtnElement.classList.remove('btn-active'); // Deactivate elevation button
                }
            }
        }
    }
    // --- END: Full definitions for UI state update functions ---

    // Initial state updates
    updateLandgenTabState();
    updateShowElevationButtonState();

    // --- Main Generate Button Listener ---
    if (mainGenerateBtn) {
        mainGenerateBtn.addEventListener('click', async (event) => {
            event.preventDefault();
            console.log("Shell: Main Generate/Load/Process Button clicked.");
            if (currentTabHandlerAPI && typeof currentTabHandlerAPI.generate === 'function') {
                console.log("Shell: Delegating action to active tab handler.");
                if (!threeViewerAPI.getScene()) initializeViewer(viewerContainer); // Ensure viewer is up
                currentTabHandlerAPI.generate();
            } else {
                if (commonViewerAPI && commonViewerAPI.showStatus) commonViewerAPI.showStatus("No active action logic for this tab.", "error");
                console.warn("Shell: mainGenerateBtn clicked, but no active tab handler or generate function.");
            }
        });
    }

    // --- Initial Tab Loading ---
    const initialActiveTab = document.querySelector('.tab-button.active-tab');
    if (initialActiveTab) {
        if (initialActiveTab.id === 'landgenTabButton' && initialActiveTab.disabled) {
            const firstAvailableTab = document.querySelector('.tab-button:not([disabled]):not(#landgenTabButton)');
            if (firstAvailableTab) { if (tabLoaderAPI && tabLoaderAPI.loadTab) tabLoaderAPI.loadTab(firstAvailableTab); else loadTabAndUpdate(firstAvailableTab, tabLoaderConfig); }
            else { const fOT = document.querySelector('.tab-button'); if (fOT && fOT !== landgenTabButtonElement && tabLoaderAPI && tabLoaderAPI.loadTab) tabLoaderAPI.loadTab(fOT); else if (fOT) loadTabAndUpdate(fOT, tabLoaderConfig); }
        } else {
            if (tabLoaderAPI && tabLoaderAPI.loadTab) tabLoaderAPI.loadTab(initialActiveTab); else loadTabAndUpdate(initialActiveTab, tabLoaderConfig);
        }
    } else if (tabButtons.length > 0) {
        const firstAvailableTab = document.querySelector('.tab-button:not([disabled])');
        if (firstAvailableTab) { if (tabLoaderAPI && tabLoaderAPI.loadTab) tabLoaderAPI.loadTab(firstAvailableTab); else loadTabAndUpdate(firstAvailableTab, tabLoaderConfig); }
        else if (tabButtons[0]) { if (tabLoaderAPI && tabLoaderAPI.loadTab) tabLoaderAPI.loadTab(tabButtons[0]); else loadTabAndUpdate(tabButtons[0], tabLoaderConfig); }
    }

    console.log("app_shell_main.js: All core modules initialized and initial tab loaded.");
});
