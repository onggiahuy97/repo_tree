// Initialize Mermaid
mermaid.initialize({
	startOnLoad: false,
	theme: 'default'
});

document.addEventListener('DOMContentLoaded', function() {
	const generateBtn = document.getElementById('generateBtn');
	const repoUrlInput = document.getElementById('repoUrl');
	const loadingIndicator = document.getElementById('loading');
	const mermaidOutput = document.getElementById('mermaidOutput');

	// API endpoint (should match your Go server)
	const API_BASE_URL = 'http://localhost:8080';

	generateBtn.addEventListener('click', async function() {
		const repoUrl = repoUrlInput.value.trim();
		if (!repoUrl) {
			alert('Please enter a GitHub repository URL');
			return;
		}

		// Show loading indicator
		loadingIndicator.style.display = 'block';
		mermaidOutput.innerHTML = '';

		try {
			// Call the API endpoint
			const response = await fetch(`${API_BASE_URL}/ai?repo=${encodeURIComponent(repoUrl)}`);

			if (!response.ok) {
				const errorText = await response.text();
				throw new Error(`API error: ${errorText}`);
			}

			const data = await response.json();

			//data = sample

			// Render the diagram
			mermaidOutput.innerHTML = data.diagram;
			await mermaid.run();

		} catch (error) {
			console.error('Error:', error);
			mermaidOutput.innerHTML = `<div style="color: red;">Error: ${error.message}</div>`;
		} finally {
			loadingIndicator.style.display = 'none';
		}
	});
});

sample = {
	diagram: `
	graph TD
    %% Deployment and Utilities
    subgraph "Deployment and Utilities"
        PCU[Process Cleanup Utility] --> |CleansPorts| DC[Docker Configuration]
        subgraph "Build/Deployment Scripts"
            MF[Makefile] 
            SP[setup.py]
        end
    end

    %% External Clients
    subgraph "External Clients"
        CR[Client Request]
        CUI[Chord UI Client]
    end

    %% API Layer
    subgraph "API Layer"
        API[HTTP API Server]
    end

    %% Distributed DHT Node
    subgraph "Distributed DHT Node"
        CM[Chord Module]
        FD[Failure Detector]
        GP[Gossip Protocol]
        NB[Network Broadcast]
        LE[Leader Election]
        NC[Network Connector]
        
        CM --> |Checks| FD
        CM --> |Gossips| GP
        CM --> |Broadcasts| NB
        CM --> |Triggers| LE
        CM --> |Communicates| NC
    end

    %% Connections between components
    MF --> |Builds| CM
    MF --> |Builds| API
    SP --> |Deploys| API
    DC --> |Hosts| API
    CR --> |APIRequest| API
    CUI --> |SimulatedRequest| API
    API --> |HandlesRequest| CM
    PCU --> |CleansPorts| DC
		`
}
