// static/js/three_viewer.js
import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

// Constants from app_shell.js that are relevant here
const DEFAULT_CAMERA_FOV = 60;
const MIN_CAMERA_FOV = 5;
const FOV_ZOOM_IN_FACTOR = 0.9;
const FOV_ZOOM_OUT_FACTOR = 1 / FOV_ZOOM_IN_FACTOR;

let scene, camera, renderer, controls;
let animationFrameId = null;
let viewerContainerElement = null; // To store the container

/**
 * Initializes the Three.js viewer.
 * @param {HTMLElement} containerElement - The DOM element to append the canvas to.
 * @returns {object|null} An API object for interacting with the viewer, or null if initialization fails.
 */
function initializeViewer(containerElement) {
    if (!containerElement) {
        console.error("ThreeViewer: Viewer container element is required for initialization.");
        return null;
    }
    viewerContainerElement = containerElement;
    viewerContainerElement.innerHTML = ''; // Clear previous content

    // Scene
    scene = new THREE.Scene();
    scene.background = new THREE.Color(0x111827); // Dark background from app_shell

    // Camera
    const aspect = (viewerContainerElement.clientWidth && viewerContainerElement.clientHeight) ?
        (viewerContainerElement.clientWidth / viewerContainerElement.clientHeight) : 1;
    camera = new THREE.PerspectiveCamera(DEFAULT_CAMERA_FOV, aspect, 0.1, 1000);
    camera.position.set(0, 0, 3.0); // Initial camera distance from app_shell
    camera.minFov = MIN_CAMERA_FOV;   // Store min FOV on camera object
    camera.maxFov = DEFAULT_CAMERA_FOV; // Store max/default FOV on camera object
    camera.fov = DEFAULT_CAMERA_FOV;    // Set current FOV
    camera.updateProjectionMatrix();

    // Renderer
    try {
        renderer = new THREE.WebGLRenderer({ antialias: true });
        if (!renderer) {
            console.error("ThreeViewer: THREE.WebGLRenderer returned null or undefined!");
            return null;
        }
    } catch (e) {
        console.error("ThreeViewer: Error during THREE.WebGLRenderer instantiation:", e);
        return null;
    }
    renderer.setSize(viewerContainerElement.clientWidth, viewerContainerElement.clientHeight);
    viewerContainerElement.appendChild(renderer.domElement);

    // Lighting (from app_shell)
    const ambientLight = new THREE.AmbientLight(0xffffff, 0.6);
    scene.add(ambientLight);
    const dirLight1 = new THREE.DirectionalLight(0xffffff, 0.8);
    dirLight1.position.set(5, 10, 7.5);
    scene.add(dirLight1);
    const dirLight2 = new THREE.DirectionalLight(0xffffff, 0.4);
    dirLight2.position.set(-5, -5, -7.5);
    scene.add(dirLight2);

    // OrbitControls (from app_shell)
    controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.dampingFactor = 0.05;
    controls.minDistance = 1.1;
    controls.maxDistance = 20; // Max distance for OrbitControls zoom
    // Auto-rotate will be controlled by the main shell via a method if needed
    // For now, initialize it to false as per app_shell's default for viewerAutoRotateCheckbox
    controls.autoRotate = false;
    controls.autoRotateSpeed = 0.3;

    // Custom wheel listener for FOV zoom (from app_shell)
    const wheelListener = function (event) {
        if (!controls || !camera) return;
        const currentDistance = controls.getDistance();

        if (event.deltaY < 0) { // Zoom in (scroll up, deltaY negative)
            if (currentDistance <= controls.minDistance + 0.01 && camera.fov > camera.minFov) {
                event.preventDefault(); event.stopPropagation();
                controls.enableZoom = false; // Temporarily disable OrbitControls zoom
                camera.fov = Math.max(camera.minFov, camera.fov * FOV_ZOOM_IN_FACTOR);
                camera.updateProjectionMatrix();
            } else {
                if (!controls.enableZoom && camera.fov >= camera.maxFov) controls.enableZoom = true;
            }
        } else if (event.deltaY > 0) { // Zoom out (scroll down, deltaY positive)
            if (camera.fov < camera.maxFov) {
                event.preventDefault(); event.stopPropagation();
                controls.enableZoom = false; // Temporarily disable OrbitControls zoom
                const newFovCalc = camera.fov * FOV_ZOOM_OUT_FACTOR;
                camera.fov = Math.min(camera.maxFov, newFovCalc);
                camera.updateProjectionMatrix();
            } else {
                if (!controls.enableZoom) controls.enableZoom = true;
            }
        }
    };
    renderer.domElement.addEventListener('wheel', wheelListener, { passive: false });
    renderer.domElement._wheelListener = wheelListener; // Store for potential removal

    startAnimation(); // Start the animation loop automatically

    console.log("ThreeViewer: Initialized successfully.");

    return {
        getScene: () => scene,
        getCamera: () => camera,
        getRenderer: () => renderer,
        getControls: () => controls,
        getDomElement: () => renderer.domElement,
        resize: (width, height) => {
            if (camera && renderer) {
                if (width > 0 && height > 0) {
                    camera.aspect = width / height;
                    camera.updateProjectionMatrix();
                    renderer.setSize(width, height);
                    console.log(`ThreeViewer: Resized to ${width}x${height}`);
                }
            }
        },
        startAnimation,
        stopAnimation,
        setAutoRotate: (enabled) => {
            if (controls) controls.autoRotate = enabled;
        },
        dispose: () => {
            console.log("ThreeViewer: Disposing viewer resources.");
            stopAnimation();
            if (renderer && renderer.domElement && renderer.domElement._wheelListener) {
                renderer.domElement.removeEventListener('wheel', renderer.domElement._wheelListener);
                delete renderer.domElement._wheelListener;
            }
            // Note: Mesh rotation listeners will be handled by the module that adds them.
            // This dispose focuses on what three_viewer.js itself sets up.
            if (controls) controls.dispose();
            if (renderer) renderer.dispose();

            // Clear scene objects (geometry, materials) - this should be done by the code managing those objects
            // For example, app_shell_main.js's clearScene would iterate through scene.children
            if (scene) {
                while (scene.children.length > 0) {
                    scene.remove(scene.children[0]);
                }
            }

            scene = null; camera = null; renderer = null; controls = null;
            viewerContainerElement = null;
        }
    };
}

function animate() {
    if (!renderer || !scene || !camera) { // Stop if core components are gone
        stopAnimation();
        return;
    }
    animationFrameId = requestAnimationFrame(animate);
    if (controls && (controls.autoRotate || controls.enableDamping)) {
        controls.update();
    }
    renderer.render(scene, camera);
}

function startAnimation() {
    if (animationFrameId === null && renderer && scene && camera) { // Only start if not already running and ready
        console.log("ThreeViewer: Starting animation loop.");
        animate();
    }
}

function stopAnimation() {
    if (animationFrameId !== null) {
        cancelAnimationFrame(animationFrameId);
        animationFrameId = null;
        console.log("ThreeViewer: Stopped animation loop.");
    }
}

export { initializeViewer };
