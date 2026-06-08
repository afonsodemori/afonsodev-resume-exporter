package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
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
	Dirs    Dirs       `json:"dirs"`
	List    []Document `json:"list"`
	Formats []string   `json:"formats"`
}

type Dirs struct {
	Output  string `json:"output"`
	Archive string `json:"archive"`
}

type Document struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	dirs Dirs   // injected after parse
}

func (d Document) NewPath(format string) string {
	return filepath.Join(d.dirs.Output, fmt.Sprintf("%s-new.%s", d.Name, format))
}

func (d Document) CurrentPath(format string) string {
	return filepath.Join(d.dirs.Output, fmt.Sprintf("%s.%s", d.Name, format))
}

func (d Document) ArchivePath(format string) string {
	return filepath.Join(d.dirs.Archive, fmt.Sprintf("%s-%s.%s", d.Name, time.Now().Format("060102-1504"), format))
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

	for i := range cfg.Documents.List {
		cfg.Documents.List[i].dirs = cfg.Documents.Dirs
	}

	return cfg, nil
}
