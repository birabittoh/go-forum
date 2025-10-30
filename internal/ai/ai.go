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

const (
	debugEndpoint   = "/debug"
	enqueueEndpoint = "/enqueue"
	queueEndpoint   = "/queue"
)

type EnqueueRequest struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	CallbackURL string `json:"callback_url"`
}

type EnqueueResponse struct {
	UUID string `json:"uuid"`
}

type QueueStatus struct {
	IsProcessing bool     `json:"is_processing"`
	QueuedIDs    []string `json:"queued_ids"`
}

type CallbackPayload struct {
	UUID             string  `json:"uuid"`
	Prediction       string  `json:"prediction"`
	Confidence       float64 `json:"confidence"`
	HumanProbability float64 `json:"human_prob"`
	AIProbability    float64 `json:"ai_prob"`
}

type AIService struct {
	config      *config.Config
	queue       map[string]uint // maps uuid to postID
	client      *http.Client
	callbackURL string
}

func New(cfg *config.Config, callbackURL string) *AIService {
	callbackBase := cfg.AICallbackURL
	if callbackBase == "" {
		callbackBase = cfg.SiteURL
	}

	s := &AIService{
		config:      cfg,
		queue:       make(map[string]uint),
		client:      &http.Client{},
		callbackURL: callbackBase + callbackURL,
	}

	if cfg.AIDetectionURL != "" && callbackBase != "" {
		cfg.AIEnabled = true
	}

	go func() {
		_, err := s.Queue()
		if err != nil {
			cfg.AIEnabled = false
		}
	}()

	return s
}

func doRequest[T any](s *AIService, method, url string, body any) (*T, error) {
	var reqBody *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = &bytes.Buffer{}
	}

	req, err := http.NewRequest(method, s.config.AIDetectionURL+url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	var result T
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *AIService) EnqueueDetection(p *models.Post) (err error) {
	if !s.config.AIEnabled {
		return nil
	}

	d := EnqueueRequest{
		ID:          strconv.FormatUint(uint64(p.ID), 10),
		Content:     p.Content,
		CallbackURL: s.callbackURL,
	}

	resp, err := doRequest[EnqueueResponse](s, "POST", enqueueEndpoint, d)
	if err != nil {
		log.Printf("Failed to enqueue detection: %v\n", err)
		return err
	}

	s.queue[resp.UUID] = p.ID
	log.Printf("Enqueued post ID %d with UUID %s\n", p.ID, resp.UUID)
	return
}

func (s *AIService) Queue() (*QueueStatus, error) {
	if !s.config.AIEnabled {
		return &QueueStatus{IsProcessing: false, QueuedIDs: []string{}}, nil
	}
	return doRequest[QueueStatus](s, "GET", queueEndpoint, nil)
}

func (s *AIService) Debug() (*map[string]any, error) {
	if !s.config.AIEnabled {
		return nil, nil
	}
	return doRequest[map[string]any](s, "POST", debugEndpoint, nil)
}

func (s *AIService) GetPostID(uuid string) (uint, bool) {
	id, ok := s.queue[uuid]
	delete(s.queue, uuid)
	return id, ok
}
