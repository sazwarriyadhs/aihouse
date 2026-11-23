package queue

import (
	"fmt"
	"os"
	"strconv"
)

func LoadConfig() (Config, error) {
	db := 0
	if value := os.Getenv("REDIS_DB"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			return Config{}, fmt.Errorf("invalid REDIS_DB")
		}
		db = parsed
	}

	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "56379"
	}

	return Config{
		Host:     envOrRedis("REDIS_HOST", "127.0.0.1"),
		Port:     port,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	}, nil
}

func envOrRedis(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
