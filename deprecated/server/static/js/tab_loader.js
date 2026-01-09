// static/js/tab_loader.js

/**
 * Loads the content and script for a given tab.
 * @param {HTMLButtonElement} tabButton - The tab button element that was clicked.
 * @param {object} config - Configuration object.
 * @param {HTMLElement} config.parametersTitle - Element for the parameters panel title.
 * @param {HTMLElement} config.parametersFormContainer - Container for tab-specific form HTML.
 * @param {NodeListOf<HTMLButtonElement>} config.allTabButtons - All tab buttons for managing active state.
 * @param {HTMLElement} config.mainGenerateBtn - The main generate/action button.
 * @param {HTMLElement} config.landgenTabButtonElement - Specific reference to the landgen tab button for state checks.
 * @param {object} config.commonViewerAPI - The common API for viewer interactions.
 * @param {function} config.setCurrentTabHandlerAPI - Callback to set the current tab handler in the main shell.
 * @param {function} config.setCurrentActiveTabId - Callback to set the current active tab ID in the main shell.
 * @param {object} config.getCurrentTabHandlerAPI - Callback to get the current tab handler for disposal.
 */
async function loadTabAndUpdate(tabButton, config) {
    if (!tabButton) return;

    const {
        parametersTitle,
        parametersFormContainer,
        allTabButtons,
        mainGenerateBtn,
        landgenTabButtonElement,
        commonViewerAPI,
        setCurrentTabHandlerAPI,
        setCurrentActiveTabId,
        getCurrentTabHandlerAPI
    } = config;

    const tabId = tabButton.getAttribute('data-tab-id');
    const contentUrl = tabButton.getAttribute('data-tab-content-url');
    const scriptUrl = tabButton.getAttribute('data-tab-js');

    setCurrentActiveTabId(tabId);

    allTabButtons.forEach(btn => btn.classList.remove('active-tab'));
    tabButton.classList.add('active-tab');

    if (parametersTitle) parametersTitle.textContent = `${tabButton.textContent.trim()} Parameters`;
    if (parametersFormContainer) parametersFormContainer.innerHTML = '<p class="text-gray-500">Loading parameters...</p>';

    // commonViewerAPI.clearScene(); // Moved to be more conditional or handled by tab handlers

    const currentHandler = getCurrentTabHandlerAPI();
    if (currentHandler && typeof currentHandler.dispose === 'function') {
        console.log(`TabLoader: Disposing handler for previous tab: ${currentHandler.tabId || 'unknown'}`);
        currentHandler.dispose();
    }
    setCurrentTabHandlerAPI(null); // Reset current handler

    if (contentUrl) {
        try {
            const response = await fetch(contentUrl);
            if (!response.ok) throw new Error(`Failed to load tab content: ${response.statusText} for ${contentUrl}`);
            const htmlFragment = await response.text();
            if (parametersFormContainer) {
                parametersFormContainer.innerHTML = htmlFragment;
            } else {
                console.error("TabLoader: parametersFormContainer not found to inject HTML fragment");
            }

            if (scriptUrl) {
                const oldScript = document.getElementById(`${tabId}-handler-script`);
                if (oldScript) oldScript.remove();

                const scriptElement = document.createElement('script');
                scriptElement.id = `${tabId}-handler-script`;
                scriptElement.src = scriptUrl;
                scriptElement.type = 'module';
                scriptElement.onload = () => {
                    const handlerName = `initialize${tabId.charAt(0).toUpperCase() + tabId.slice(1)}Tab`;
                    let newHandler = null;
                    if (window[handlerName] && typeof window[handlerName] === 'function') {
                        newHandler = window[handlerName](commonViewerAPI);
                        if (newHandler) newHandler.tabId = tabId; // Store tabId for debugging dispose
                        setCurrentTabHandlerAPI(newHandler);
                        console.log(`TabLoader: Initialized tab handler for ${tabId}`);

                        if (newHandler && typeof newHandler.updateGenerateButton === 'function') {
                            newHandler.updateGenerateButton();
                        } else if (mainGenerateBtn) { // Fallback default button state
                            mainGenerateBtn.textContent = 'Generate';
                            mainGenerateBtn.classList.remove('btn-secondary');
                            mainGenerateBtn.classList.add('btn-primary');
                            // Disable main button if landgen tab is active AND its button element is disabled
                            mainGenerateBtn.disabled = (tabId === 'landgen' && landgenTabButtonElement && landgenTabButtonElement.disabled);
                        }
                    } else {
                        console.warn(`TabLoader: Init function ${handlerName} not found for tab ${tabId}`);
                        // Set default button state if handler fails to load/init
                        if (mainGenerateBtn) {
                            mainGenerateBtn.textContent = 'Generate';
                            mainGenerateBtn.classList.remove('btn-secondary');
                            mainGenerateBtn.classList.add('btn-primary');
                            mainGenerateBtn.disabled = true; // Disable if tab handler is missing
                        }
                    }
                };
                scriptElement.onerror = () => {
                    console.error(`TabLoader: Failed to load script for tab ${tabId}: ${scriptUrl}`);
                    if (parametersFormContainer) parametersFormContainer.innerHTML = `<p class="text-red-500">Error loading script for ${tabId}.</p>`;
                    if (mainGenerateBtn) { // Also disable button on script error
                        mainGenerateBtn.textContent = 'Error';
                        mainGenerateBtn.disabled = true;
                    }
                };
                document.body.appendChild(scriptElement);
            }
        } catch (error) {
            console.error("TabLoader: Error loading tab content:", error);
            if (parametersFormContainer) parametersFormContainer.innerHTML = `<p class="text-red-500">Error loading parameters for ${tabId}.</p>`;
            if (commonViewerAPI && commonViewerAPI.showStatus) commonViewerAPI.showStatus(`Error loading tab: ${error.message}`, 'error');
            if (mainGenerateBtn) {
                mainGenerateBtn.textContent = 'Error';
                mainGenerateBtn.disabled = true;
            }
        }
    }
}


/**
 * Initializes the tab loading system.
 * @param {object} config - Configuration object (see loadTabAndUpdate for details).
 */
function initializeTabLoader(config) {
    const { allTabButtons, landgenTabButtonElement } = config;

    allTabButtons.forEach(button => {
        // Add click listener to all tabs except the initially disabled landgen tab
        // The landgen tab listener is managed by updateLandgenTabState and MutationObserver in app_shell_main
        if (button.id !== 'landgenTabButton') {
            button.addEventListener('click', () => loadTabAndUpdate(button, config));
        }
    });

    // The landgenTabButtonElement's click listener is handled in app_shell_main.js
    // via handleLandgenTabClick and MutationObserver, which will also call loadTabAndUpdate.

    console.log("TabLoader: Initialized tab button listeners (excluding initially disabled landgen).");

    // Expose loadTabAndUpdate if needed to be called externally (e.g., by landgen tab enabling logic)
    return {
        loadTab: (tabButton) => loadTabAndUpdate(tabButton, config)
    };
}

export { initializeTabLoader, loadTabAndUpdate }; // Export loadTabAndUpdate directly if needed by app_shell_main for landgen tab
