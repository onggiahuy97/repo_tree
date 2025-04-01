package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

// Add these structs for Claude API request/response
type ClaudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ClaudeResponse struct {
	Content []ClaudeContent `json:"content"`
}

type ClaudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// RepoInfo is used to parse /repos/{owner}/{repo} response (to get default_branch, etc.)
type RepoInfo struct {
	DefaultBranch string `json:"default_branch"`
}

// TreeResponse corresponds to /repos/{owner}/{repo}/git/trees/{sha}?recursive=1
type TreeResponse struct {
	SHA       string      `json:"sha"`
	Truncated bool        `json:"truncated"`
	Tree      []TreeEntry `json:"tree"`
}

// TreeEntry is each file or directory from the Git Tree API
type TreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"` // "blob" (file), "tree" (folder), or "commit"
	SHA  string `json:"sha"`
	URL  string `json:"url"`
}

// Node represents a file or directory in our in-memory tree
type Node struct {
	Name     string
	IsDir    bool
	Children map[string]*Node
}

func main() {
	// Load environment variables from .env (like GITHUB_TOKEN)
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: could not load .env file (ignore if environment variables are already set).")
	}

	http.HandleFunc("/tree", corsMiddleware(handleTree))
	http.HandleFunc("/ai", corsMiddleware(handleDescribeRepo))

	fmt.Println("Server starting on port :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

// cors helper
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS preflight request
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// For actual requests, set the CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Call the actual handler
		next(w, r)
	}
}

// handleTree returns a plain-text ASCII tree of the repo's contents.
func handleTree(w http.ResponseWriter, r *http.Request) {
	// We'll do partial flushes if the client supports streaming (optional)
	flusher, ok := w.(http.Flusher)

	repoURL := r.URL.Query().Get("repo")
	if repoURL == "" {
		http.Error(w, "Missing 'repo' query param, e.g. ?repo=https://github.com/owner/repo", http.StatusBadRequest)
		return
	}

	owner, repo, err := parseGitHubRepo(repoURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return as plain text
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// 1) Get default branch
	fmt.Fprintln(w, "🥶 Building the entire tree via Git Tree API...")
	if ok {
		flusher.Flush()
	}

	defaultBranch, err := fetchDefaultBranch(owner, repo)
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		if ok {
			flusher.Flush()
		}
		return
	}

	// 2) Fetch the entire tree from the default branch
	treeResp, err := fetchGitTree(owner, repo, defaultBranch)
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		if ok {
			flusher.Flush()
		}
		return
	}

	// 3) Build an in-memory tree
	root := buildTree(treeResp.Tree)
	fmt.Printf("root: %+v\n", root)

	// 4) Print ASCII tree
	fmt.Fprintln(w, "🥶 Project structure:")
	if ok {
		flusher.Flush()
	}
	printNode(w, flusher, root, "")
}

// Complete the handleDescribeRepo function
func handleDescribeRepo(w http.ResponseWriter, r *http.Request) {
	// Get the repo URL from query parameters
	repoURL := r.URL.Query().Get("repo")
	if repoURL == "" {
		http.Error(w, "Missing 'repo' query param, e.g. ?repo=https://github.com/owner/repo", http.StatusBadRequest)
		return
	}

	// Parse the GitHub repo
	owner, repo, err := parseGitHubRepo(repoURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get the default branch
	defaultBranch, err := fetchDefaultBranch(owner, repo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch the Git tree
	treeResp, err := fetchGitTree(owner, repo, defaultBranch)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build the tree structure
	root := buildTree(treeResp.Tree)
	treeText := BuildTreeText(root)

	// Generate the diagram using the Claude API
	diagramText, err := generateDiagramFromTree(treeText)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"diagram": diagramText,
	})

}

// Function to call the Claude API
func generateDiagramFromTree(treeText string) (string, error) {
	// Get the Claude API key from environment variables
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("CLAUDE_API_KEY not set in environment variables")
	}

	// Create the prompt for Claude
	prompt := fmt.Sprintf(`# Generate Architecture Diagram from Repository Structure

I have a repository with the following structure:
%s

## Instructions:
Create a clean, well-organized Mermaid architecture diagram based on this repository structure. Your diagram should:

1. Begin EXACTLY with "graph TD" as the first line
2. Group related components into logical subgraphs based on functionality
3. Create explicit nodes for ALL entities that will have relationships
4. Connect only actual nodes (not subgraphs) with descriptive relationship arrows
5. Use proper Mermaid syntax, escaping or renaming file names with special characters
6. Be space-efficient and visually clean
7. Show clear component hierarchies and data flows

Example of good pattern:
graph TD
   %% Component Group
   subgraph "Service Layer"
       ServiceA[Service A] --> |Uses| ServiceB[Service B]
   end
   
   %% Another component group
   subgraph "Data Layer"
       DB[Database]
       Cache[Cache Service]
   end
   
   ServiceB --> |Reads from| DB

Return ONLY the raw Mermaid diagram code without any introduction or explanation. The diagram must be valid and renderable by Mermaid.js.`, treeText)

	fmt.Printf("Prompt for Claude: %s\n", prompt)

	// Create the request body
	reqBody := ClaudeRequest{
		Model:     "claude-3-haiku-20240307",
		MaxTokens: 4000,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Marshal the request body to JSON
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var claudeResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return "", err
	}

	// Extract the diagram code from the response
	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from Claude API")
	}

	diagramText := claudeResp.Content[0].Text

	// Extract just the mermaid code if it's wrapped in markdown code blocks
	if strings.Contains(diagramText, "```mermaid") {
		parts := strings.Split(diagramText, "```mermaid")
		if len(parts) > 1 {
			parts = strings.Split(parts[1], "```")
			diagramText = strings.TrimSpace(parts[0])
		}
	} else if strings.Contains(diagramText, "```") {
		parts := strings.Split(diagramText, "```")
		if len(parts) > 1 {
			diagramText = strings.TrimSpace(parts[1])
		}
	}

	return diagramText, nil
}

// parseGitHubRepo extracts "owner" and "repo" from a URL like "https://github.com/owner/repo"
func parseGitHubRepo(rawURL string) (string, string, error) {
	const prefix = "https://github.com/"
	if !strings.HasPrefix(rawURL, prefix) {
		return "", "", fmt.Errorf("URL must start with %q", prefix)
	}
	path := strings.TrimPrefix(rawURL, prefix)
	path = strings.TrimSuffix(path, "/")

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("Invalid GitHub URL, must have at least 'owner/repo'")
	}
	return parts[0], parts[1], nil
}

// fetchDefaultBranch calls /repos/{owner}/{repo} to find the default_branch
func fetchDefaultBranch(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	body, err := doGet(url)
	if err != nil {
		return "", err
	}
	var info RepoInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return "", err
	}
	if info.DefaultBranch == "" {
		return "main", nil
	}
	return info.DefaultBranch, nil
}

// fetchGitTree calls /repos/{owner}/{repo}/git/trees/{branchOrSha}?recursive=1
func fetchGitTree(owner, repo, branchOrSha string) (*TreeResponse, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, repo, branchOrSha)

	body, err := doGet(url)
	if err != nil {
		return nil, err
	}

	var tr TreeResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, err
	}
	if tr.Truncated {
		fmt.Println("Warning: the tree is truncated by GitHub!")
	}
	return &tr, nil
}

// doGet sets "Authorization: Bearer ..." if GITHUB_TOKEN is in .env, then performs the GET
func doGet(url string) ([]byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	str := json.NewDecoder(resp.Body)
	fmt.Println(str)

	return io.ReadAll(resp.Body)
}

// buildTree from the entire list of file/directory entries
func buildTree(entries []TreeEntry) *Node {
	root := &Node{Name: "", IsDir: true, Children: map[string]*Node{}}
	for _, entry := range entries {
		parts := strings.Split(entry.Path, "/")
		insertPath(root, parts, entry.Type)
	}
	return root
}

// insertPath inserts a path (e.g. "folder/subfolder/file.go") into the Node tree
func insertPath(current *Node, parts []string, entryType string) {
	if len(parts) == 0 {
		return
	}
	name := parts[0]

	child, exists := current.Children[name]
	if !exists {
		child = &Node{
			Name:     name,
			IsDir:    (entryType == "tree"),
			Children: map[string]*Node{},
		}
		current.Children[name] = child
	}

	if len(parts) > 1 {
		insertPath(child, parts[1:], entryType)
	}
}

// printNode recursively prints a node with ASCII prefixes, optionally flushing each line
func printNode(w http.ResponseWriter, flusher http.Flusher, node *Node, prefix string) {
	names := make([]string, 0, len(node.Children))
	for name := range node.Children {
		names = append(names, name)
	}
	sort.Strings(names)

	for i, name := range names {
		child := node.Children[name]
		isLast := (i == len(names)-1)

		branch := "├── "
		childPrefix := "│   "
		if isLast {
			branch = "└── "
			childPrefix = "    "
		}

		fmt.Fprintf(w, "%s%s%s\n", prefix, branch, child.Name)
		// flush if supported
		if flusher != nil {
			flusher.Flush()
		}

		if child.IsDir {
			newPrefix := prefix + childPrefix
			printNode(w, flusher, child, newPrefix)
		}
	}
}

func BuildTreeText(node *Node) string {
	var builder strings.Builder
	buildTreeText(&builder, node, "")
	return builder.String()
}

func buildTreeText(builder *strings.Builder, node *Node, prefix string) {
	names := make([]string, 0, len(node.Children))
	for name := range node.Children {
		names = append(names, name)
	}
	sort.Strings(names)

	for i, name := range names {
		child := node.Children[name]
		isLast := i == len(names)-1

		branch := "├── "
		childPrefix := "│   "
		if isLast {
			branch = "└── "
			childPrefix = "    "
		}

		builder.WriteString(fmt.Sprintf("%s%s%s\n", prefix, branch, child.Name))
		if child.IsDir {
			buildTreeText(builder, child, prefix+childPrefix)
		}
	}
}
