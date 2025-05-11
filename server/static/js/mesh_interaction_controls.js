// static/js/mesh_interaction_controls.js
import * as THREE from 'three'; // For THREE.Vector3 and THREE.Quaternion

// Constants for mesh rotation (previously in app_shell.js)
const MESH_ROTATION_SENSITIVITY = 0.001;
const REFERENCE_ROTATION_DISTANCE = 3.0;
const DEFAULT_CAMERA_FOV = 60; // Assuming this is the reference FOV
const REFERENCE_ROTATION_FOV = DEFAULT_CAMERA_FOV;


let isRightMouseDownForMeshRotation = false;
let previousMousePositionForMeshRotation = { x: 0, y: 0 };

let commonAPI = null; // To store commonViewerAPI
let rendererDomElementRef = null;
let orbitControlsRef = null;
let cameraRef = null;

/**
 * Handles mouse down for mesh rotation.
 * @param {MouseEvent} event 
 */
function onMeshRotateMouseDown(event) {
    if (event.button === 2) { // Right mouse button
        isRightMouseDownForMeshRotation = true;
        if (orbitControlsRef) {
            orbitControlsRef.enabled = false; // Disable OrbitControls during mesh rotation
        }

        if (rendererDomElementRef) {
            const rect = rendererDomElementRef.getBoundingClientRect();
            previousMousePositionForMeshRotation = {
                x: event.clientX - rect.left,
                y: event.clientY - rect.top
            };
        } else {
            // Fallback if rect isn't available (should not happen if initialized correctly)
            previousMousePositionForMeshRotation = { x: event.clientX, y: event.clientY };
        }
        event.preventDefault(); // Prevent context menu
    }
}

/**
 * Handles mouse move for mesh rotation.
 * @param {MouseEvent} event 
 */
function onMeshRotateMouseMove(event) {
    if (!isRightMouseDownForMeshRotation || !rendererDomElementRef) return;

    const rect = rendererDomElementRef.getBoundingClientRect();
    const currentMouseX = event.clientX - rect.left;
    const currentMouseY = event.clientY - rect.top;

    const deltaMove = {
        x: currentMouseX - previousMousePositionForMeshRotation.x,
        y: currentMouseY - previousMousePositionForMeshRotation.y
    };

    let meshToRotate = null;
    let outlineToRotate = null; // For Voronoi

    if (commonAPI) {
        const currentMeshType = commonAPI.getCurrentMeshType ? commonAPI.getCurrentMeshType() : null;
        if (currentMeshType === 'icosphere') {
            meshToRotate = commonAPI.getIcosphereDisplayMesh ? commonAPI.getIcosphereDisplayMesh() : null;
        } else if (currentMeshType === 'voronoi') {
            const voronoiData = commonAPI.getVoronoiDisplayData ? commonAPI.getVoronoiDisplayData() : null;
            if (voronoiData) {
                meshToRotate = voronoiData.solidMesh;
                outlineToRotate = voronoiData.outlineMesh;
            }
        } else if (currentMeshType === 'land') {
            meshToRotate = commonAPI.getLandDisplayMesh ? commonAPI.getLandDisplayMesh() : null;
        }
    }


    if (meshToRotate) {
        let effectiveSensitivity = MESH_ROTATION_SENSITIVITY;
        if (cameraRef && orbitControlsRef) {
            const currentDistance = orbitControlsRef.getDistance();
            const currentFov = cameraRef.fov;
            const distanceRatio = currentDistance / REFERENCE_ROTATION_DISTANCE;
            const fovRatio = currentFov / REFERENCE_ROTATION_FOV;
            let dynamicFactor = distanceRatio * fovRatio;
            effectiveSensitivity = MESH_ROTATION_SENSITIVITY * dynamicFactor;
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

    previousMousePositionForMeshRotation = {
        x: currentMouseX,
        y: currentMouseY
    };
}

/**
 * Handles mouse up/out for mesh rotation.
 * @param {MouseEvent} event 
 */
function onMeshRotateMouseUpOrOut(event) {
    // Check event.button for mouseup, or if it's a mouseout event during rotation
    if (isRightMouseDownForMeshRotation && (event.type === 'mouseup' && event.button === 2) || event.type === 'mouseout') {
        isRightMouseDownForMeshRotation = false;
        if (orbitControlsRef) {
            orbitControlsRef.enabled = true; // Re-enable OrbitControls
        }
    }
}

/**
 * Prevents the context menu on right-click.
 * @param {MouseEvent} event 
 */
function preventContextMenu(event) {
    event.preventDefault();
}

/**
 * Initializes mesh interaction controls.
 * @param {object} viewerAPI - The commonViewerAPI from app_shell_main.js
 */
function initializeMeshInteraction(viewerAPI) {
    if (!viewerAPI) {
        console.error("MeshInteractionControls: commonViewerAPI is required for initialization.");
        return;
    }
    commonAPI = viewerAPI; // Store for use in event handlers

    rendererDomElementRef = commonAPI.getRendererDomElement ? commonAPI.getRendererDomElement() : null;
    orbitControlsRef = commonAPI.getOrbitControls ? commonAPI.getOrbitControls() : null;
    cameraRef = commonAPI.getCamera ? commonAPI.getCamera() : null; // Assuming getCamera is added to commonAPI

    if (!rendererDomElementRef) {
        console.error("MeshInteractionControls: Renderer DOM element not available. Cannot attach interaction listeners.");
        return;
    }
    if (!orbitControlsRef) {
        console.warn("MeshInteractionControls: OrbitControls not available. Right-click drag might conflict.");
    }
    if (!cameraRef) {
        console.warn("MeshInteractionControls: Camera not available. Dynamic sensitivity might not work as expected.");
    }


    rendererDomElementRef.addEventListener('mousedown', onMeshRotateMouseDown, false);
    rendererDomElementRef.addEventListener('mousemove', onMeshRotateMouseMove, false);
    // Listen to mouseup on the window to catch drags that end outside the canvas
    window.addEventListener('mouseup', onMeshRotateMouseUpOrOut, false);
    rendererDomElementRef.addEventListener('mouseout', onMeshRotateMouseUpOrOut, false);
    rendererDomElementRef.addEventListener('contextmenu', preventContextMenu, false);

    console.log("MeshInteractionControls: Initialized.");

    // Return a dispose function for cleanup
    return {
        dispose: () => {
            console.log("MeshInteractionControls: Disposing interaction listeners.");
            if (rendererDomElementRef) {
                rendererDomElementRef.removeEventListener('mousedown', onMeshRotateMouseDown, false);
                rendererDomElementRef.removeEventListener('mousemove', onMeshRotateMouseMove, false);
                window.removeEventListener('mouseup', onMeshRotateMouseUpOrOut, false);
                rendererDomElementRef.removeEventListener('mouseout', onMeshRotateMouseUpOrOut, false);
                rendererDomElementRef.removeEventListener('contextmenu', preventContextMenu, false);
            }
            rendererDomElementRef = null;
            orbitControlsRef = null;
            cameraRef = null;
            commonAPI = null;
        }
    };
}

export { initializeMeshInteraction };
