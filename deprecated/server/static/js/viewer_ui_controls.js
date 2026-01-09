// static/js/viewer_ui_controls.js

/**
 * Initializes the viewer UI controls.
 * @param {object} commonViewerAPI - The common viewer API.
 * @param {object} elements - An object containing references to necessary DOM elements.
 * @param {HTMLElement} elements.showIcosphereBtn
 * @param {HTMLElement} elements.showVoronoiBtn
 * @param {HTMLElement} elements.showElevationBtnElement
 * @param {HTMLElement} elements.showLandBtn
 * @param {HTMLElement} elements.viewerWireframeCheckbox
 * @param {HTMLElement} elements.viewerAutoRotateCheckbox
 */
function initializeViewerUIControls(commonViewerAPI, elements) {
    const {
        showIcosphereBtn,
        showVoronoiBtn,
        showElevationBtnElement, // Using the passed reference
        showLandBtn,
        viewerWireframeCheckbox,
        viewerAutoRotateCheckbox
    } = elements;

    if (!commonViewerAPI) {
        console.error("ViewerUIControls: commonViewerAPI is required.");
        return;
    }

    if (showIcosphereBtn) {
        showIcosphereBtn.addEventListener('click', () => {
            if (typeof commonViewerAPI.displayMesh === 'function') {
                commonViewerAPI.displayMesh('icosphere');
            }
        });
    } else {
        console.warn("ViewerUIControls: showIcosphereBtn not provided.");
    }

    if (showVoronoiBtn) {
        showVoronoiBtn.addEventListener('click', () => {
            const voronoiDisplayData = commonViewerAPI.getVoronoiDisplayData ? commonViewerAPI.getVoronoiDisplayData() : null;
            if (voronoiDisplayData && voronoiDisplayData.solidMesh &&
                voronoiDisplayData.solidMesh.material &&
                voronoiDisplayData.solidMesh.material.uniforms &&
                voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring) {
                voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring.value = false;
            }
            if (typeof commonViewerAPI.displayMesh === 'function') {
                commonViewerAPI.displayMesh('voronoi');
            }
        });
    } else {
        console.warn("ViewerUIControls: showVoronoiBtn not provided.");
    }

    if (showElevationBtnElement) {
        showElevationBtnElement.addEventListener('click', () => {
            if (showElevationBtnElement.disabled) return;

            const voronoiDisplayData = commonViewerAPI.getVoronoiDisplayData ? commonViewerAPI.getVoronoiDisplayData() : null;
            const activeMeshData = commonViewerAPI.getActiveMeshData ? commonViewerAPI.getActiveMeshData() : {};
            const currentElevationDataSubdivision = commonViewerAPI.getCurrentElevationDataSubdivision ? commonViewerAPI.getCurrentElevationDataSubdivision() : -1;
            const activeBaseMeshSubdivision = activeMeshData.subdivisionLevel !== undefined ? activeMeshData.subdivisionLevel : -1;

            if (voronoiDisplayData && voronoiDisplayData.solidMesh &&
                voronoiDisplayData.solidMesh.material &&
                voronoiDisplayData.solidMesh.material.uniforms &&
                currentElevationDataSubdivision === activeBaseMeshSubdivision &&
                currentElevationDataSubdivision !== -1) {

                voronoiDisplayData.solidMesh.material.uniforms.u_useElevationColoring.value = true;
                if (typeof commonViewerAPI.displayMesh === 'function') {
                    commonViewerAPI.displayMesh('voronoi');
                }
            } else {
                if (typeof commonViewerAPI.showStatus === 'function') {
                    commonViewerAPI.showStatus("Elevation data not available or not applicable to current Voronoi mesh.", "warning");
                }
            }
        });
    } else {
        console.warn("ViewerUIControls: showElevationBtnElement not provided.");
    }


    if (showLandBtn) {
        showLandBtn.addEventListener('click', () => {
            if (typeof commonViewerAPI.displayMesh === 'function') {
                commonViewerAPI.displayMesh('land');
            }
        });
    } else {
        console.warn("ViewerUIControls: showLandBtn not provided.");
    }

    if (viewerWireframeCheckbox) {
        viewerWireframeCheckbox.addEventListener('change', () => {
            if (typeof commonViewerAPI.setWireframe === 'function') {
                commonViewerAPI.setWireframe(viewerWireframeCheckbox.checked);
            }
        });
    } else {
        console.warn("ViewerUIControls: viewerWireframeCheckbox not provided.");
    }

    if (viewerAutoRotateCheckbox) {
        viewerAutoRotateCheckbox.addEventListener('change', () => {
            if (typeof commonViewerAPI.setAutoRotate === 'function') {
                commonViewerAPI.setAutoRotate(viewerAutoRotateCheckbox.checked);
            }
        });
    } else {
        console.warn("ViewerUIControls: viewerAutoRotateCheckbox not provided.");
    }

    console.log("ViewerUIControls: Initialized.");
}

export { initializeViewerUIControls };
