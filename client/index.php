<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>GitHub Tree Viewer</title>
  <style>
    body {
      margin: 0;
      padding: 0;
      font-family: Arial, sans-serif;
    }
    #container {
      display: flex;
      flex-direction: column;
      align-items: center;
      margin-top: 50px;
    }
    #inputRow {
      display: flex;
      align-items: center;
      margin-bottom: 20px;
    }
    #urlInput {
      width: 400px;
      padding: 8px;
      margin-right: 10px;
      border: 1px solid #ccc;
      border-radius: 4px;
    }
    #btnFetch {
      padding: 8px 16px;
      cursor: pointer;
      border: 1px solid #ccc;
      background-color: #eee;
      border-radius: 4px;
      min-width: 80px;
    }
    #btnFetch:hover {
      background-color: #ddd;
    }
    #btnFetch:disabled {
      cursor: wait;
      opacity: 0.7;
    }
    #responseContainer {
      position: relative;
      width: 65%;
    }
    #responseArea {
      white-space: pre-wrap;
      font-family: monospace;
      width: 100%;
      min-height: 300px;
      border: 1px solid #ccc;
      border-radius: 4px;
      padding: 10px;
      overflow-y: auto;
    }
    #copyButton {
      position: absolute;
      top: 10px;
      right: 10px;
      padding: 6px 12px;
      background-color: #f8f9fa;
      border: 1px solid #ccc;
      border-radius: 4px;
      cursor: pointer;
      font-size: 14px;
      z-index: 1;
    }
    #copyButton:hover {
      background-color: #e9ecef;
    }
    h1 {
      margin-bottom: 30px;
    }
  </style>
</head>
<body>
<div id="container">
  <h1>GitHub Tree Viewer</h1>
  <div id="inputRow">
    <input type="text" id="urlInput" placeholder="e.g. https://github.com/onggiahuy97/iFruitsRecipe"/>
    <button id="btnFetch">Fetch</button>
  </div>
  <div id="responseContainer">
    <button id="copyButton">Copy</button>
    <div id="responseArea">[ Tree response will appear here ]</div>
  </div>
</div>
<script>
  const btnFetch = document.getElementById('btnFetch');
  const urlInput = document.getElementById('urlInput');
  const responseArea = document.getElementById('responseArea');
  const copyButton = document.getElementById('copyButton');

  function setLoading(isLoading) {
    btnFetch.disabled = isLoading;
    urlInput.disabled = isLoading;
    btnFetch.textContent = isLoading ? 'Loading...' : 'Fetch';
  }

  // On button click, fetch the plain-text tree from your Go API
  btnFetch.addEventListener('click', async () => {
    const urlVal = urlInput.value.trim();
    if (!urlVal) {
      alert('Please enter a valid GitHub URL!');
      return;
    }

    setLoading(true);

    try {
      const endpoint = `http://localhost:8080/tree?repo=${encodeURIComponent(urlVal)}`;
      const response = await fetch(endpoint);

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const textData = await response.text();
      responseArea.textContent = textData;
    } catch (error) {
      console.error('Fetch error:', error);
      responseArea.textContent = 'Error: ' + error.message;
    } finally {
      setLoading(false);
    }
  });

  // Copy button functionality
  copyButton.addEventListener('click', async () => {
    const text = responseArea.textContent;

    try {
      await navigator.clipboard.writeText(text);
      copyButton.textContent = 'Copied!';
      setTimeout(() => {
        copyButton.textContent = 'Copy';
      }, 2000);
    } catch (err) {
      console.error('Failed to copy text:', err);
      alert('Failed to copy text to clipboard');
    }
  });
</script>
</body>
</html>
