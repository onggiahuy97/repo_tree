// package main

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net/http"
// 	"os"
// 	"strings"
// 	"sync"
// 	"time"

// 	"github.com/joho/godotenv"
// )

// type RepoContent struct {
// 	Name    string `json:"name"`
// 	URL     string `json:"url"`
// 	Type    string `json:"type"`
// 	Message string `json:"message"` // For error message from Github API
// }

// type Item struct {
// 	Name  string
// 	Type  string
// 	Items []Item
// }

// type responseWriter struct {
// 	http.ResponseWriter
// 	flusher http.Flusher
// }

// type ErrorResponse struct {
// 	Error string `json:"error"`
// }

// func init() {
// 	if err := godotenv.Load(); err != nil {
// 		log.Printf("No .env file found or error loading: %v", err)
// 	}
// }

// func FetchJSON(url string) ([]RepoContent, error) {
// 	client := &http.Client{}

// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
// 		req.Header.Add("Authorization", "Bearer "+token)
// 	}
// 	req.Header.Add("Accept", "application/json")

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var contents []RepoContent
// 	err = json.Unmarshal(body, &contents)
// 	if err != nil {
// 		var singleContent RepoContent
// 		err = json.Unmarshal(body, &singleContent)
// 		if err != nil {
// 			return nil, err
// 		}

// 		if singleContent.Message != "" {
// 			return nil, fmt.Errorf("Github API error: %s", singleContent.Message)
// 		}

// 		contents = []RepoContent{singleContent}
// 	}

// 	return contents, nil
// }

// func buildTree(url string) ([]Item, error) {
// 	contents, err := FetchJSON(url)
// 	if err != nil {
// 		return nil, err
// 	}

// 	items := make([]Item, 0, len(contents))
// 	for _, content := range contents {
// 		item := Item{
// 			Name: content.Name,
// 			Type: content.Type,
// 		}

// 		if content.Type == "dir" {
// 			subItems, err := buildTree(content.URL)
// 			if err != nil {
// 				return nil, err
// 			}
// 			item.Items = subItems
// 		}

// 		items = append(items, item)

// 	}

// 	return items, nil

// }

// func printTree(items []Item, prefix string, isLast bool) {
// 	for i, item := range items {
// 		isLastItem := i == len(items)-1

// 		// Print current item
// 		if isLast {
// 			fmt.Printf("%s└── %s\n", prefix, item.Name)
// 		} else {
// 			fmt.Printf("%s├── %s\n", prefix, item.Name)
// 		}

// 		// Handle nested items
// 		if len(item.Items) > 0 {
// 			newPrefix := prefix
// 			if isLastItem {
// 				newPrefix += "    " // 4 spaces when it's the last item
// 			} else {
// 				newPrefix += "│   " // vertical line with 3 spaces for non-last items
// 			}
// 			printTree(item.Items, newPrefix, isLastItem)
// 		}
// 	}
// }

// func renderTree(items []Item, prefix string, isLast bool, buffer *bytes.Buffer) {
// 	for i, item := range items {
// 		isLastItem := i == len(items)-1

// 		if isLast {
// 			buffer.WriteString(fmt.Sprintf("%s└── %s\n", prefix, item.Name))
// 		} else {
// 			buffer.WriteString(fmt.Sprintf("%s├── %s\n", prefix, item.Name))
// 		}

// 		if len(item.Items) > 0 {
// 			newPrefix := prefix
// 			if isLastItem {
// 				newPrefix += "	 "
// 			} else {
// 				newPrefix += "│   "
// 			}
// 			renderTree(item.Items, newPrefix, isLastItem, buffer)
// 		}
// 	}
// }

// func makeURL(url string) string {
// 	var baseURL = "https://api.github.com/repos/"
// 	url = strings.TrimSuffix(url, "/")
// 	parts := strings.Split(url, "github.com/")
// 	if len(parts) != 2 {
// 		return ""
// 	}
// 	repoPath := parts[1]
// 	return baseURL + repoPath + "/contents"
// }

// func handleTree(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodGet {
// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	repoURL := r.URL.Query().Get("repo")
// 	if repoURL == "" {
// 		json.NewEncoder(w).Encode(ErrorResponse{Error: "repo parameter is required"})
// 		return
// 	}

// 	apiURL := makeURL(repoURL)
// 	if apiURL == "" {
// 		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid repository URL"})
// 		return
// 	}

// 	tree, err := buildTree(apiURL)
// 	if err != nil {
// 		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
// 		return
// 	}

// 	var buffer bytes.Buffer
// 	buffer.WriteString("🥶 Project structure:\n")
// 	renderTree(tree, "", true, &buffer)

// 	w.Header().Set("Content-Type", "text/plain")
// 	w.Write(buffer.Bytes())

// }

// func streamTree(items []Item, prefix string, isLast bool, w *responseWriter) {
// 	for i, item := range items {
// 		isLastItem := i == len(items)-1

// 		// Create line for current item
// 		var line string
// 		if isLast {
// 			line = fmt.Sprintf("%s└── %s\n", prefix, item.Name)
// 		} else {
// 			line = fmt.Sprintf("%s├── %s\n", prefix, item.Name)
// 		}

// 		// Write and flush the line
// 		fmt.Fprintf(w, "data: %s\n\n", line)
// 		w.flusher.Flush()

// 		// add small time to simulate the processing time
// 		time.Sleep(100 * time.Millisecond)

// 		if len(item.Items) > 0 {
// 			newPrefix := prefix
// 			if isLastItem {
// 				newPrefix += "	 "
// 			} else {
// 				newPrefix += "│   "
// 			}
// 			streamTree(item.Items, newPrefix, isLastItem, w)
// 		}
// 	}
// }

// func buildStreamingTree(url string, w *responseWriter) error {
// 	fmt.Fprintln(w, "🥶 Project structure:")
// 	w.flusher.Flush()

// 	// Channel to receive tree items
// 	itemChan := make(chan string)
// 	done := make(chan bool)
// 	var wg sync.WaitGroup

// 	// Writer goroutine
// 	go func() {
// 		for line := range itemChan {
// 			fmt.Fprint(w, line)
// 			w.flusher.Flush()
// 		}
// 		done <- true
// 	}()

// 	// Recursive function to process directories
// 	var processDir func(url, prefix string)
// 	processDir = func(url, prefix string) {
// 		defer wg.Done()

// 		contents, err := FetchJSON(url)
// 		if err != nil {
// 			return
// 		}

// 		for i, content := range contents {
// 			isLast := i == len(contents)-1

// 			// Send the current item immediately
// 			if isLast {
// 				itemChan <- fmt.Sprintf("%s└── %s\n", prefix, content.Name)
// 			} else {
// 				itemChan <- fmt.Sprintf("%s├── %s\n", prefix, content.Name)
// 			}

// 			if content.Type == "dir" {
// 				newPrefix := prefix
// 				if isLast {
// 					newPrefix += "    "
// 				} else {
// 					newPrefix += "│   "
// 				}

// 				wg.Add(1)
// 				processDir(content.URL, newPrefix)
// 			}
// 		}
// 	}

// 	// Start processing
// 	wg.Add(1)
// 	processDir(url, "")

// 	// Wait for all directories to be processed
// 	go func() {
// 		wg.Wait()
// 		close(itemChan)
// 	}()

// 	// Wait for writer to finish
// 	<-done
// 	return nil
// }

// func handleStreamingTree(w http.ResponseWriter, r *http.Request) {
// 	flusher, ok := w.(http.Flusher)
// 	if !ok {
// 		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
// 		return
// 	}

// 	repoURL := r.URL.Query().Get("repo")
// 	if repoURL == "" {
// 		http.Error(w, "repo parameter is required", http.StatusBadRequest)
// 		return
// 	}

// 	apiURL := makeURL(repoURL)
// 	if apiURL == "" {
// 		http.Error(w, "invalid repository URL", http.StatusBadRequest)
// 		return
// 	}

// 	// Set headers for streaming
// 	w.Header().Set("Content-Type", "text/plain")
// 	w.Header().Set("Cache-Control", "no-cache")
// 	w.Header().Set("Connection", "keep-alive")
// 	w.Header().Set("Access-Control-Allow-Origin", "*")

// 	writer := &responseWriter{
// 		ResponseWriter: w,
// 		flusher:        flusher,
// 	}

// 	if err := buildStreamingTree(apiURL, writer); err != nil {
// 		fmt.Fprintf(writer, "Error: %s\n", err.Error())
// 		writer.flusher.Flush()
// 		return
// 	}
// }

// func main() {
// 	http.HandleFunc("/tree", handleTree)
// 	http.HandleFunc("/tree-stream", handleStreamingTree)

//		port := ":8080"
//		fmt.Printf("Server starting on port %s\n", port)
//		if err := http.ListenAndServe(port, nil); err != nil {
//			panic(err)
//		}
//	}
// package main

// import (
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net/http"
// 	"os"
// 	"strings"

// 	"github.com/joho/godotenv"
// )

// type RepoContent struct {
// 	Name        string `json:"name"`
// 	Path        string `json:"path"`
// 	SHA         string `json:"sha"`
// 	Size        int    `json:"size"`
// 	URL         string `json:"url"`
// 	HTMLURL     string `json:"html_url"`
// 	GitURL      string `json:"git_url"`
// 	DownloadURL string `json:"download_url"`
// 	Type        string `json:"type"`
// 	// For error handling
// 	Message string `json:"message,omitempty"`
// }

// type responseWriter struct {
// 	http.ResponseWriter
// 	flusher http.Flusher
// }

// func init() {
// 	if err := godotenv.Load(); err != nil {
// 		log.Printf("No .env file found or error loading: %v", err)
// 	}
// }

// // FetchJSON fetches the JSON representation of a GitHub repo (directory or file).
// var count int

// func FetchJSON(url string) ([]RepoContent, error) {
// 	count += 1
// 	fmt.Printf("Count: %d\n", count)
// 	client := &http.Client{}
// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// If you have a GitHub token in your environment, add it.
// 	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
// 		req.Header.Add("Authorization", "Bearer "+token)
// 	}
// 	req.Header.Add("Accept", "application/json")

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var contents []RepoContent
// 	err = json.Unmarshal(body, &contents)
// 	if err != nil {
// 		// In case the endpoint returns a single object instead of an array
// 		var singleContent RepoContent
// 		err2 := json.Unmarshal(body, &singleContent)
// 		if err2 != nil {
// 			return nil, err
// 		}
// 		// Possibly an error from GitHub?
// 		if singleContent.Message != "" {
// 			return nil, fmt.Errorf("GitHub API error: %s", singleContent.Message)
// 		}
// 		contents = []RepoContent{singleContent}
// 	}
// 	return contents, nil
// }

// // printContents prints the contents of the given directory, with proper
// // ASCII-tree indentation, recursively.
// func printContents(w *responseWriter, contents []RepoContent, prefix string) error {
// 	for i, content := range contents {
// 		isLast := (i == len(contents)-1)

// 		// Decide which branch character to use: ├── or └──
// 		branch := "├── "
// 		childPrefix := "│   "
// 		if isLast {
// 			branch = "└── "
// 			childPrefix = "    "
// 		}

// 		// Print this file or directory name
// 		fmt.Fprintf(w, "%s%s%s\n", prefix, branch, content.Name)
// 		w.flusher.Flush()

// 		// If it's a directory, recurse
// 		if content.Type == "dir" {
// 			subContents, err := FetchJSON(content.URL)
// 			if err != nil {
// 				// Print error and continue to the next item
// 				fmt.Fprintf(w, "%s    (error fetching %s: %v)\n", prefix, content.Name, err)
// 				w.flusher.Flush()
// 				continue
// 			}
// 			newPrefix := prefix + childPrefix
// 			if err := printContents(w, subContents, newPrefix); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

// func buildStreamingTree(url string, w *responseWriter) error {
// 	fmt.Fprintln(w, "🥶 Project structure:")
// 	w.flusher.Flush()

// 	contents, err := FetchJSON(url)
// 	if err != nil {
// 		return err
// 	}

// 	// Print recursively from the root
// 	if err := printContents(w, contents, ""); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func handleStreamingTree(w http.ResponseWriter, r *http.Request) {
// 	flusher, ok := w.(http.Flusher)
// 	if !ok {
// 		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
// 		return
// 	}

// 	repoURL := r.URL.Query().Get("repo")
// 	if repoURL == "" {
// 		http.Error(w, "repo parameter is required", http.StatusBadRequest)
// 		return
// 	}

// 	apiURL := makeURL(repoURL)
// 	if apiURL == "" {
// 		http.Error(w, "invalid repository URL", http.StatusBadRequest)
// 		return
// 	}

// 	// Set headers for streaming
// 	w.Header().Set("Content-Type", "text/plain")
// 	w.Header().Set("Cache-Control", "no-cache")
// 	w.Header().Set("Connection", "keep-alive")
// 	w.Header().Set("Access-Control-Allow-Origin", "*")

// 	writer := &responseWriter{
// 		ResponseWriter: w,
// 		flusher:        flusher,
// 	}

// 	if err := buildStreamingTree(apiURL, writer); err != nil {
// 		fmt.Fprintf(writer, "Error: %s\n", err.Error())
// 		writer.flusher.Flush()
// 	}
// }

// // Convert a GitHub repo URL into the GitHub API URL for fetching directory/file info.
// //
// // For example:
// //
// //	"https://github.com/onggiahuy97/learn-cicd-starter"
// //	becomes
// //	"https://api.github.com/repos/onggiahuy97/learn-cicd-starter/contents"
// func makeURL(rawRepoURL string) string {
// 	// Basic approach: trim "https://github.com/", then prefix with "https://api.github.com/repos/"
// 	// and append "/contents".
// 	const prefix = "https://github.com/"
// 	if !strings.HasPrefix(rawRepoURL, prefix) {
// 		return ""
// 	}
// 	repoPath := strings.TrimPrefix(rawRepoURL, prefix)
// 	if repoPath == "" {
// 		return ""
// 	}
// 	return "https://api.github.com/repos/" + repoPath + "/contents"
// }

// func main() {
// 	http.HandleFunc("/tree-stream", handleStreamingTree)
// 	port := ":8080"
// 	fmt.Printf("Server starting on port %s\n", port)
// 	if err := http.ListenAndServe(port, nil); err != nil {
// 		panic(err)
// 	}
// }
