package main

import (
	"github.com/go-redis/redis/v8"
	"os"
)

type Repository struct {
	RedisClient  *redis.Client
	UnleashedApi ApiConfig
}

type ApiConfig struct {
	ApiUrl string
	ApiKey string
	ApiID  string
}

func NewRedisCache() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"), // no password set
		DB:       0,                           // use default DB
	})
}

func NewUnleashed(apiConfig ApiConfig, redisClient *redis.Client) *Repository {

	return &Repository{
		RedisClient:  redisClient,
		UnleashedApi: apiConfig,
	}
}
