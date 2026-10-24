// Package reference provides an in-process HTTP server that implements the finalized
// GeeGooSignal lab-api contract (PR #2 / feat/lab-api). Used for Agent-side contract
// integration when the Signal repository is not available in the environment.
package reference

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DefaultAPIKey matches reference server Bearer for integration tests.
const DefaultAPIKey = "test-lab-key"

// Server is a minimal lab-api + in-process worker (max running concurrency, job timeout).
type Server struct {
	APIKey     string
	MaxRunning int
	JobTimeout time.Duration
	HTTP       *http.Server

	mu      sync.Mutex
	seq     int
	jobs    map[string]*jobRecord
	running int
}

type jobRecord struct {
	ID        string
	Status    string
	Error     string
	Request   createBody
	Result    *resultBody
	created   time.Time
	started   time.Time
}

type createBody struct {
	Code       string `json:"code"`
	Frequency  string `json:"frequency"`
	MonthsBack int    `json:"months_back"`
	AsOf       string `json:"as_of"`
}

type resultBody struct {
	Bars []map[string]any `json:"bars"`
}

type jobResponse struct {
	ID     string       `json:"id"`
	Status string       `json:"status"`
	Error  string       `json:"error,omitempty"`
	Result *resultBody  `json:"result,omitempty"`
}

// NewServer returns a configured reference lab-api listening on localhost:0 when started.
func NewServer() *Server {
	s := &Server{
		APIKey:     DefaultAPIKey,
		MaxRunning: 4,
		JobTimeout: 5 * time.Minute,
		jobs:       map[string]*jobRecord{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /v1/lab/kline/jobs", s.handleCreate)
	mux.HandleFunc("GET /v1/lab/jobs/", s.handleGet)
	s.HTTP = &http.Server{Handler: mux}
	return s
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeErr(w, http.StatusUnauthorized, "missing or invalid authorization")
		return
	}
	var body createBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	body.Code = strings.TrimSpace(body.Code)
	body.Frequency = strings.TrimSpace(body.Frequency)
	if body.Code == "" || body.Frequency == "" || body.MonthsBack <= 0 {
		writeErr(w, http.StatusBadRequest, "code, frequency, months_back required")
		return
	}
	switch body.Frequency {
	case "5m", "60m", "daily", "weekly":
	default:
		writeErr(w, http.StatusBadRequest, "unsupported frequency")
		return
	}
	s.mu.Lock()
	s.seq++
	id := "labjob-" + time.Now().Format("150405") + "-" + itoa(s.seq)
	rec := &jobRecord{ID: id, Status: "pending", Request: body, created: time.Now()}
	s.jobs[id] = rec
	s.mu.Unlock()
	s.scheduleWorker(id)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"job_id": id})
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeErr(w, http.StatusUnauthorized, "missing or invalid authorization")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/lab/jobs/")
	id = strings.TrimSpace(id)
	s.mu.Lock()
	rec, ok := s.jobs[id]
	s.mu.Unlock()
	if !ok {
		writeErr(w, http.StatusNotFound, "job not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jobResponse{
		ID: rec.ID, Status: rec.Status, Error: rec.Error, Result: rec.Result,
	})
}

func (s *Server) scheduleWorker(id string) {
	go func() {
		time.Sleep(5 * time.Millisecond)
		s.mu.Lock()
		rec := s.jobs[id]
		if rec == nil {
			s.mu.Unlock()
			return
		}
		for s.running >= s.maxRunning() {
			s.mu.Unlock()
			time.Sleep(2 * time.Millisecond)
			s.mu.Lock()
			rec = s.jobs[id]
			if rec == nil || rec.Status != "pending" {
				s.mu.Unlock()
				return
			}
		}
		rec.Status = "running"
		rec.started = time.Now()
		s.running++
		req := rec.Request
		s.mu.Unlock()

		timeout := s.JobTimeout
		if timeout <= 0 {
			timeout = 5 * time.Minute
		}
		done := make(chan struct{})
		go func() {
			time.Sleep(15 * time.Millisecond)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(timeout):
		}

		s.mu.Lock()
		defer s.mu.Unlock()
		s.running--
		rec = s.jobs[id]
		if rec == nil {
			return
		}
		if strings.EqualFold(req.Code, "FAIL.HK") {
			rec.Status = "failed"
			rec.Error = "simulated worker failure"
			return
		}
		bars := synthesizeBars(req)
		rec.Status = "completed"
		out := make([]map[string]any, len(bars))
		copy(out, bars)
		rec.Result = &resultBody{Bars: out}
	}()
}

func (s *Server) maxRunning() int {
	if s.MaxRunning <= 0 {
		return 4
	}
	return s.MaxRunning
}

func (s *Server) authOK(r *http.Request) bool {
	key := s.APIKey
	if key == "" {
		key = DefaultAPIKey
	}
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(h, "Bearer ") {
		return false
	}
	return strings.TrimSpace(strings.TrimPrefix(h, "Bearer ")) == key
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func synthesizeBars(req createBody) []map[string]any {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	n := 200
	switch req.Frequency {
	case "weekly":
		n = 30
	case "daily":
		n = 80
	case "60m":
		n = 150
	}
	end := time.Now().In(loc)
	if req.AsOf != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", req.AsOf, loc); err == nil {
			end = t
		}
	}
	var bars []map[string]any
	for i := 0; i < n; i++ {
		ts := end.Add(-time.Duration(n-i) * time.Hour)
		if req.Frequency == "daily" {
			ts = end.AddDate(0, 0, -(n - i))
		}
		if req.Frequency == "weekly" {
			ts = end.AddDate(0, 0, -7*(n-i))
		}
		if req.AsOf != "" && ts.After(end) {
			continue
		}
		c := 100.0 + float64(i)*0.1
		bars = append(bars, map[string]any{
			"time":   ts.Format("2006-01-02 15:04:05"),
			"open":   c - 0.05,
			"high":   c + 1,
			"low":    c - 1,
			"close":  c,
			"volume": 1000.0 + float64(i),
		})
	}
	if req.AsOf != "" {
		filtered := bars[:0]
		for _, b := range bars {
			ts, _ := time.ParseInLocation("2006-01-02 15:04:05", b["time"].(string), loc)
			if !ts.After(end) {
				filtered = append(filtered, b)
			}
		}
		bars = filtered
	}
	if len(bars) == 0 {
		bars = []map[string]any{}
	}
	return bars
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}
