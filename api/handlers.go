package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"origins-api/models"
)

// --- SSE Broker (Updated for Redis PubSub) ---

type Broker struct {
	newClients     chan chan []byte
	closingClients chan chan []byte
	clients        map[chan []byte]bool
}

func NewBroker() *Broker {
	return &Broker{
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]bool),
	}
}

// Listen now subscribes to Redis "events" channel
func (s *Server) listenToRedis() {
	pubsub := s.store.Subscribe()
	defer pubsub.Close()
	ch := pubsub.Channel()

	for {
		select {
		// Handle new/closing HTTP clients
		case clientChan := <-s.broker.newClients:
			s.broker.clients[clientChan] = true
			log.Printf("Client connected. Total: %d", len(s.broker.clients))
		
		case clientChan := <-s.broker.closingClients:
			delete(s.broker.clients, clientChan)
			log.Printf("Client disconnected. Total: %d", len(s.broker.clients))

		// Handle incoming message from Redis
		case msg := <-ch:
			// Broadcast to all connected SSE clients
			for clientChan := range s.broker.clients {
				clientChan <- []byte(msg.Payload)
			}
		}
	}
}

// --- HTTP Handlers ---

func (s *Server) StreamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	messageChan := make(chan []byte)
	s.broker.newClients <- messageChan

	defer func() {
		s.broker.closingClients <- messageChan
	}()

	// Send initial state from Redis
	state, err := s.store.GetFullState()
	if err == nil {
		initialData, _ := json.Marshal(state)
		fmt.Fprintf(w, "data: %s\n\n", initialData)
		w.(http.Flusher).Flush()
	}

	// Keep connection open
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-messageChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			w.(http.Flusher).Flush()
		}
	}
}

func (s *Server) StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	state, err := s.store.GetFullState()
	if err != nil {
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(state)
}

func (s *Server) GithubPushHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Read Error", http.StatusInternalServerError)
		return
	}

	if s.cfg.GithubSecret != "" {
		signature := r.Header.Get("X-Hub-Signature-256")
		if !checkSignature(body, signature, s.cfg.GithubSecret) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	var payload struct {
		Ref        string `json:"ref"`
		HeadCommit struct {
			ID        string `json:"id"`
			Message   string `json:"message"`
			Timestamp string `json:"timestamp"`
			Author    struct {
				Name string `json:"name"`
			} `json:"author"`
			URL string `json:"url"`
		} `json:"head_commit"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	commit := models.Commit{
		ID:        payload.HeadCommit.ID,
		Message:   payload.HeadCommit.Message,
		Author:    payload.HeadCommit.Author.Name,
		Timestamp: time.Now().Format(time.RFC3339),
		URL:       payload.HeadCommit.URL,
	}

	// Save to Redis (Publishes event automatically)
	if err := s.store.AddCommit(commit); err != nil {
		log.Printf("Redis Error: %v", err)
		http.Error(w, "Storage Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) VercelDeployHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ID string `json:"id"`
		State string `json:"type"` 
		Payload struct {
			Deployment struct {
				URL string `json:"url"`
			} `json:"deployment"`
		} `json:"payload"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	status := "BUILDING"
	if payload.State == "deployment.succeeded" {
		status = "READY"
	} else if payload.State == "deployment.error" {
		status = "ERROR"
	}

	deploy := models.Deployment{
		ID:        payload.ID,
		Status:    status,
		URL:       "https://" + payload.Payload.Deployment.URL,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := s.store.AddDeployment(deploy); err != nil {
		log.Printf("Redis Error: %v", err)
		http.Error(w, "Storage Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func checkSignature(body []byte, signature, secret string) bool {
	if signature == "" { return false }
	parts := strings.SplitN(signature, "=", 2)
	if len(parts) != 2 || parts[0] != "sha256" { return false }
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	incomingMAC, _ := hex.DecodeString(parts[1])
	return hmac.Equal(incomingMAC, expectedMAC)
}