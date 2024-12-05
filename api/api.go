package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/go-redis/redis/v8"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Repository struct {
	Redis *redis.Client
}

func NewApi(redis *redis.Client) *Repository {
	return &Repository{
		Redis: redis,
	}
}

var Repo *Repository

func NewRepo(app *Repository) *Repository {
	Repo = app
	return Repo
}

func (r *Repository) request(method, endpoint string, data []byte, params map[string]string) ([]byte, error) {
	fullUrl := fmt.Sprintf("%s/%s", os.Getenv("API_URL"), endpoint)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if params != nil {
		urlParams := url.Values{}
		for key, value := range params {
			urlParams.Add(key, value)
		}
		fullUrl = fmt.Sprintf("%s?%s", fullUrl, urlParams.Encode())
	}
	_, err := url.Parse(fullUrl)
	if err != nil {
		log.Println("Invalid URL:", err)
		return nil, err
	}
	key := fmt.Sprintf("%s_%s", method, fullUrl)
	safeKey := base64.URLEncoding.EncodeToString([]byte(key))
	cachedResponse, err := r.Redis.Get(ctx, safeKey).Result()
	if err == nil {
		fmt.Println(method, fullUrl, "cached response")
		return []byte(cachedResponse), err
	}

	var req *http.Request

	if data != nil {
		req, err = http.NewRequest(method, fullUrl, bytes.NewBuffer(data))
	} else {
		req, err = http.NewRequest(method, fullUrl, nil)
	}

	if err != nil {
		log.Println("Cannot create request", err)
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("api-auth-id", os.Getenv("API_ID"))
	req.Header.Add("api-auth-signature", createHmacSha256(os.Getenv("API_KEY"), params))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Failed to close response body", err)
		}
	}(resp.Body)
	bodyBytes, _ := io.ReadAll(resp.Body)
	err = r.Redis.Set(ctx, safeKey, string(bodyBytes), time.Hour*24).Err()
	if err != nil {
		log.Println("Failed to cache response in Redis:", err)
	}
	return bodyBytes, nil
}

func (r *Repository) Get(endpoint string, params map[string]string) ([]byte, error) {
	return r.request("GET", endpoint, nil, params)
}

func (r *Repository) Post(endpoint string, data []byte, params map[string]string) ([]byte, error) {
	return r.request("POST", endpoint, data, params)
}

func (r *Repository) Put(endpoint string, data []byte, params map[string]string) ([]byte, error) {
	return r.request("PUT", endpoint, data, params)
}

func (r *Repository) Delete(endpoint string, params map[string]string) ([]byte, error) {
	return r.request("DELETE", endpoint, nil, params)
}

func createHmacSha256(apiKey string, params map[string]string) string {
	h := hmac.New(sha256.New, []byte(apiKey))

	var paramStrings []string
	if params != nil {
		for key, value := range params {
			paramStrings = append(paramStrings, fmt.Sprintf("%s=%s", key, value))
		}
	}

	h.Write([]byte(strings.Join(paramStrings, "&")))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
