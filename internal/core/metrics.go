package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type UsageEvent struct {
	Name       string            `json:"name"`
	Timestamp  string            `json:"timestamp"`
	Properties map[string]string `json:"properties,omitempty"`
}

const (
	metricsDirName  = "metrics"
	metricsFileName = "events.jsonl"
)

func TrackEvent(name string, properties map[string]string) error {
	if name == "" {
		return fmt.Errorf("event name is required")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to resolve home directory: %w", err)
	}

	metricsDir := filepath.Join(home, ApsHomeDir, metricsDirName)
	if err := os.MkdirAll(metricsDir, 0755); err != nil {
		return fmt.Errorf("failed to create metrics directory: %w", err)
	}

	event := UsageEvent{
		Name:       name,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Properties: properties,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	metricsPath := filepath.Join(metricsDir, metricsFileName)
	file, err := os.OpenFile(metricsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open metrics file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write metrics event: %w", err)
	}

	return nil
}
