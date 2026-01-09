// Path: static/js/icosphere_handler.js
import * as THREE from 'three';

let meshWorker = null;
let commonViewerAPIInstance = null;

let currentSubdivisions = -1; // Tracks the subdivision value from the form for this tab
let mainGenerateBtnRef = null;
let generationFormRef = null;

const ACTION_GENERATE = 'generate';
const ACTION_LOAD = 'load';
let currentButtonAction = ACTION_GENERATE;

// Store raw data from server/cache to pass to app_shell
let rawIcosphereDataFromServer = null;
let rawVoronoiDataFromServer = null;

async function checkSavedMesh(subdivisions) {
    if (!commonViewerAPIInstance) {
        console.warn("IcosphereHandler: commonViewerAPIInstance not available for checkSavedMesh.");
        return;
    }
    if (!mainGenerateBtnRef) {
        mainGenerateBtnRef = document.getElementById('mainGenerateBtn');
        if (!mainGenerateBtnRef) {
            console.warn("IcosphereHandler: Main generate button (mainGenerateBtn) NOT FOUND. Cannot update button.");
            return;
        }
    }

    if (typeof subdivisions !== 'number' || subdivisions < 0 || isNaN(subdivisions)) {
        console.warn(`IcosphereHandler: Invalid subdivision value (${subdivisions}). Defaulting button to GENERATE.`);
        if (mainGenerateBtnRef) {
            mainGenerateBtnRef.textContent = 'Generate';
            mainGenerateBtnRef.classList.remove('btn-secondary');
            mainGenerateBtnRef.classList.add('btn-primary');
            mainGenerateBtnRef.disabled = false;
            currentButtonAction = ACTION_GENERATE;
        }
        return;
    }

    try {
        const response = await fetch(`/api/check_saved_mesh?subdivisions=${subdivisions}`);
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`HTTP ${response.status}: ${errorText}`);
        }
        const result = await response.json();

        if (result.icosphereExists === true) {
            mainGenerateBtnRef.textContent = `Load Cached (Sub ${subdivisions})`;
            mainGenerateBtnRef.classList.remove('btn-primary');
            mainGenerateBtnRef.classList.add('btn-secondary');
            currentButtonAction = ACTION_LOAD;
        } else {
            mainGenerateBtnRef.textContent = 'Generate';
            mainGenerateBtnRef.classList.remove('btn-secondary');
            mainGenerateBtnRef.classList.add('btn-primary');
            currentButtonAction = ACTION_GENERATE;
        }
        mainGenerateBtnRef.disabled = false;
    } catch (error) {
        console.error('IcosphereHandler: Error in checkSavedMesh fetch/processing:', error);
        if (commonViewerAPIInstance && commonViewerAPIInstance.showStatus) {
            commonViewerAPIInstance.showStatus(`Error checking cache: ${error.message}`, 'error');
        }
        if (mainGenerateBtnRef) {
            mainGenerateBtnRef.textContent = 'Generate';
            mainGenerateBtnRef.classList.remove('btn-secondary');
            mainGenerateBtnRef.classList.add('btn-primary');
            mainGenerateBtnRef.disabled = false;
            currentButtonAction = ACTION_GENERATE;
        }
    }
}

function handleSubdivisionChange(event) {
    const newSubdivisions = parseInt(event.target.value, 10);
    if (!isNaN(newSubdivisions) && newSubdivisions >= 0 && newSubdivisions !== currentSubdivisions) {
        currentSubdivisions = newSubdivisions; // Update the tab-local currentSubdivisions
        checkSavedMesh(currentSubdivisions);
    } else if (isNaN(newSubdivisions) || newSubdivisions < 0) {
        if (mainGenerateBtnRef) {
            mainGenerateBtnRef.textContent = 'Generate';
            mainGenerateBtnRef.classList.remove('btn-secondary');
            mainGenerateBtnRef.classList.add('btn-primary');
            mainGenerateBtnRef.disabled = false;
        }
        currentButtonAction = ACTION_GENERATE;
        currentSubdivisions = -1;
    }
}

function initializeIcosphereTab(commonViewerAPI) {
    console.log("Initializing Icosphere Tab specific JavaScript (v5.9.6 - Pass Subdivision to Shell)...");
    commonViewerAPIInstance = commonViewerAPI;
    generationFormRef = document.getElementById('generationForm');

    if (!generationFormRef) {
        console.error("IcosphereHandler: Generation form not found!");
        return null;
    }

    const subdivisionsInput = generationFormRef.querySelector('#subdivisions');
    if (subdivisionsInput) {
        const initialSubdivisions = parseInt(subdivisionsInput.value, 10);
        currentSubdivisions = (!isNaN(initialSubdivisions) && initialSubdivisions >= 0) ? initialSubdivisions : -1;
        // Event listeners are added below, updateGenerateButton will call checkSavedMesh
    } else {
        console.warn("IcosphereHandler: Subdivisions input not found.");
    }

    let localIcosphereMesh = null;
    let localVoronoiData = null;

    rawIcosphereDataFromServer = null;
    rawVoronoiDataFromServer = null;


    function getIcosphereFormData() {
        const formData = new FormData(generationFormRef);
        const params = {};
        for (let [key, value] of formData.entries()) {
            const inputElement = generationFormRef.elements[key];
            if (!inputElement) continue;
            if (inputElement.type === 'checkbox') params[key] = inputElement.checked;
            else if (inputElement.type === 'number') {
                const floatVal = parseFloat(value);
                if (key === 'subdivisions' || key === 'relaxMaxIterations') params[key] = parseInt(value, 10);
                else params[key] = floatVal;
            } else params[key] = value;
        }
        if (params.subdivisions !== undefined) params.subdivisions = parseInt(params.subdivisions, 10);
        if (params.relaxMaxIterations !== undefined) params.relaxMaxIterations = parseInt(params.relaxMaxIterations, 10);
        return params;
    }

    function createIcosphereMeshFromWorkerData(processedData) {
        if (localIcosphereMesh) {
            if (localIcosphereMesh.geometry) localIcosphereMesh.geometry.dispose();
            if (localIcosphereMesh.material) localIcosphereMesh.material.dispose();
            localIcosphereMesh = null;
        }
        if (!processedData || !processedData.vertices || !processedData.faces || processedData.faces.length === 0) {
            console.error("IcosphereHandler: Invalid or empty processed icosphere data from worker.", processedData);
            return null;
        }
        const geometry = new THREE.BufferGeometry();
        try {
            geometry.setAttribute('position', new THREE.BufferAttribute(processedData.vertices, 3));
            geometry.setIndex(new THREE.BufferAttribute(processedData.faces, 1));
            if (!geometry.index || !geometry.index.array || geometry.index.array.length === 0) throw new Error("Invalid index.");
            geometry.computeVertexNormals();
        } catch (e) {
            console.error("IcosphereHandler: Error creating BufferGeometry for icosphere:", e);
            if (geometry) geometry.dispose(); return null;
        }
        return new THREE.Mesh(geometry);
    }

    function createVoronoiObjectsFromWorkerData(processedVoronoi) {
        // This function is from icosphere_handler_updated_elevation_shader
        // and should contain the shader setup with elevation uniforms.
        if (localVoronoiData) { /* ... dispose previous ... */ }
        if (!processedVoronoi || (!processedVoronoi.solid && !processedVoronoi.outline)) { return null; }
        let solidMesh = null;
        if (processedVoronoi.solid && processedVoronoi.solid.positions && processedVoronoi.solid.positions.length > 0) {
            const solidGeometry = new THREE.BufferGeometry();
            solidGeometry.setAttribute('position', new THREE.BufferAttribute(processedVoronoi.solid.positions, 3));
            solidGeometry.computeVertexNormals();
            if (processedVoronoi.solid.cellIds) solidGeometry.setAttribute('a_cellId', new THREE.BufferAttribute(processedVoronoi.solid.cellIds, 1));
            if (processedVoronoi.solid.barycentrics) solidGeometry.setAttribute('a_barycentric', new THREE.BufferAttribute(processedVoronoi.solid.barycentrics, 3));

            const vertexShader = `
                attribute float a_cellId; attribute vec3 a_barycentric;
                varying float v_cellId; varying vec3 v_barycentric; varying vec3 v_normal;
                void main() { v_cellId = a_cellId; v_barycentric = a_barycentric; vec4 mvPosition = modelViewMatrix * vec4(position, 1.0); v_normal = normalize(normalMatrix * normal); gl_Position = projectionMatrix * mvPosition; }`;
            const fragmentShader = `
                varying float v_cellId; varying vec3 v_barycentric; varying vec3 v_normal;
                uniform float u_edgeFeather; uniform vec3 u_wireframeColor;
                uniform bool u_useElevationColoring;
                uniform sampler2D u_elevationDataTexture; uniform vec2 u_elevationTextureDim;
                uniform float u_minElevation; uniform float u_maxElevation;
                vec3 hsv2rgb(vec3 c) { vec4 K = vec4(1.0, 2.0 / 3.0, 1.0 / 3.0, 3.0); vec3 p = abs(fract(c.xxx + K.xyz) * 6.0 - K.www); return c.z * mix(K.xxx, clamp(p - K.xxx, 0.0, 1.0), c.y); }
                vec3 getCellColorById(float cellId) { float hue = fract(cellId * 0.618033988749895); return hsv2rgb(vec3(hue, 0.65, 0.85)); }
                vec3 getElevationColor(float t) {
                    t = clamp(t, 0.0, 1.0); const int NUM_COLORS = 7; vec3 colors[NUM_COLORS];
                    colors[0] = vec3(0.0, 0.0, 0.5);    colors[1] = vec3(0.0, 0.5, 1.0); 
                    colors[2] = vec3(0.2, 0.8, 0.2);    colors[3] = vec3(0.8, 0.8, 0.2);
                    colors[4] = vec3(1.0, 0.5, 0.0);    colors[5] = vec3(0.6, 0.2, 0.0);
                    colors[6] = vec3(1.0, 1.0, 1.0);   
                    float stops[NUM_COLORS]; stops[0] = 0.0; stops[1] = 0.15; stops[2] = 0.3; stops[3] = 0.5;
                    stops[4] = 0.7; stops[5] = 0.85; stops[6] = 1.0;
                    if (t <= stops[0]) return colors[0];
                    for (int i = 1; i < NUM_COLORS; i++) { if (t < stops[i]) { float b = (t - stops[i-1]) / (stops[i] - stops[i-1]); return mix(colors[i-1], colors[i], b); } }
                    return colors[NUM_COLORS-1];
                }
                float edgeFactor(vec3 bary, float feather) { if (feather < 0.0) return 1.0; vec3 d = fwidth(bary); vec3 a3 = smoothstep(vec3(0.0), d * feather, bary); return min(min(a3.x, a3.y), a3.z); }
                void main() { 
                    vec3 finalCellColor;
                    if (u_useElevationColoring && u_elevationTextureDim.x > 0.0 && u_elevationTextureDim.y > 0.0) {
                        float texX = mod(v_cellId, u_elevationTextureDim.x); float texY = floor(v_cellId / u_elevationTextureDim.x);
                        vec2 uv = vec2((texX + 0.5) / u_elevationTextureDim.x, (texY + 0.5) / u_elevationTextureDim.y);
                        float elevation = texture2D(u_elevationDataTexture, uv).r;
                        float normalizedElevation = (elevation - u_minElevation) / (u_maxElevation - u_minElevation + 0.00001);
                        finalCellColor = getElevationColor(normalizedElevation);
                    } else { finalCellColor = getCellColorById(v_cellId); }
                    vec3 N = normalize(v_normal); vec3 L = normalize(vec3(1.0, 1.0, 0.8)); float lambertian = max(dot(N, L), 0.0);
                    vec3 ambientColor = vec3(0.3); vec3 diffuseColor = finalCellColor * lambertian * 0.7;
                    vec3 lightingColor = ambientColor * finalCellColor + diffuseColor; 
                    float ef = edgeFactor(v_barycentric, u_edgeFeather); 
                    gl_FragColor = vec4(mix(u_wireframeColor, lightingColor, ef), 1.0); 
                }`;
            const dummyElevData = new Float32Array([0]);
            const dummyElevationTexture = new THREE.DataTexture(dummyElevData, 1, 1, THREE.RedFormat, THREE.FloatType);
            dummyElevationTexture.needsUpdate = true;
            const shaderMaterial = new THREE.ShaderMaterial({
                uniforms: {
                    u_edgeFeather: { value: -1.0 }, u_wireframeColor: { value: new THREE.Color(0x000000) },
                    u_useElevationColoring: { value: false },
                    u_elevationDataTexture: { value: dummyElevationTexture },
                    u_elevationTextureDim: { value: new THREE.Vector2(1.0, 1.0) },
                    u_minElevation: { value: 0.0 }, u_maxElevation: { value: 1.0 },
                }, vertexShader, fragmentShader, side: THREE.DoubleSide,
            });
            solidMesh = new THREE.Mesh(solidGeometry, shaderMaterial);
        }
        let outlineMesh = null;
        if (processedVoronoi.outline && processedVoronoi.outline.points && processedVoronoi.outline.points.length > 0) {
            const outlineGeometry = new THREE.BufferGeometry();
            outlineGeometry.setAttribute('position', new THREE.Float32BufferAttribute(processedVoronoi.outline.points, 3));
            const outlineMaterial = new THREE.LineBasicMaterial({ color: 0x000000, linewidth: 1.5 });
            outlineMesh = new THREE.LineSegments(outlineGeometry, outlineMaterial);
        }
        localVoronoiData = { solidMesh, outlineMesh };
        return localVoronoiData;
    }

    async function performAction() {
        if (!generationFormRef) { /* ... error ... */ return; }
        const params = getIcosphereFormData(); // This params object contains the 'subdivisions' value
        let endpointUrl = '';
        let fetchOptions = {};
        let actionTypeForLogging = currentButtonAction;

        if (currentButtonAction === ACTION_LOAD) { /* ... load logic ... */ }
        else { /* ... generate logic ... */ }
        // (The existing fetch and worker logic from v5.8/v5.9.6 remains here)
        // ...
        // MODIFICATION POINT: Inside meshWorker.onmessage, after successful processing:
        // commonViewerAPIInstance.setActiveMeshData(rawIcosphereDataFromServer, rawVoronoiDataFromServer, params.subdivisions);
        // ...
        // And on error:
        // commonViewerAPIInstance.setActiveMeshData(null, null, -1);
        // (Full performAction logic from v5.8/v5.9.6 should be here)
        if (currentButtonAction === ACTION_LOAD) {
            commonViewerAPIInstance.showStatus(`Loading cached data for subdivision ${params.subdivisions}...`, 'info');
            let typeToLoad = params.voronoiEnable ? "both" : "icosphere";
            endpointUrl = `/api/load_mesh?subdivisions=${params.subdivisions}&type=${typeToLoad}`;
            fetchOptions = { method: 'GET' };
        } else {
            commonViewerAPIInstance.showStatus('Requesting Icosphere/Voronoi generation...', 'info');
            endpointUrl = '/api/generate';
            fetchOptions = {
                method: 'POST',
                headers: { 'Content-Type': 'application/json', },
                body: JSON.stringify(params),
            };
        }

        try {
            const response = await fetch(endpointUrl, fetchOptions);
            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`HTTP error! ${response.status}: ${errorText} (Action: ${actionTypeForLogging})`);
            }
            const resultFromServer = await response.json();
            console.log(`Icosphere Tab: Raw data from server (Action: ${actionTypeForLogging}):`, resultFromServer);

            if (resultFromServer.status !== 'success') {
                throw new Error(resultFromServer.message || `Server error: ${resultFromServer.status}`);
            }

            rawIcosphereDataFromServer = resultFromServer.icosphereData || null;
            rawVoronoiDataFromServer = resultFromServer.voronoiData || null;

            commonViewerAPIInstance.showStatus('Processing received data...', 'info');

            if (meshWorker) meshWorker.terminate();
            meshWorker = new Worker('/static/js/mesh_worker.js');

            meshWorker.onmessage = function (e) {
                const { status, icosphere, voronoi } = e.data;
                if (status === 'complete') {
                    commonViewerAPIInstance.clearScene();
                    let icosphereProcessed = false, voronoiProcessed = false;

                    if (icosphere) {
                        localIcosphereMesh = createIcosphereMeshFromWorkerData(icosphere);
                        if (localIcosphereMesh) {
                            commonViewerAPIInstance.setIcosphereMesh(localIcosphereMesh);
                            icosphereProcessed = true;
                        } else commonViewerAPIInstance.showStatus("Error creating icosphere mesh.", "error");
                    }

                    const voronoiExpected = params.voronoiEnable;
                    if (voronoi && voronoiExpected) {
                        localVoronoiData = createVoronoiObjectsFromWorkerData(voronoi);
                        if (localVoronoiData && (localVoronoiData.solidMesh || localVoronoiData.outlineMesh)) {
                            commonViewerAPIInstance.setVoronoiData(localVoronoiData);
                            voronoiProcessed = true;
                        } else if (voronoiExpected) commonViewerAPIInstance.showStatus("Error creating Voronoi objects.", "error");
                    }

                    // --- MODIFIED: Pass subdivision level to setActiveMeshData ---
                    const subdivisionLevelForActiveData = params.subdivisions; // Get from form params
                    if (icosphereProcessed && (voronoiExpected ? voronoiProcessed : true)) {
                        commonViewerAPIInstance.setActiveMeshData(rawIcosphereDataFromServer, rawVoronoiDataFromServer, subdivisionLevelForActiveData);
                    } else {
                        commonViewerAPIInstance.setActiveMeshData(null, null, -1); // Clear with invalid subdivision
                    }
                    // --- END MODIFICATION ---

                    let displayPreference = 'icosphere';
                    if (voronoiExpected && voronoiProcessed) displayPreference = 'voronoi';
                    else if (!icosphereProcessed) displayPreference = null;

                    if (displayPreference) commonViewerAPIInstance.displayMesh(displayPreference);
                    else commonViewerAPIInstance.showStatus('No mesh data to display.', 'info');

                    commonViewerAPIInstance.showStatus('Data processed and meshes updated.', 'success');

                } else if (status === 'error') {
                    commonViewerAPIInstance.showStatus(`Worker Error: ${e.data.message}`, 'error');
                    commonViewerAPIInstance.setActiveMeshData(null, null, -1); // Clear on error
                }
                if (meshWorker) meshWorker.terminate(); meshWorker = null;
                if (actionTypeForLogging === ACTION_GENERATE) checkSavedMesh(params.subdivisions);
            };
            meshWorker.onerror = function (error) {
                console.error('IcosphereHandler: Error from meshWorker:', error.message, error);
                commonViewerAPIInstance.showStatus(`Worker Error: ${error.message}`, 'error');
                if (meshWorker) meshWorker.terminate(); meshWorker = null;
                commonViewerAPIInstance.setActiveMeshData(null, null, -1); // Clear on error
            };
            meshWorker.postMessage({
                type: 'generate',
                icosphereData: resultFromServer.icosphereData,
                voronoiData: resultFromServer.voronoiData,
                voronoiEnabled: params.voronoiEnable
            });
        } catch (error) {
            console.error(`Error during ${actionTypeForLogging}:`, error);
            commonViewerAPIInstance.showStatus(`Error: ${error.message}`, 'error');
            if (meshWorker) { meshWorker.terminate(); meshWorker = null; }
            commonViewerAPIInstance.setActiveMeshData(null, null, -1); // Clear on error
            if (actionTypeForLogging === ACTION_LOAD) checkSavedMesh(params.subdivisions);
        }
    }


    const handlerAPI = {
        generate: performAction,
        updateGenerateButton: () => {
            if (!mainGenerateBtnRef) mainGenerateBtnRef = document.getElementById('mainGenerateBtn');
            const subdivInput = generationFormRef ? generationFormRef.querySelector('#subdivisions') : null;
            let currentSubdivValue = subdivInput && subdivInput.value !== "" ? parseInt(subdivInput.value, 10) : -1;

            if (!isNaN(currentSubdivValue) && currentSubdivValue >= 0) {
                checkSavedMesh(currentSubdivValue);
            } else {
                if (mainGenerateBtnRef) {
                    mainGenerateBtnRef.textContent = 'Generate';
                    mainGenerateBtnRef.classList.remove('btn-secondary');
                    mainGenerateBtnRef.classList.add('btn-primary');
                    mainGenerateBtnRef.disabled = false;
                    currentButtonAction = ACTION_GENERATE;
                }
            }
        },
        dispose: () => {
            console.log("Disposing Icosphere Tab specific resources...");
            if (meshWorker) { meshWorker.terminate(); meshWorker = null; }
            if (localIcosphereMesh) { /* ... */ }
            if (localVoronoiData) { /* ... */ }
            localIcosphereMesh = null; localVoronoiData = null;
            rawIcosphereDataFromServer = null; rawVoronoiDataFromServer = null;

            const subdivisionsInputInstance = generationFormRef ? generationFormRef.querySelector('#subdivisions') : null;
            if (subdivisionsInputInstance) {
                subdivisionsInputInstance.removeEventListener('change', handleSubdivisionChange);
                subdivisionsInputInstance.removeEventListener('input', handleSubdivisionChange);
            }
        }
    };

    if (typeof handlerAPI.updateGenerateButton === 'function') {
        handlerAPI.updateGenerateButton();
    }

    if (subdivisionsInput) { // Add listeners after updateGenerateButton has run once
        subdivisionsInput.addEventListener('change', handleSubdivisionChange);
        subdivisionsInput.addEventListener('input', handleSubdivisionChange);
    }

    // UI conditional visibility logic (remains the same)
    // ... 

    console.log("Icosphere Tab specific JavaScript Initialized and handler ready.");
    return handlerAPI;
}

window.initializeIcosphereTab = initializeIcosphereTab;
