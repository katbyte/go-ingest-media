package radarr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Client is a minimal Radarr API client
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// NewClient creates a new Radarr client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTP:    &http.Client{},
	}
}

// RootFolder represents a root directory in Radarr
type RootFolder struct {
	ID              int              `json:"id"`
	Path            string           `json:"path"`
	UnmappedFolders []UnmappedFolder `json:"unmappedFolders"`
}

// UnmappedFolder represents a folder on disk not associated with a movie in Radarr
type UnmappedFolder struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Movie represents a movie in the Radarr database
type Movie struct {
	ID             int    `json:"id"`
	Title          string `json:"title"`
	Year           int    `json:"year"`
	Path           string `json:"path"`
	FolderName     string `json:"folderName"`
	RootFolderPath string `json:"rootFolderPath"`
	HasFile        bool   `json:"hasFile"`
	TmdbId         int    `json:"tmdbId"`
}

func (c *Client) get(endpoint string, params url.Values, v interface{}) error {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid radarr url: %w", err)
	}

	u.Path = "/api/v3" + endpoint
	if params != nil {
		u.RawQuery = params.Encode()
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Api-Key", c.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("radarr api returned status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// GetRootFolders gets all configured root folders (includes unmapped folders)
func (c *Client) GetRootFolders() ([]RootFolder, error) {
	var folders []RootFolder
	if err := c.get("/rootfolder", nil, &folders); err != nil {
		return nil, err
	}
	return folders, nil
}

// GetMovies gets all movies in the Radarr database
func (c *Client) GetMovies() ([]Movie, error) {
	var movies []Movie
	if err := c.get("/movie", nil, &movies); err != nil {
		return nil, err
	}
	return movies, nil
}

// LookupMovie searches for a movie by term (uses Radarr's TMDB matching)
func (c *Client) LookupMovie(term string) ([]Movie, error) {
	params := url.Values{}
	params.Set("term", term)

	var movies []Movie
	if err := c.get("/movie/lookup", params, &movies); err != nil {
		return nil, err
	}
	return movies, nil
}
