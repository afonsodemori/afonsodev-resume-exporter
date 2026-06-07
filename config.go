package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Cloudflare Cloudflare `json:"cloudflare"`
	Documents  Documents  `json:"documents"`
}

type Cloudflare struct {
	AccountID string `json:"account_id"`
	R2        R2     `json:"r2"`
}

type R2 struct {
	PublicAPI       string `json:"public_api"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

type Documents struct {
	List    []Document `json:"list"`
	Formats []string   `json:"formats"`
}

type Document struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config %q: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config %q: %w", path, err)
	}

	return cfg, nil
}
