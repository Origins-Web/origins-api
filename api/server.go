package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"origins-api/models"
)

type Store struct {
	client *redis.Client
	ctx    context.Context
}

func New(redisURL string) (*Store, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}

	client := redis.NewClient(opt)
	
	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Store{
		client: client,
		ctx:    ctx,
	}, nil
}

func (s *Store) AddCommit(c models.Commit) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	pipe := s.client.Pipeline()
	// Push to head of list
	pipe.LPush(s.ctx, "commits", data)
	// Keep only last 50
	pipe.LTrim(s.ctx, "commits", 0, 49)
	// Publish event for real-time subscribers
	pipe.Publish(s.ctx, "events", data)
	
	_, err = pipe.Exec(s.ctx)
	return err
}

func (s *Store) AddDeployment(d models.Deployment) error {
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}

	pipe := s.client.Pipeline()
	pipe.LPush(s.ctx, "deployments", data)
	pipe.LTrim(s.ctx, "deployments", 0, 49)
	pipe.Publish(s.ctx, "events", data)
	
	_, err = pipe.Exec(s.ctx)
	return err
}

// GetFullState reconstructs the dashboard state from Redis
func (s *Store) GetFullState() (*models.ProjectState, error) {
	// Fetch Commits
	commitData, err := s.client.LRange(s.ctx, "commits", 0, 49).Result()
	if err != nil {
		return nil, err
	}

	var commits []models.Commit
	for _, raw := range commitData {
		var c models.Commit
		json.Unmarshal([]byte(raw), &c)
		commits = append(commits, c)
	}

	// Fetch Deployments
	deployData, err := s.client.LRange(s.ctx, "deployments", 0, 49).Result()
	if err != nil {
		return nil, err
	}

	var deploys []models.Deployment
	for _, raw := range deployData {
		var d models.Deployment
		json.Unmarshal([]byte(raw), &d)
		deploys = append(deploys, d)
	}

	return &models.ProjectState{
		Commits:     commits,
		Deployments: deploys,
	}, nil
}

// Subscribe allows the SSE broker to listen for new Redis events
func (s *Store) Subscribe() *redis.PubSub {
	return s.client.Subscribe(s.ctx, "events")
}