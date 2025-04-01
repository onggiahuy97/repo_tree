// Initialize Mermaid
mermaid.initialize({
	startOnLoad: false,
	theme: 'default',
	flowchart: {
		curve: 'basis',
		useMaxWidth: false,
		htmlLabels: true,
		padding: 10,
		rankSpacing: 100,
		nodeSpacing: 100
	},
	securityLevel: 'loose',
	themeVariables: {
		primaryColor: '#f0f7ff',
		primaryTextColor: '#333',
		primaryBorderColor: '#7aa6c2',
		lineColor: '#666',
		secondaryColor: '#f8f9fa',
		tertiaryColor: '#fff8dc'
	}
});

// Add panning functionality
let isDragging = false;
let startX, startY, scrollLeft, scrollTop;

mermaidOutput.addEventListener('mousemove', function(e) {
	if (!isDragging) return;
	e.preventDefault();

	// Calculate the distance moved
	const x = e.pageX - mermaidOutput.offsetLeft;
	const y = e.pageY - mermaidOutput.offsetTop;

	// Calculate the new scroll position
	const newScrollLeft = scrollLeft - (x - startX);
	const newScrollTop = scrollTop - (y - startY);

	// Apply the scroll position
	mermaidOutput.scrollLeft = newScrollLeft;
	mermaidOutput.scrollTop = newScrollTop;

	console.log('Dragging:', { x, y, newScrollLeft, newScrollTop });
});

mermaidOutput.addEventListener('mousedown', function(e) {
	isDragging = true;
	startX = e.pageX - mermaidOutput.offsetLeft;
	startY = e.pageY - mermaidOutput.offsetTop;
	scrollLeft = mermaidOutput.scrollLeft;
	scrollTop = mermaidOutput.scrollTop;
	mermaidOutput.style.cursor = 'grabbing';
});

mermaidOutput.addEventListener('mouseleave', function() {
	isDragging = false;
	mermaidOutput.style.cursor = 'grab';
});

mermaidOutput.addEventListener('mouseup', function() {
	isDragging = false;
	mermaidOutput.style.cursor = 'grab';
});

document.addEventListener('DOMContentLoaded', function() {
	// Elements
	const generateBtn = document.getElementById('generateBtn');
	const btnClear = document.getElementById('btnClear');
	const repoUrlInput = document.getElementById('repoUrl');
	const responseArea = document.getElementById('responseArea');
	const copyButton = document.getElementById('copyButton');
	const loadingIndicator = document.getElementById('loading');
	const mermaidOutput = document.getElementById('mermaidOutput');
	const darkModeToggle = document.getElementById('dark-mode-toggle');
	const zoomIn = document.getElementById('zoomIn');
	const zoomOut = document.getElementById('zoomOut');
	const resetZoom = document.getElementById('resetZoom');

	// API endpoint
	const API_BASE_URL = 'http://localhost:8080';

	// Current zoom level
	let currentZoom = 1;

	// Check for saved dark mode preference
	if (localStorage.getItem('darkMode') === 'enabled') {
		document.body.classList.add('dark-mode');
		darkModeToggle.checked = true;
	}

	// Dark mode toggle functionality
	darkModeToggle.addEventListener('change', () => {
		if (darkModeToggle.checked) {
			document.body.classList.add('dark-mode');
			localStorage.setItem('darkMode', 'enabled');
		} else {
			document.body.classList.remove('dark-mode');
			localStorage.setItem('darkMode', 'disabled');
		}
	});

	// Format tree output with colors and icons
	function formatTreeOutput(text) {
		if (!text || text.includes('[ Repository tree will appear here ]')) {
			return text;
		}

		// Add classes for different elements
		let formatted = text
			.replace(/🥶 Building the entire tree via Git Tree API\.\.\./g, '<span class="emoji">🔍</span> <strong>Building the entire tree via Git Tree API...</strong>')
			.replace(/🥶 Project structure:/g, '<span class="emoji">📂</span> <strong>Project structure:</strong>')
			.replace(/(\|\-\-\- [a-zA-Z0-9_\-\.]+\/)|(├── [a-zA-Z0-9_\-\.]+\/)/g, match => `<span class="directory">${match}</span>`)
			.replace(/(\|\-\-\- [a-zA-Z0-9_\-\.]+)|(├── [a-zA-Z0-9_\-\.]+)(?!\/)(?!\w)/g, match => `<span class="file">${match}</span>`);

		return `<div class="file-tree">${formatted}</div>`;
	}

	// Set loading state
	function setLoading(isLoading) {
		generateBtn.disabled = isLoading;
		repoUrlInput.disabled = isLoading;
		generateBtn.innerHTML = isLoading ? '<i class="fas fa-spinner fa-spin"></i> Loading...' : 'Generate <i class="fas fa-code-branch"></i>';

		if (isLoading) {
			responseArea.innerHTML = '<div class="loading"><div class="spinner"></div>Fetching repository structure...</div>';
		}
	}

	// Apply zoom to the diagram
	function applyZoom() {
		const svg = document.querySelector('#mermaidOutput svg');
		if (svg) {
			// Apply zoom to the SVG
			svg.style.transform = `scale(${currentZoom})`;

			// Adjust container size to fit zoomed content
			if (currentZoom > 1) {
				// Increase container min-width/height when zoomed in
				mermaidOutput.style.minWidth = `${800 * currentZoom}px`;
				mermaidOutput.style.minHeight = `${600 * currentZoom}px`;
			} else {
				// Reset to default when zoomed out
				mermaidOutput.style.minWidth = '';
				mermaidOutput.style.minHeight = '';
			}
		}
	}

	// Zoom controls
	zoomIn.addEventListener('click', function() {
		currentZoom += 0.1;
		applyZoom();
	});

	zoomOut.addEventListener('click', function() {
		currentZoom -= 0.1;
		if (currentZoom < 0.5) currentZoom = 0.5;
		applyZoom();
	});

	resetZoom.addEventListener('click', function() {
		currentZoom = 1;
		applyZoom();
	});

	// Clear button functionality
	btnClear.addEventListener('click', () => {
		repoUrlInput.value = '';
		responseArea.innerHTML = '[ Repository tree will appear here ]';
		mermaidOutput.innerHTML = '';
	});

	// Copy button functionality
	copyButton.addEventListener('click', async () => {
		const text = responseArea.textContent;

		try {
			await navigator.clipboard.writeText(text);
			copyButton.innerHTML = '<i class="fas fa-check"></i> Copied!';
			setTimeout(() => {
				copyButton.innerHTML = '<i class="fas fa-copy"></i> Copy';
			}, 2000);
		} catch (err) {
			console.error('Failed to copy text:', err);
			alert('Failed to copy text to clipboard');
		}
	});

	// Generate button functionality
	generateBtn.addEventListener('click', async () => {
		const repoUrl = repoUrlInput.value.trim();
		if (!repoUrl) {
			alert('Please enter a GitHub repository URL');
			return;
		}

		setLoading(true);
		mermaidOutput.innerHTML = '';

		try {
			// Fetch the repository tree structure
			const treeEndpoint = `${API_BASE_URL}/tree?repo=${encodeURIComponent(repoUrl)}`;
			const treeResponse = await fetch(treeEndpoint);

			if (!treeResponse.ok) {
				throw new Error(`HTTP error! Status: ${treeResponse.status}`);
			}

			// Process the tree structure
			const reader = treeResponse.body.getReader();
			const decoder = new TextDecoder();
			let result = '';

			while (true) {
				const { done, value } = await reader.read();
				if (done) break;

				const chunk = decoder.decode(value, { stream: true });
				result += chunk;

				// Format the result with colors
				responseArea.innerHTML = formatTreeOutput(result);
			}

			// Final decoding to handle any remaining bytes
			const finalChunk = decoder.decode();
			if (finalChunk) result += finalChunk;

			// Format the final result with colors
			responseArea.innerHTML = formatTreeOutput(result);

			// Generate the diagram
			loadingIndicator.style.display = 'block';

			// Call the diagram API endpoint
			const diagramEndpoint = `${API_BASE_URL}/ai?repo=${encodeURIComponent(repoUrl)}`;
			const diagramResponse = await fetch(diagramEndpoint);

			if (!diagramResponse.ok) {
				throw new Error(`API error: ${await diagramResponse.text()}`);
			}

			const data = await diagramResponse.json();
			console.log('API response:', data);

			// Create a div element with the mermaid class
			const mermaidDiv = document.createElement('div');
			mermaidDiv.className = 'mermaid';

			// Extract just the diagram content from the response
			mermaidDiv.textContent = data.diagram;

			// Clear output area and append the div
			mermaidOutput.innerHTML = '';
			mermaidOutput.appendChild(mermaidDiv);

			// Render the diagram
			await mermaid.run();

			// Reset the zoom
			currentZoom = 1;
			applyZoom();

		} catch (error) {
			console.error('Error:', error);
			responseArea.innerHTML = `<div style="color: #d73a49;"><i class="fas fa-exclamation-circle"></i> Error: ${error.message}</div>`;
			mermaidOutput.innerHTML = `<div style="color: #d73a49;"><i class="fas fa-exclamation-circle"></i> Error: ${error.message}</div>`;
		} finally {
			setLoading(false);
			loadingIndicator.style.display = 'none';
		}
	});

	// Auto-fetch if URL parameter is present
	const urlParams = new URLSearchParams(window.location.search);
	const repoParam = urlParams.get('repo');

	if (repoParam) {
		repoUrlInput.value = decodeURIComponent(repoParam);
		generateBtn.click();
	}
});
