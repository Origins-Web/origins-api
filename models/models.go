package models

type Commit struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	Timestamp string `json:"timestamp"`
	URL       string `json:"url"`
}

type Deployment struct {
	ID        string `json:"id"`
	Status    string `json:"status"` // "BUILDING", "READY", "ERROR"
	URL       string `json:"url"`
	Timestamp string `json:"timestamp"`
}

type PullRequest struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"` // "open", "closed", "merged"
	Author string `json:"author"`
}

type ProjectState struct {
	Commits      []Commit      `json:"commits"`
	Deployments  []Deployment  `json:"deployments"`
	PullRequests []PullRequest `json:"pull_requests"`
}