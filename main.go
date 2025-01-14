package main

import (
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

}

func main() {
	baseURL := makeURL("https://github.com/onggiahuy97/learn-cicd-starter")
	tree, err := buildTree(baseURL)
	if err != nil {
		panic(err)
	}

	fmt.Println("Repository structure:")
	printTree(tree, "", true)

}
