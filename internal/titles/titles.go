package titles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"goforum/internal/config"
	C "goforum/internal/constants"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

const (
	titlesURL     = "https://raw.githubusercontent.com/birabittoh/xtitles/refs/heads/main/"
	picturesExt   = ".png"
	picturesURL   = "titles/%s/%s" + picturesExt
	submoduleDir  = "xtitles"
	submoduleFile = "titles.filtered.json"
)

type Title struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Pictures []string `json:"pictures"`
}

type TitlesService struct {
	config    *config.Config
	data      []Title
	names     []string
	total     int
	validPics map[string]bool
	dataMap   map[string]Title
}

func getKey(id, pic string) string { // expects lowercase id and pic
	return id + "/" + pic
}

func (s *TitlesService) LoadTitles() error {
	defer func() {
		slices.SortFunc(s.data, func(a, b Title) int {
			return strings.Compare(b.ID, a.ID) // reverse order
		})

		s.total = len(s.data)
		s.names = make([]string, s.total)
		s.dataMap = make(map[string]Title, s.total)
		s.validPics = make(map[string]bool)

		for i, title := range s.data {
			s.names[i] = title.Name
			lowerID := strings.ToLower(title.ID)
			s.dataMap[lowerID] = title

			for _, pic := range title.Pictures {
				s.validPics[getKey(lowerID, strings.ToLower(pic))] = true
			}
		}
	}()

	submoduleFile := filepath.Join(submoduleDir, submoduleFile)

	var cacheFile string
	_, err := os.Stat(submoduleFile)
	if err == nil {
		s.config.LocalTitles = true
		cacheFile = submoduleFile
		log.Println("Using local xtitles submodule")
	} else {
		s.config.LocalTitles = false
		cacheFile = filepath.Join(s.config.DataDir, submoduleFile)
		log.Println("Using xtitles from GitHub")
	}

	if data, err := os.ReadFile(cacheFile); err == nil {
		if err := json.Unmarshal(data, &s.data); err == nil {
			return nil
		}
	}

	r, err := http.Get(titlesURL + submoduleFile)
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

	if err := os.MkdirAll(s.config.DataDir, 0755); err == nil {
		os.WriteFile(cacheFile, buf.Bytes(), 0644)
	}

	return nil
}

func New(cfg *config.Config) (*TitlesService, error) {
	s := &TitlesService{config: cfg}
	return s, s.LoadTitles()
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

func (s *TitlesService) ValidatePicture(picture string) bool {
	return s.validPics[strings.ToLower(picture)]
}

func (s *TitlesService) ServePicture(c *gin.Context) {
	id := strings.ToLower(c.Param("id"))
	picture := strings.ToLower(strings.TrimSuffix(c.Param("picture"), picturesExt))

	if !s.validPics[getKey(id, picture)] {
		c.JSON(http.StatusNotFound, gin.H{"error": "Picture not found"})
		return
	}

	pictureURL := fmt.Sprintf(picturesURL, id, picture)

	if s.config.LocalTitles {
		c.File(filepath.Join(submoduleDir, pictureURL))
		return
	}

	c.Redirect(http.StatusFound, titlesURL+pictureURL)
}
