package titles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"goforum/internal/config"
	C "goforum/internal/constants"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

const (
	titlesURL   = "https://raw.githubusercontent.com/birabittoh/xtitles/refs/heads/main/titles.filtered.min.json"
	picturesURL = "https://raw.githubusercontent.com/birabittoh/xtitles/refs/heads/main/titles/%s/%s.png"
	filename    = "titles.json"
)

type Title struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Pictures []string `json:"pictures"`
}

type TitlesService struct {
	data      []Title
	names     []string
	total     int
	validPics map[string]bool
}

func getKey(id, pic string) string {
	return id + ":" + pic
}

func (s *TitlesService) afterLoad() {
	s.total = len(s.data)

	s.names = make([]string, s.total)
	s.validPics = make(map[string]bool)
	for i, title := range s.data {
		s.names[i] = title.Name

		for _, pic := range title.Pictures {
			s.validPics[getKey(title.ID, pic)] = true
		}
	}
}

func (s *TitlesService) LoadTitles(config *config.Config) error {
	defer s.afterLoad()

	cacheFile := filepath.Join(config.DataDir, filename)
	if data, err := os.ReadFile(cacheFile); err == nil {
		if err := json.Unmarshal(data, &s.data); err == nil {
			return nil
		}
	}

	r, err := http.Get(titlesURL)
	if err != nil {
		return fmt.Errorf("failed to fetch titles: %w", err)
	}
	defer r.Body.Close()

	var buf bytes.Buffer
	body := io.TeeReader(r.Body, &buf)

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch titles: status %d", r.StatusCode)
	}

	if err := json.NewDecoder(body).Decode(&s.data); err != nil {
		return fmt.Errorf("failed to decode titles JSON: %w", err)
	}

	// Save to cache file
	if err := os.MkdirAll(config.DataDir, 0755); err == nil {
		os.WriteFile(cacheFile, buf.Bytes(), 0644)
	}

	return nil
}

func New(cfg *config.Config) (*TitlesService, error) {
	s := &TitlesService{}

	if err := s.LoadTitles(cfg); err != nil {
		return nil, fmt.Errorf("failed to load titles: %w", err)
	}

	return s, nil
}

func (s *TitlesService) GetTitles(page, limit int, query string) *C.PaginatedResponse {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	var titles []Title
	var total int
	var pages int

	if query != "" {
		matches := fuzzy.RankFindNormalizedFold(query, s.names)
		sort.Slice(matches, matches.Less)

		total = len(matches)
		end := min(offset+limit, total)
		if offset > total {
			offset = total
		}

		for i := offset; i < end; i++ {
			if i < total {
				titles = append(titles, s.data[matches[i].OriginalIndex])
			}
		}
		pages = (total + limit - 1) / limit
	} else {
		total = s.total
		end := min(offset+limit, s.total)
		if offset > s.total {
			offset = s.total
		}
		titles = s.data[offset:end]
		pages = (s.total + limit - 1) / limit
	}

	return &C.PaginatedResponse{
		Items:  titles,
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Page:   page,
		Pages:  pages,
	}
}

func (s *TitlesService) ServePicture(c *gin.Context) {
	id := c.Param("id")
	picture := strings.TrimSuffix(c.Param("picture"), ".png")

	if !s.validPics[getKey(id, picture)] {
		c.JSON(http.StatusNotFound, gin.H{"error": "Picture not found"})
		return
	}

	url := fmt.Sprintf(picturesURL, strings.ToLower(id), strings.ToLower(picture))
	c.Redirect(http.StatusMovedPermanently, url)
}
