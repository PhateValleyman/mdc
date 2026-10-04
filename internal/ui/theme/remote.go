package theme

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

const githubContentsURL = "https://api.github.com/repos/kooler/MiddayCommander/contents/themes"

var httpClient = &http.Client{Timeout: 10 * time.Second}

// githubEntry represents a file entry from the GitHub Contents API.
type githubEntry struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
}

// FetchRemoteThemes fetches theme files from the GitHub repository.
// localKeys contains keys of themes already available locally; those are skipped.
// Returns nil on any network or parse error
func FetchRemoteThemes(localKeys map[string]bool) []AvailableTheme {
	entries, err := listGithubThemes()
	if err != nil {
		return nil
	}

	var result []AvailableTheme
	for _, e := range entries {
		if !strings.HasSuffix(e.Name, ".toml") || e.DownloadURL == "" {
			continue
		}
		key := strings.TrimSuffix(e.Name, ".toml")
		if localKeys[key] {
			continue
		}
		data, err := fetchURL(e.DownloadURL)
		if err != nil {
			continue
		}
		at, err := ParseTOML(key, data)
		if err != nil {
			continue
		}
		result = append(result, at)
	}
	return result
}

func listGithubThemes() ([]githubEntry, error) {
	req, err := http.NewRequest("GET", githubContentsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	var entries []githubEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func fetchURL(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
