package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type RepoContent struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Type    string `json:"type"`
	Message string `json:"message"` // For error message from Github API
}

type Item struct {
	Name  string
	Type  string
	Items []Item
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func FetchJSON(url string) ([]RepoContent, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var contents []RepoContent
	err = json.Unmarshal(body, &contents)
	if err != nil {
		var singleContent RepoContent
		err = json.Unmarshal(body, &singleContent)
		if err != nil {
			return nil, err
		}

		if singleContent.Message != "" {
			return nil, fmt.Errorf("Github API error: %s", singleContent.Message)
		}

		contents = []RepoContent{singleContent}
	}

	return contents, nil
}

func buildTree(url string) ([]Item, error) {
	contents, err := FetchJSON(url)
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(contents))
	for _, content := range contents {
		item := Item{
			Name: content.Name,
			Type: content.Type,
		}

		if content.Type == "dir" {
			subItems, err := buildTree(content.URL)
			if err != nil {
				return nil, err
			}
			item.Items = subItems
		}

		items = append(items, item)

	}

	return items, nil

}

func printTree(items []Item, prefix string, isLast bool) {
	for i, item := range items {
		isLastItem := i == len(items)-1

		// Print current item
		if isLast {
			fmt.Printf("%s└── %s\n", prefix, item.Name)
		} else {
			fmt.Printf("%s├── %s\n", prefix, item.Name)
		}

		// Handle nested items
		if len(item.Items) > 0 {
			newPrefix := prefix
			if isLastItem {
				newPrefix += "    " // 4 spaces when it's the last item
			} else {
				newPrefix += "│   " // vertical line with 3 spaces for non-last items
			}
			printTree(item.Items, newPrefix, isLastItem)
		}
	}
}

func renderTree(items []Item, prefix string, isLast bool, buffer *bytes.Buffer) {
	for i, item := range items {
		isLastItem := i == len(items)-1

		if isLast {
			buffer.WriteString(fmt.Sprintf("%s└── %s\n", prefix, item.Name))
		} else {
			buffer.WriteString(fmt.Sprintf("%s├── %s\n", prefix, item.Name))
		}

		if len(item.Items) > 0 {
			newPrefix := prefix
			if isLastItem {
				newPrefix += "	 "
			} else {
				newPrefix += "│   "
			}
			renderTree(item.Items, newPrefix, isLastItem, buffer)
		}
	}
}

func makeURL(url string) string {
	var baseURL = "https://api.github.com/repos/"
	url = strings.TrimSuffix(url, "/")
	parts := strings.Split(url, "github.com/")
	if len(parts) != 2 {
		return ""
	}
	repoPath := parts[1]
	return baseURL + repoPath + "/contents"
}

func handleTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	repoURL := r.URL.Query().Get("repo")
	if repoURL == "" {
		json.NewEncoder(w).Encode(ErrorResponse{Error: "repo parameter is required"})
		return
	}

	apiURL := makeURL(repoURL)
	if apiURL == "" {
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid repository URL"})
		return
	}

	tree, err := buildTree(apiURL)
	if err != nil {
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	var buffer bytes.Buffer
	buffer.WriteString("🥶 Project structure:\n")
	renderTree(tree, "", true, &buffer)

	w.Header().Set("Content-Type", "text/plain")
	w.Write(buffer.Bytes())

}

func main() {
	http.HandleFunc("/tree", handleTree)

	port := ":8080"
	fmt.Printf("Server starting on port %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		panic(err)
	}
}
