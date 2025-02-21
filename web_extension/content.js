// Function to add the custom button
function addCustomButton() {
	const container = document.querySelector('.jxTzTd');

	if (!container || container.querySelector('.custom-github-button')) {
		return; // Exit if container not found or button already exists
	}

	const button = document.createElement('button');
	button.className = 'prc-Button-ButtonBase-c50BI custom-github-button';
	button.setAttribute('type', 'button');
	button.setAttribute('data-size', 'medium');
	button.setAttribute('data-variant', 'default');
	button.style.borderColor = 'green';

	button.innerHTML = `
        <span data-component="buttonContent" class="prc-Button-ButtonContent-HKbr-">
            <span data-component="text" class="prc-Button-Label-pTQ3x">
                 🌲 Tree View
            </span>
        </span>
    `;

	button.addEventListener('click', () => {
		const currentURL = window.location.href;
		// Remove any trailing slashes and /tree/main if present
		const cleanURL = currentURL.replace(/\/+$/, '').replace(/\/tree\/main$/, '');
		// Create the URL for your tree viewer page - remove /tree path
		const treeViewerURL = `http://localhost:8081/index.html?repo=${encodeURIComponent(cleanURL)}`;
		window.open(treeViewerURL, '_blank');
	});

	container.appendChild(button);
}

// Handle GitHub's dynamic page loading
function initializeObserver() {
	const observer = new MutationObserver((mutations) => {
		for (const mutation of mutations) {
			if (mutation.addedNodes.length) {
				addCustomButton();
			}
		}
	});

	observer.observe(document.body, {
		childList: true,
		subtree: true
	});
}

// Initial run
if (document.readyState === 'loading') {
	document.addEventListener('DOMContentLoaded', initializeObserver);
} else {
	initializeObserver();
}


