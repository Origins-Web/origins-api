package store

import (
	"encoding/json"
	"sync"

	"origins-api/models"
)

type Store struct {
	state models.ProjectState
	mu    sync.RWMutex
}

func New() *Store {
	return &Store{
		state: models.ProjectState{
			Commits:      []models.Commit{},
			Deployments:  []models.Deployment{},
			PullRequests: []models.PullRequest{},
		},
	}
}

func (s *Store) AddCommit(c models.Commit) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Commits = append([]models.Commit{c}, s.state.Commits...)
	// Limit history to 50
	if len(s.state.Commits) > 50 {
		s.state.Commits = s.state.Commits[:50]
	}
}

func (s *Store) AddDeployment(d models.Deployment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Deployments = append([]models.Deployment{d}, s.state.Deployments...)
}

func (s *Store) GetStateJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.Marshal(s.state)
}

// Helper to get raw struct if needed (read-only recommended)
func (s *Store) GetState() models.ProjectState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}