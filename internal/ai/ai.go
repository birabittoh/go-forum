package ai

import (
	"bytes"
	"encoding/json"
	"goforum/internal/config"
	"goforum/internal/models"
	"log"
	"net/http"
	"strconv"
)

type EnqueueRequest struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	CallbackURL string `json:"callback_url"`
}

type EnqueueResponse struct {
	UUID string `json:"uuid"`
}

type CallbackPayload struct {
	UUID             string  `json:"uuid"`
	Prediction       string  `json:"prediction"`
	Confidence       float64 `json:"confidence"`
	HumanProbability float64 `json:"human_prob"`
	AIProbability    float64 `json:"ai_prob"`
}

type AIService struct {
	config *config.Config
	queue  map[string]uint // maps uuid to postID
	client *http.Client
}

func New(cfg *config.Config) *AIService {
	return &AIService{
		config: cfg,
		queue:  make(map[string]uint),
		client: &http.Client{},
	}
}

func (s *AIService) EnqueueDetection(p *models.Post) (err error) {
	if s.config.AIDetectionURL == "" {
		return nil // AI detection not configured
	}

	d := EnqueueRequest{
		ID:          strconv.FormatUint(uint64(p.ID), 10),
		Content:     p.Content,
		CallbackURL: s.config.SiteURL + "/aide/callback",
	}

	reqBody, err := json.Marshal(d)
	if err != nil {
		log.Printf("Failed to marshal detection request: %v\n", err)
		return
	}

	req, err := http.NewRequest("POST", s.config.AIDetectionURL+"/enqueue", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("Failed to create request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("Failed to send request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Non-OK HTTP status: %s\n", resp.Status)
		return
	}

	var er EnqueueResponse
	err = json.NewDecoder(resp.Body).Decode(&er)
	if err != nil {
		log.Printf("Failed to decode response: %v\n", err)
		return
	}

	s.queue[er.UUID] = p.ID
	log.Printf("Enqueued post ID %d with UUID %s\n", p.ID, er.UUID)
	return
}

func (s *AIService) GetPostID(uuid string) (uint, bool) {
	id, ok := s.queue[uuid]
	delete(s.queue, uuid)
	return id, ok
}
