package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

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

	http.HandleFunc("/tree", handleTree)

	fmt.Println("Server starting on port :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

// handleTree returns a plain-text ASCII tree of the repo's contents.
func handleTree(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight request
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// For actual GET requests, set the CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

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

	// 4) Print ASCII tree
	fmt.Fprintln(w, "🥶 Project structure:")
	if ok {
		flusher.Flush()
	}
	printNode(w, flusher, root, "")
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
