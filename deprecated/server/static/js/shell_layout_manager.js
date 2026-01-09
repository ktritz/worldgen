// static/js/shell_layout_manager.js

/**
 * Initializes the shell layout manager which handles hamburger menu,
 * parameters panel, and fullscreen toggle.
 * * @param {object} commonViewerAPI - The common viewer API, expected to have an onWindowResize method.
 * @param {object} elements - An object containing references to necessary DOM elements.
 * @param {HTMLElement} elements.hamburgerBtn - The hamburger button element.
 * @param {HTMLElement} elements.parametersColumn - The parameters panel element.
 * @param {HTMLElement} elements.mainContentArea - The main content area element.
 * @param {HTMLElement} elements.toggleFullscreenBtn - The fullscreen toggle button element.
 * @param {HTMLElement} elements.viewerContainer - The 3D viewer container element.
 */
function initializeShellLayout(commonViewerAPI, elements) {
    const {
        hamburgerBtn,
        parametersColumn,
        mainContentArea,
        toggleFullscreenBtn,
        viewerContainer
    } = elements;

    if (!hamburgerBtn || !parametersColumn || !mainContentArea) {
        console.warn("ShellLayoutManager: Hamburger button, parameters column, or main content area not provided. Panel toggle functionality will be disabled.");
    } else {
        hamburgerBtn.addEventListener('click', () => {
            parametersColumn.classList.toggle('is-open');
            hamburgerBtn.classList.toggle('is-active');
            mainContentArea.classList.toggle('panel-is-open');

            // Notify viewer to resize after panel animation
            parametersColumn.addEventListener('transitionend', function onTransitionEnd() {
                if (commonViewerAPI && typeof commonViewerAPI.onWindowResize === 'function') {
                    commonViewerAPI.onWindowResize();
                }
                parametersColumn.removeEventListener('transitionend', onTransitionEnd);
            }, { once: true });
            // Fallback resize if transitionend doesn't fire (e.g. no transition or interrupted)
            setTimeout(() => {
                if (commonViewerAPI && typeof commonViewerAPI.onWindowResize === 'function') {
                    commonViewerAPI.onWindowResize();
                }
            }, 350); // Should match CSS transition duration for parametersColumn
        });
    }

    // --- Fullscreen Toggle Logic ---
    function enterFullscreen() {
        if (!viewerContainer || !toggleFullscreenBtn) return;
        document.body.classList.add('viewer-fullscreen-active');
        viewerContainer.classList.add('viewer-fullscreen');
        const icon = toggleFullscreenBtn.querySelector('i');
        if (icon) {
            icon.classList.remove('fa-expand');
            icon.classList.add('fa-compress');
        }
        toggleFullscreenBtn.title = "Exit Fullscreen";

        // Notify viewer to resize after fullscreen transition
        viewerContainer.addEventListener('transitionend', function onFsEnter() {
            if (commonViewerAPI && typeof commonViewerAPI.onWindowResize === 'function') {
                commonViewerAPI.onWindowResize();
            }
            viewerContainer.removeEventListener('transitionend', onFsEnter);
        }, { once: true });
        // Fallback resize
        setTimeout(() => {
            if (commonViewerAPI && typeof commonViewerAPI.onWindowResize === 'function') {
                commonViewerAPI.onWindowResize();
            }
        }, 50); // Short delay for quick transition
    }

    function exitFullscreen() {
        if (!viewerContainer || !toggleFullscreenBtn) return;
        document.body.classList.remove('viewer-fullscreen-active');
        viewerContainer.classList.remove('viewer-fullscreen');
        const icon = toggleFullscreenBtn.querySelector('i');
        if (icon) {
            icon.classList.remove('fa-compress');
            icon.classList.add('fa-expand');
        }
        toggleFullscreenBtn.title = "Toggle Fullscreen";

        // Notify viewer to resize after fullscreen transition
        // Use a timeout that matches the transition duration for viewerContainer
        setTimeout(() => {
            if (commonViewerAPI && typeof commonViewerAPI.onWindowResize === 'function') {
                commonViewerAPI.onWindowResize();
            }
        }, 350); // Assuming 0.3s transition for viewerContainer itself
    }

    if (!toggleFullscreenBtn || !viewerContainer) {
        console.warn("ShellLayoutManager: Fullscreen toggle button or viewer container not provided. Fullscreen functionality will be disabled.");
    } else {
        toggleFullscreenBtn.addEventListener('click', (event) => {
            event.preventDefault();
            if (viewerContainer.classList.contains('viewer-fullscreen')) {
                exitFullscreen();
            } else {
                enterFullscreen();
            }
        });
    }

    // Global keydown listener for Escape from fullscreen
    document.addEventListener('keydown', (event) => {
        if (event.key === "Escape" && viewerContainer && viewerContainer.classList.contains('viewer-fullscreen')) {
            exitFullscreen();
        }
    });

    console.log("ShellLayoutManager: Initialized.");

    // This module doesn't need to return an API for now, it just sets up listeners.
    // If other modules needed to programmatically toggle panel/fullscreen, an API could be exposed.
    return {
        // Example of a potential API method if needed later:
        // togglePanel: () => {
        //     if (hamburgerBtn) hamburgerBtn.click();
        // }
    };
}

export { initializeShellLayout };
