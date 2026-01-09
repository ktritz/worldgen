import * as THREE from 'three';
import { OBJLoader } from 'three/addons/loaders/OBJLoader.js';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';
import { mergeGeometries } from 'three/addons/utils/BufferGeometryUtils.js';

const DEFAULT_CAMERA_FOV = 60;
const MIN_CAMERA_FOV = 5;
const FOV_ZOOM_IN_FACTOR = 0.9;
const FOV_ZOOM_OUT_FACTOR = 1 / FOV_ZOOM_IN_FACTOR;

let isRightMouseDownForMeshRotation = false;
let previousMousePositionForMeshRotation = { x: 0, y: 0 };
const MESH_ROTATION_SENSITIVITY = 0.001; // User confirmed this value
const REFERENCE_ROTATION_DISTANCE = 3.0;
const REFERENCE_ROTATION_FOV = DEFAULT_CAMERA_FOV;

// --- START: Variables for new features ---
let activeIcosphereData = null; // Stores raw icosphere data { vertices, faces } from server/cache
let activeVoronoiData = null;   // Stores raw voronoi data { vertices, cells } from server/cache
let landgenTabButtonElement = null; // Reference to the Landgen tab button
let defaultVoronoiUniforms = null; // To store initial Voronoi shader state

let showElevationBtnElement = null; // Button to toggle elevation view
let activeBaseMeshSubdivision = -1; // Subdivision level of the current Icosphere/Voronoi base
let currentElevationDataSubdivision = -1; // Subdivision level for which elevation data was generated
// --- END: Variables for new features ---

document.addEventListener('DOMContentLoaded', () => {
    console.log("DOM Content Loaded. Initializing app_shell.js (v5.9.7 - Complete Merge & All Features)...");

    // --- Global Shell UI Elements ---
    const hamburgerBtn = document.getElementById('hamburgerBtn');
    const parametersColumn = document.getElementById('parametersColumn');
    const parametersTitle = document.getElementById('parametersTitle');
    const parametersFormContainer = document.getElementById('parametersFormContainer');
    const mainGenerateBtn = document.getElementById('mainGenerateBtn');
    const mainContentArea = document.getElementById('mainContentArea');
    const tabButtons = document.querySelectorAll('.tab-button');
    landgenTabButtonElement = document.getElementById('landgenTabButton');
    showElevationBtnElement = document.getElementById('showElevationBtn');

    // --- Common Viewer Elements ---
    const statusMessageDiv = document.getElementById('statusMessage');
    const viewerContainer = document.getElementById('viewerContainer');
    const showIcosphereBtn = document.getElementById('showIcosphereBtn');
    const showVoronoiBtn = document.getElementById('showVoronoiBtn');
    const showLandBtn = document.getElementById('showLandBtn');
    const viewerWireframeCheckbox = document.getElementById('viewerWireframeCheckbox');
    const viewerAutoRotateCheckbox = document.getElementById('viewerAutoRotateCheckbox');
    const toggleFullscreenBtn = document.getElementById('toggleFullscreenBtn');

    let currentTabHandlerAPI = null;
    let currentActiveTabId = null;

    // --- Three.js variables ---
    let scene, camera, renderer, controls;
    let icosphereDisplayMesh = null;
    let voronoiDisplayData = null; // { solidMesh, outlineMesh } - These are THREE.Mesh objects
    let landDisplayMesh = null;    // For displaying land/terrain, also a THREE.Mesh
    let currentVisibleMeshType = null;
    let commonAnimationFrameId = null;

    if (viewerWireframeCheckbox) viewerWireframeCheckbox.checked = true;
    else console.warn("Shell: viewerWireframeCheckbox not found.");

    if (viewerAutoRotateCheckbox) viewerAutoRotateCheckbox.checked = false;
    else console.warn("Shell: viewerAutoRotateCheckbox not found.");

    // --- START: Function to update Show Elevation button state ---
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

            const canShowElevation = hasVoronoiMeshAndMaterial && elevationDataMatchesBase;

            showElevationBtnElement.disabled = !canShowElevation;
            showElevationBtnElement.title = canShowElevation ?
                "Show elevation data on Voronoi mesh" :
                "Generate/Load Voronoi mesh and then generate Land data for that specific mesh to enable elevation view.";

            // Button is always visible (no 'hidden' class management here for visibility)
            // Its active state will be handled by displayMesh

            if (!canShowElevation && hasVoronoiMeshAndMaterial &&
                voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring.value) {

                voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring.value = false;
                if (currentVisibleMeshType === 'voronoi') {
                    if (showVoronoiBtn) showVoronoiBtn.classList.add('btn-active');
                    if (showElevationBtnElement) showElevationBtnElement.classList.remove('btn-active');
                }
            }
        }
    }
    // --- END: Function to update Show Elevation button state ---

    // --- Function to update Landgen tab enabled state ---
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
                if (firstAvailableTab) {
                    loadTab(firstAvailableTab);
                } else {
                    const firstOverallTab = document.querySelector('.tab-button');
                    if (firstOverallTab && firstOverallTab !== landgenTabButtonElement) loadTab(firstOverallTab);
                }
            }
            if (canEnableLandgen && previousDisabledState === true) {
                if (!landgenTabButtonElement._hasClickListener) {
                    console.log("Shell: Landgen tab enabled, adding click listener.");
                    landgenTabButtonElement.addEventListener('click', handleLandgenTabClick);
                    landgenTabButtonElement._hasClickListener = true;
                }
            } else if (!canEnableLandgen && landgenTabButtonElement._hasClickListener) {
                console.log("Shell: Landgen tab disabled, removing click listener.");
                landgenTabButtonElement.removeEventListener('click', handleLandgenTabClick);
                landgenTabButtonElement._hasClickListener = false;
            }
        }
    }
    function handleLandgenTabClick() { loadTab(landgenTabButtonElement); }

    // --- Hamburger Menu Logic ---
    if (hamburgerBtn && parametersColumn && mainContentArea) {
        hamburgerBtn.addEventListener('click', () => {
            parametersColumn.classList.toggle('is-open');
            hamburgerBtn.classList.toggle('is-active');
            mainContentArea.classList.toggle('panel-is-open');
            parametersColumn.addEventListener('transitionend', function onTransitionEnd() {
                if (commonViewerAPI && typeof commonViewerAPI.onWindowResize === 'function') {
                    commonViewerAPI.onWindowResize();
                }
                parametersColumn.removeEventListener('transitionend', onTransitionEnd);
            }, { once: true });
            setTimeout(() => {
                if (commonViewerAPI && typeof commonViewerAPI.onWindowResize === 'function') {
                    commonViewerAPI.onWindowResize();
                }
            }, 350);
        });
    }

    // --- Fullscreen Toggle Logic ---
    function enterFullscreen() {
        if (!viewerContainer || !toggleFullscreenBtn) return;
        document.body.classList.add('viewer-fullscreen-active');
        viewerContainer.classList.add('viewer-fullscreen');
        const icon = toggleFullscreenBtn.querySelector('i');
        if (icon) { icon.classList.remove('fa-expand'); icon.classList.add('fa-compress'); }
        toggleFullscreenBtn.title = "Exit Fullscreen";
        viewerContainer.addEventListener('transitionend', function onFsEnter() {
            if (commonViewerAPI && typeof commonViewerAPI.onWindowResize === 'function') commonViewerAPI.onWindowResize();
            viewerContainer.removeEventListener('transitionend', onFsEnter);
        }, { once: true });
        setTimeout(() => { if (commonViewerAPI && typeof commonViewerAPI.onWindowResize === 'function') commonViewerAPI.onWindowResize(); }, 50);
    }
    function exitFullscreen() {
        if (!viewerContainer || !toggleFullscreenBtn) return;
        document.body.classList.remove('viewer-fullscreen-active');
        viewerContainer.classList.remove('viewer-fullscreen');
        const icon = toggleFullscreenBtn.querySelector('i');
        if (icon) { icon.classList.remove('fa-compress'); icon.classList.add('fa-expand'); }
        toggleFullscreenBtn.title = "Toggle Fullscreen";
        viewerContainer.addEventListener('transitionend', function onFsExit() {
            if (commonViewerAPI && typeof commonViewerAPI.onWindowResize === 'function') commonViewerAPI.onWindowResize();
            viewerContainer.removeEventListener('transitionend', onFsExit);
        }, { once: true });
        setTimeout(() => { if (commonViewerAPI && typeof commonViewerAPI.onWindowResize === 'function') commonViewerAPI.onWindowResize(); }, 350);
    }
    if (toggleFullscreenBtn && viewerContainer) {
        toggleFullscreenBtn.addEventListener('click', (event) => {
            event.preventDefault();
            if (viewerContainer.classList.contains('viewer-fullscreen')) exitFullscreen(); else enterFullscreen();
        });
    }
    document.addEventListener('keydown', (event) => {
        const vc = document.getElementById('viewerContainer');
        if (event.key === "Escape" && vc && vc.classList.contains('viewer-fullscreen')) {
            exitFullscreen();
        }
    });

    // --- Fine Mesh Rotation Event Handlers ---
    function onMeshRotateMouseDown(event) {
        if (event.button === 2) {
            isRightMouseDownForMeshRotation = true;
            if (controls) controls.enabled = false;
            if (renderer && renderer.domElement) {
                const rect = renderer.domElement.getBoundingClientRect();
                previousMousePositionForMeshRotation = { x: event.clientX - rect.left, y: event.clientY - rect.top };
            } else {
                previousMousePositionForMeshRotation = { x: event.clientX, y: event.clientY };
            }
        }
    }
    function onMeshRotateMouseMove(event) {
        if (!isRightMouseDownForMeshRotation || !renderer || !renderer.domElement) return;
        const rect = renderer.domElement.getBoundingClientRect();
        const currentMouseX = event.clientX - rect.left;
        const currentMouseY = event.clientY - rect.top;
        const deltaMove = { x: currentMouseX - previousMousePositionForMeshRotation.x, y: currentMouseY - previousMousePositionForMeshRotation.y };
        let meshToRotate = null, outlineToRotate = null;

        if (currentVisibleMeshType === 'icosphere' && icosphereDisplayMesh) meshToRotate = icosphereDisplayMesh;
        else if (currentVisibleMeshType === 'voronoi' && voronoiDisplayData) {
            if (voronoiDisplayData.solidMesh) meshToRotate = voronoiDisplayData.solidMesh;
            if (voronoiDisplayData.outlineMesh) outlineToRotate = voronoiDisplayData.outlineMesh;
        } else if (currentVisibleMeshType === 'land' && landDisplayMesh) meshToRotate = landDisplayMesh;

        if (meshToRotate) {
            let effectiveSensitivity = MESH_ROTATION_SENSITIVITY;
            if (camera && controls) {
                const currentDistance = controls.getDistance();
                const currentFov = camera.fov;
                const distanceRatio = currentDistance / REFERENCE_ROTATION_DISTANCE;
                const fovRatio = currentFov / REFERENCE_ROTATION_FOV;
                effectiveSensitivity = MESH_ROTATION_SENSITIVITY * (distanceRatio * fovRatio);
            }
            const yawAngle = deltaMove.x * effectiveSensitivity;
            const pitchAngle = deltaMove.y * effectiveSensitivity;
            const yawQuaternion = new THREE.Quaternion().setFromAxisAngle(new THREE.Vector3(0, 1, 0), yawAngle);
            const pitchQuaternion = new THREE.Quaternion().setFromAxisAngle(new THREE.Vector3(1, 0, 0), pitchAngle);
            meshToRotate.quaternion.premultiply(yawQuaternion);
            meshToRotate.quaternion.multiply(pitchQuaternion);
            if (outlineToRotate && outlineToRotate !== meshToRotate) {
                outlineToRotate.quaternion.premultiply(yawQuaternion);
                outlineToRotate.quaternion.multiply(pitchQuaternion);
            }
        }
        previousMousePositionForMeshRotation = { x: currentMouseX, y: currentMouseY };
    }
    function onMeshRotateMouseUp(event) {
        if (event.button === 2 || isRightMouseDownForMeshRotation) {
            isRightMouseDownForMeshRotation = false;
            if (controls) controls.enabled = true;
        }
    }

    // --- Common Three.js Setup ---
    function initCommonThreeJS() {
        if (renderer && renderer.domElement.parentElement === viewerContainer) {
            commonViewerAPI.clearScene();
            console.log("Shell: Common Three.js scene cleared for re-use.");
            if (camera && typeof camera.maxFov === 'number') {
                camera.fov = camera.maxFov;
                camera.updateProjectionMatrix();
            }
            if (!controls && camera && renderer) {
                controls = new OrbitControls(camera, renderer.domElement);
                controls.enableDamping = true; controls.dampingFactor = 0.05;
                controls.minDistance = 1.1; controls.maxDistance = 20;
                if (viewerAutoRotateCheckbox) controls.autoRotate = viewerAutoRotateCheckbox.checked; else controls.autoRotate = false;
                controls.autoRotateSpeed = 0.3;
            }
            return;
        }
        if (renderer) {
            if (renderer.domElement && renderer.domElement._wheelListener) {
                renderer.domElement.removeEventListener('wheel', renderer.domElement._wheelListener);
                delete renderer.domElement._wheelListener;
            }
            if (renderer.domElement) {
                renderer.domElement.removeEventListener('mousedown', onMeshRotateMouseDown, false);
                renderer.domElement.removeEventListener('mousemove', onMeshRotateMouseMove, false);
                renderer.domElement.removeEventListener('mouseup', onMeshRotateMouseUp, false);
                renderer.domElement.removeEventListener('mouseout', onMeshRotateMouseUp, false);
                renderer.domElement.removeEventListener('contextmenu', preventContextMenu, false);
            }
            renderer.dispose();
            if (commonAnimationFrameId) cancelAnimationFrame(commonAnimationFrameId);
            commonAnimationFrameId = null;
        }
        if (!viewerContainer) { console.error("Shell: Cannot init ThreeJS: viewerContainer not found!"); return; }
        viewerContainer.innerHTML = '';
        scene = new THREE.Scene();
        scene.background = new THREE.Color(0x111827);
        const aspect = (viewerContainer.clientWidth && viewerContainer.clientHeight) ? (viewerContainer.clientWidth / viewerContainer.clientHeight) : 1;
        camera = new THREE.PerspectiveCamera(DEFAULT_CAMERA_FOV, aspect, 0.1, 1000);
        camera.position.set(0, 0, 3.0);
        camera.minFov = MIN_CAMERA_FOV; camera.maxFov = DEFAULT_CAMERA_FOV; camera.fov = DEFAULT_CAMERA_FOV;
        camera.updateProjectionMatrix();
        try {
            renderer = new THREE.WebGLRenderer({ antialias: true });
            if (!renderer) console.error("Shell: THREE.WebGLRenderer returned null!");
        } catch (e) { console.error("Shell: Error during WebGLRenderer instantiation:", e); if (commonViewerAPI && commonViewerAPI.showStatus) commonViewerAPI.showStatus("Critical Error: Could not initialize 3D renderer.", "error"); return; }

        if (renderer && viewerContainer) {
            renderer.setSize(viewerContainer.clientWidth, viewerContainer.clientHeight);
            viewerContainer.appendChild(renderer.domElement);
            const wheelListener = function (event) {
                if (!controls || !camera) return;
                const currentDistance = controls.getDistance();
                if (event.deltaY < 0) {
                    if (currentDistance <= controls.minDistance + 0.01 && camera.fov > camera.minFov) {
                        event.preventDefault(); event.stopPropagation();
                        controls.enableZoom = false;
                        camera.fov = Math.max(camera.minFov, camera.fov * FOV_ZOOM_IN_FACTOR);
                        camera.updateProjectionMatrix();
                    } else {
                        if (!controls.enableZoom && camera.fov >= camera.maxFov) controls.enableZoom = true;
                    }
                } else if (event.deltaY > 0) {
                    if (camera.fov < camera.maxFov) {
                        event.preventDefault(); event.stopPropagation();
                        controls.enableZoom = false;
                        const newFovCalc = camera.fov * FOV_ZOOM_OUT_FACTOR;
                        camera.fov = Math.min(camera.maxFov, newFovCalc);
                        camera.updateProjectionMatrix();
                    } else {
                        if (!controls.enableZoom) controls.enableZoom = true;
                    }
                }
            };
            renderer.domElement.addEventListener('wheel', wheelListener, { passive: false });
            renderer.domElement._wheelListener = wheelListener;
            renderer.domElement.addEventListener('mousedown', onMeshRotateMouseDown, false);
            renderer.domElement.addEventListener('mousemove', onMeshRotateMouseMove, false);
            renderer.domElement.addEventListener('mouseup', onMeshRotateMouseUp, false);
            renderer.domElement.addEventListener('mouseout', onMeshRotateMouseUp, false);
            renderer.domElement.addEventListener('contextmenu', preventContextMenu, false);
        } else { console.error("Shell: Renderer not initialized or viewerContainer not found."); }

        const ambientLight = new THREE.AmbientLight(0xffffff, 0.6); scene.add(ambientLight);
        const dirLight1 = new THREE.DirectionalLight(0xffffff, 0.8); dirLight1.position.set(5, 10, 7.5); scene.add(dirLight1);
        const dirLight2 = new THREE.DirectionalLight(0xffffff, 0.4); dirLight2.position.set(-5, -5, -7.5); scene.add(dirLight2);
        controls = new OrbitControls(camera, renderer.domElement);
        if (controls) { controls.enableDamping = true; controls.dampingFactor = 0.05; controls.minDistance = 1.1; controls.maxDistance = 20; if (viewerAutoRotateCheckbox) controls.autoRotate = viewerAutoRotateCheckbox.checked; else controls.autoRotate = false; controls.autoRotateSpeed = 0.3; }
        window.addEventListener('resize', commonViewerAPI.onWindowResize, false);
        animateCommonThreeJS();
        console.log("Shell: initCommonThreeJS fully restored.");
    }

    function preventContextMenu(event) { event.preventDefault(); }
    function animateCommonThreeJS() {
        if (!renderer) { if (commonAnimationFrameId) { cancelAnimationFrame(commonAnimationFrameId); commonAnimationFrameId = null; } return; }
        commonAnimationFrameId = requestAnimationFrame(animateCommonThreeJS);
        if (controls && (controls.autoRotate || controls.enableDamping)) controls.update();
        renderer.render(scene, camera);
    }

    // --- API for Tab Handlers ---
    const commonViewerAPI = {
        showStatus: (message, type = 'info') => {
            if (!statusMessageDiv) { console.warn("Shell API: statusMessageDiv not found."); return; }
            statusMessageDiv.textContent = message;
            statusMessageDiv.className = 'mb-3 p-3 text-sm rounded-lg';
            if (type === 'success') statusMessageDiv.classList.add('bg-green-100', 'border-green-200', 'text-green-700');
            else if (type === 'error') statusMessageDiv.classList.add('bg-red-100', 'border-red-200', 'text-red-700');
            else statusMessageDiv.classList.add('bg-blue-50', 'border-blue-200', 'text-blue-700');
            statusMessageDiv.classList.remove('hidden');
        },
        clearScene: () => {
            console.log("Shell API: clearScene called.");
            if (!scene) { console.log("Shell API: Scene not initialized for clearing."); return; }

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
                voronoiDisplayData = null;
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
            activeIcosphereData = null;
            activeVoronoiData = null;
            activeBaseMeshSubdivision = -1;
            currentElevationDataSubdivision = -1;

            updateLandgenTabState();
            updateShowElevationButtonState();
            if (showLandBtn) showLandBtn.classList.add('hidden');
        },
        setIcosphereMesh: (mesh) => { icosphereDisplayMesh = mesh; },
        setVoronoiData: (data) => {
            voronoiDisplayData = data;
            if (voronoiDisplayData && voronoiDisplayData.solidMesh && voronoiDisplayData.solidMesh.material && !defaultVoronoiUniforms) {
                const uniforms = voronoiDisplayData.solidMesh.material.uniforms;
                if (uniforms && uniforms.u_useElevationColoring && uniforms.u_elevationDataTexture && uniforms.u_elevationTextureDim && uniforms.u_minElevation && uniforms.u_maxElevation) {
                    defaultVoronoiUniforms = {
                        u_useElevationColoring: { value: uniforms.u_useElevationColoring.value },
                        u_elevationDataTexture: { value: uniforms.u_elevationDataTexture.value },
                        u_elevationTextureDim: { value: new THREE.Vector2().copy(uniforms.u_elevationTextureDim.value) },
                        u_minElevation: { value: uniforms.u_minElevation.value },
                        u_maxElevation: { value: uniforms.u_maxElevation.value },
                    };
                    console.log("Shell API: Default Voronoi uniforms captured.");
                } else {
                    console.warn("Shell API: Voronoi shader uniforms not fully available to capture defaults at setVoronoiData time.");
                }
            }
            updateShowElevationButtonState();
        },
        setLandMesh: (mesh) => {
            landDisplayMesh = mesh;
            if (landDisplayMesh && showLandBtn) showLandBtn.classList.remove('hidden');
            else if (showLandBtn) showLandBtn.classList.add('hidden');
        },
        setActiveMeshData: (icoData, voroData, subdivisionLevel) => {
            console.log(`Shell API: setActiveMeshData called for subdivision: ${subdivisionLevel}`);
            activeIcosphereData = icoData;
            activeVoronoiData = voroData;
            activeBaseMeshSubdivision = subdivisionLevel !== undefined ? subdivisionLevel : -1;

            if (currentElevationDataSubdivision !== -1 && currentElevationDataSubdivision !== activeBaseMeshSubdivision) {
                console.log("Shell API: Base mesh changed, resetting current elevation data subdivision and Voronoi coloring.");
                currentElevationDataSubdivision = -1;
                if (voronoiDisplayData && voronoiDisplayData.solidMesh && voronoiDisplayData.solidMesh.material.uniforms && defaultVoronoiUniforms) {
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
            }
            updateLandgenTabState();
            updateShowElevationButtonState();
        },
        getActiveMeshData: () => {
            return {
                icosphereData: activeIcosphereData,
                voronoiData: activeVoronoiData,
                subdivisionLevel: activeBaseMeshSubdivision
            };
        },
        updateVoronoiElevationVisuals: (elevationMap, minElev, maxElev, texWidth, texHeight, baseSubdivisionLevel) => {
            if (!voronoiDisplayData || !voronoiDisplayData.solidMesh || !voronoiDisplayData.solidMesh.material || !voronoiDisplayData.solidMesh.material.uniforms) {
                console.warn("Shell API: Voronoi mesh or material not ready for elevation update.");
                commonViewerAPI.showStatus("Voronoi mesh not ready for elevation display.", "warning");
                return;
            }
            console.log(`Shell API: Updating Voronoi elevation for sub ${baseSubdivisionLevel}. Min: ${minElev}, Max: ${maxElev}, TexDims: ${texWidth}x${texHeight}`);
            const uniforms = voronoiDisplayData.solidMesh.material.uniforms;

            const totalCellsInTexture = texWidth * texHeight;
            const textureData = new Float32Array(totalCellsInTexture);
            const numIcosphereSites = activeIcosphereData ? activeIcosphereData.vertices.length / 3 : 0;

            for (let i = 0; i < totalCellsInTexture; i++) {
                const cellId = i;
                if (cellId < numIcosphereSites) {
                    textureData[i] = elevationMap.has(cellId) ? elevationMap.get(cellId) : minElev;
                } else {
                    textureData[i] = minElev;
                }
            }

            if (uniforms.u_elevationDataTexture.value && uniforms.u_elevationDataTexture.value !== (defaultVoronoiUniforms ? defaultVoronoiUniforms.u_elevationDataTexture.value : null)) {
                uniforms.u_elevationDataTexture.value.dispose();
            }

            const elevationTexture = new THREE.DataTexture(textureData, texWidth, texHeight, THREE.RedFormat, THREE.FloatType);
            elevationTexture.needsUpdate = true;

            uniforms.u_elevationDataTexture.value = elevationTexture;
            uniforms.u_elevationTextureDim.value.set(parseFloat(texWidth), parseFloat(texHeight));
            uniforms.u_minElevation.value = parseFloat(minElev);
            uniforms.u_maxElevation.value = parseFloat(maxElev);
            uniforms.u_useElevationColoring.value = true;

            voronoiDisplayData.solidMesh.material.needsUpdate = true;

            currentElevationDataSubdivision = baseSubdivisionLevel;
            updateShowElevationButtonState();

            commonViewerAPI.showStatus("Elevation map applied to Voronoi cells.", "success");
            if (currentVisibleMeshType !== 'voronoi' || (showElevationBtnElement && !showElevationBtnElement.classList.contains('btn-active'))) {
                commonViewerAPI.displayMesh('voronoi');
            }
        },
        displayMesh: (meshType) => {
            if (!scene) { initCommonThreeJS(); }
            if (!scene) { console.error("Shell displayMesh: Scene still not initialized!"); return; }
            console.log(`Shell displayMesh: Attempting to display mesh type: '${meshType}'`);

            if (icosphereDisplayMesh) scene.remove(icosphereDisplayMesh);
            if (voronoiDisplayData) {
                if (voronoiDisplayData.solidMesh) scene.remove(voronoiDisplayData.solidMesh);
                if (voronoiDisplayData.outlineMesh) scene.remove(voronoiDisplayData.outlineMesh);
            }
            if (landDisplayMesh) scene.remove(landDisplayMesh);

            currentVisibleMeshType = meshType;

            if (showIcosphereBtn) showIcosphereBtn.classList.remove('btn-active');
            if (showVoronoiBtn) showVoronoiBtn.classList.remove('btn-active');
            if (showElevationBtnElement) showElevationBtnElement.classList.remove('btn-active');
            if (showLandBtn) showLandBtn.classList.remove('btn-active');

            if (meshType === 'icosphere' && icosphereDisplayMesh) {
                if (!icosphereDisplayMesh.material || !(icosphereDisplayMesh.material instanceof THREE.MeshStandardMaterial)) {
                    if (icosphereDisplayMesh.material) icosphereDisplayMesh.material.dispose();
                    icosphereDisplayMesh.material = new THREE.MeshStandardMaterial({ color: 0x00dd00, side: THREE.DoubleSide, metalness: 0.1, roughness: 0.6 });
                }
                icosphereDisplayMesh.material.wireframe = viewerWireframeCheckbox ? viewerWireframeCheckbox.checked : true;
                scene.add(icosphereDisplayMesh);
                if (showIcosphereBtn) showIcosphereBtn.classList.add('btn-active');
            } else if (meshType === 'voronoi' && voronoiDisplayData) {
                if (voronoiDisplayData.solidMesh) {
                    scene.add(voronoiDisplayData.solidMesh);
                    if (voronoiDisplayData.solidMesh.material && voronoiDisplayData.solidMesh.material.uniforms) {
                        voronoiDisplayData.solidMesh.material.uniforms.u_edgeFeather.value = (viewerWireframeCheckbox && viewerWireframeCheckbox.checked) ? 1.5 : -1.0;
                        const useElevColor = voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring.value;
                        const elevDataValid = currentElevationDataSubdivision === activeBaseMeshSubdivision && currentElevationDataSubdivision !== -1;

                        if (useElevColor && elevDataValid) {
                            if (showElevationBtnElement) showElevationBtnElement.classList.add('btn-active');
                        } else {
                            voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring.value = false;
                            if (showVoronoiBtn) showVoronoiBtn.classList.add('btn-active');
                        }
                    } else {
                        if (showVoronoiBtn) showVoronoiBtn.classList.add('btn-active');
                    }
                }
                if (voronoiDisplayData.outlineMesh) {
                    voronoiDisplayData.outlineMesh.visible = viewerWireframeCheckbox ? viewerWireframeCheckbox.checked : true;
                    scene.add(voronoiDisplayData.outlineMesh);
                }
            } else if (meshType === 'land' && landDisplayMesh) {
                scene.add(landDisplayMesh);
                if (landDisplayMesh.material && landDisplayMesh.material.hasOwnProperty('wireframe')) {
                    landDisplayMesh.material.wireframe = viewerWireframeCheckbox ? viewerWireframeCheckbox.checked : false;
                }
                if (showLandBtn) showLandBtn.classList.add('btn-active');
            } else {
                console.warn(`Shell displayMesh: Mesh type "${meshType}" or its data not available.`);
            }
        },
        getViewerControlsState: () => ({ wireframe: viewerWireframeCheckbox ? viewerWireframeCheckbox.checked : true }),
        onWindowResize: () => {
            if (!camera || !renderer || !viewerContainer) return;
            const width = viewerContainer.clientWidth; const height = viewerContainer.clientHeight;
            if (width > 0 && height > 0) {
                camera.aspect = width / height;
                camera.updateProjectionMatrix();
                renderer.setSize(width, height);
            }
        },
        getCurrentMeshType: () => currentVisibleMeshType,
    };
    window.worldgenViewerAPI = commonViewerAPI;

    // --- Tab Loading and Switching Logic ---
    async function loadTab(tabButton) {
        if (!tabButton) return;
        const tabId = tabButton.getAttribute('data-tab-id');
        const contentUrl = tabButton.getAttribute('data-tab-content-url');
        const scriptUrl = tabButton.getAttribute('data-tab-js');
        currentActiveTabId = tabId;
        tabButtons.forEach(btn => btn.classList.remove('active-tab'));
        tabButton.classList.add('active-tab');
        if (parametersTitle) parametersTitle.textContent = `${tabButton.textContent.trim()} Parameters`;
        if (parametersFormContainer) parametersFormContainer.innerHTML = '<p class="text-gray-500">Loading parameters...</p>';

        if (currentTabHandlerAPI && typeof currentTabHandlerAPI.dispose === 'function') {
            console.log(`Shell: Disposing handler for previous tab: ${currentTabHandlerAPI.tabId || 'unknown'}`);
            currentTabHandlerAPI.dispose();
        }
        currentTabHandlerAPI = null;

        if (contentUrl) {
            try {
                const response = await fetch(contentUrl);
                if (!response.ok) throw new Error(`Failed to load tab content: ${response.statusText} for ${contentUrl}`);
                const htmlFragment = await response.text();
                if (parametersFormContainer) { parametersFormContainer.innerHTML = htmlFragment; }
                else { console.error("parametersFormContainer not found to inject HTML fragment"); }
                if (scriptUrl) {
                    const oldScript = document.getElementById(`${tabId}-handler-script`);
                    if (oldScript) oldScript.remove();
                    const scriptElement = document.createElement('script');
                    scriptElement.id = `${tabId}-handler-script`;
                    scriptElement.src = scriptUrl;
                    scriptElement.type = 'module';
                    scriptElement.onload = () => {
                        const handlerName = `initialize${tabId.charAt(0).toUpperCase() + tabId.slice(1)}Tab`;
                        if (window[handlerName] && typeof window[handlerName] === 'function') {
                            currentTabHandlerAPI = window[handlerName](commonViewerAPI);
                            if (currentTabHandlerAPI) currentTabHandlerAPI.tabId = tabId;
                            console.log(`Shell: Initialized tab handler for ${tabId}`);
                            if (currentTabHandlerAPI && typeof currentTabHandlerAPI.updateGenerateButton === 'function') {
                                currentTabHandlerAPI.updateGenerateButton();
                            } else if (mainGenerateBtn) {
                                mainGenerateBtn.textContent = 'Generate';
                                mainGenerateBtn.classList.remove('btn-secondary');
                                mainGenerateBtn.classList.add('btn-primary');
                                mainGenerateBtn.disabled = (tabId === 'landgen' && landgenTabButtonElement && landgenTabButtonElement.disabled);
                            }
                        } else { console.warn(`Shell: Init function ${handlerName} not found for tab ${tabId}`); }
                    };
                    scriptElement.onerror = () => { console.error(`Shell: Failed to load script for tab ${tabId}: ${scriptUrl}`); if (parametersFormContainer) parametersFormContainer.innerHTML = `<p class="text-red-500">Error loading script for ${tabId}.</p>`; };
                    document.body.appendChild(scriptElement);
                }
            } catch (error) { console.error("Shell: Error loading tab:", error); if (parametersFormContainer) parametersFormContainer.innerHTML = `<p class="text-red-500">Error loading parameters for ${tabId}.</p>`; if (commonViewerAPI && commonViewerAPI.showStatus) commonViewerAPI.showStatus(`Error loading tab: ${error.message}`, 'error'); }
        }
    }

    tabButtons.forEach(button => {
        if (button.id !== 'landgenTabButton') {
            button.addEventListener('click', () => loadTab(button));
        }
    });
    if (landgenTabButtonElement) {
        if (!landgenTabButtonElement.disabled && !landgenTabButtonElement._hasClickListener) {
            landgenTabButtonElement.addEventListener('click', handleLandgenTabClick);
            landgenTabButtonElement._hasClickListener = true;
        }
        const landgenTabObserver = new MutationObserver(mutations => {
            mutations.forEach(mutation => {
                if (mutation.type === 'attributes' && mutation.attributeName === 'disabled') {
                    if (!landgenTabButtonElement.disabled && !landgenTabButtonElement._hasClickListener) {
                        console.log("Shell (Observer): Landgen tab enabled, adding click listener.");
                        landgenTabButtonElement.addEventListener('click', handleLandgenTabClick);
                        landgenTabButtonElement._hasClickListener = true;
                    } else if (landgenTabButtonElement.disabled && landgenTabButtonElement._hasClickListener) {
                        console.log("Shell (Observer): Landgen tab disabled, removing click listener.");
                        landgenTabButtonElement.removeEventListener('click', handleLandgenTabClick);
                        landgenTabButtonElement._hasClickListener = false;
                    }
                }
            });
        });
        landgenTabObserver.observe(landgenTabButtonElement, { attributes: true });
    }

    // --- Main Generate Button Listener ---
    if (mainGenerateBtn) {
        mainGenerateBtn.addEventListener('click', async (event) => {
            event.preventDefault();
            console.log("Shell: Main Generate/Load/Process Button clicked.");
            if (currentTabHandlerAPI && typeof currentTabHandlerAPI.generate === 'function') {
                console.log("Shell: Delegating action to active tab handler.");
                if (!scene) initCommonThreeJS();
                currentTabHandlerAPI.generate();
            } else {
                if (commonViewerAPI && commonViewerAPI.showStatus) commonViewerAPI.showStatus("No active action logic for this tab.", "error");
                console.warn("Shell: mainGenerateBtn clicked, but no active tab handler or generate function.");
            }
        });
    }

    // --- Initialization ---
    initCommonThreeJS();
    updateLandgenTabState();
    updateShowElevationButtonState();
    const initialActiveTab = document.querySelector('.tab-button.active-tab');
    if (initialActiveTab) {
        if (initialActiveTab.id === 'landgenTabButton' && initialActiveTab.disabled) {
            const firstAvailableTab = document.querySelector('.tab-button:not([disabled]):not(#landgenTabButton)');
            if (firstAvailableTab) loadTab(firstAvailableTab);
            else if (tabButtons.length > 0 && tabButtons[0] !== landgenTabButtonElement) loadTab(tabButtons[0]);
            else console.warn("Shell: No suitable tab to switch to from initially disabled landgen tab.");
        } else {
            loadTab(initialActiveTab);
        }
    } else if (tabButtons.length > 0) {
        const firstAvailableTab = document.querySelector('.tab-button:not([disabled])');
        if (firstAvailableTab) loadTab(firstAvailableTab);
        else if (tabButtons[0]) loadTab(tabButtons[0]);
    }

    // --- Viewer Control Listeners ---
    if (showIcosphereBtn) showIcosphereBtn.addEventListener('click', () => commonViewerAPI.displayMesh('icosphere'));
    if (showVoronoiBtn) {
        showVoronoiBtn.addEventListener('click', () => {
            if (voronoiDisplayData && voronoiDisplayData.solidMesh && voronoiDisplayData.solidMesh.material.uniforms &&
                voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring) {
                voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring.value = false;
            }
            commonViewerAPI.displayMesh('voronoi');
        });
    }
    if (showElevationBtnElement) {
        showElevationBtnElement.addEventListener('click', () => {
            if (showElevationBtnElement.disabled) return;
            if (voronoiDisplayData && voronoiDisplayData.solidMesh && voronoiDisplayData.solidMesh.material.uniforms &&
                currentElevationDataSubdivision === activeBaseMeshSubdivision && currentElevationDataSubdivision !== -1) {
                voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring.value = true;
                commonViewerAPI.displayMesh('voronoi');
            } else {
                commonViewerAPI.showStatus("Elevation data not available or not applicable to current Voronoi mesh.", "warning");
            }
        });
    }
    if (showLandBtn) showLandBtn.addEventListener('click', () => commonViewerAPI.displayMesh('land'));

    if (viewerWireframeCheckbox) {
        viewerWireframeCheckbox.addEventListener('change', () => {
            const isChecked = viewerWireframeCheckbox.checked;
            if (currentVisibleMeshType === 'icosphere' && icosphereDisplayMesh && icosphereDisplayMesh.material) {
                icosphereDisplayMesh.material.wireframe = isChecked;
            } else if (currentVisibleMeshType === 'voronoi' && voronoiDisplayData) {
                if (voronoiDisplayData.solidMesh && voronoiDisplayData.solidMesh.material && voronoiDisplayData.solidMesh.material.uniforms) {
                    voronoiDisplayData.solidMesh.material.uniforms.u_edgeFeather.value = isChecked ? 1.5 : -1.0;
                }
                if (voronoiDisplayData.outlineMesh) {
                    voronoiDisplayData.outlineMesh.visible = isChecked;
                }
            } else if (currentVisibleMeshType === 'land' && landDisplayMesh && landDisplayMesh.material) {
                if (landDisplayMesh.material.hasOwnProperty('wireframe')) {
                    landDisplayMesh.material.wireframe = isChecked;
                }
            }
        });
    }
    if (viewerAutoRotateCheckbox) {
        viewerAutoRotateCheckbox.addEventListener('change', () => {
            if (controls) controls.autoRotate = viewerAutoRotateCheckbox.checked;
        });
    }
});
