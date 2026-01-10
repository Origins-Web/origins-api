package config

import "os"

type Config struct {
	Port         string
	GithubSecret string
	VercelSecret string
	RedisURL     string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	}
	if port[0] != ':' {
		port = ":" + port
	}

	return &Config{
		Port:         port,
		GithubSecret: os.Getenv("GITHUB_SECRET"),
		VercelSecret: os.Getenv("VERCEL_SECRET"),
		RedisURL:     os.Getenv("REDIS_URL"), // e.g., redis://user:pass@host:port/0
	}
}