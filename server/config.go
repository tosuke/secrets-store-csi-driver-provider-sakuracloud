package server

import (
	"encoding/json"
	"fmt"

	"github.com/go-faster/yaml"
)

type MountConfig struct {
	PodName      string `json:"csi.storage.k8s.io/pod.name,omitempty"`
	PodNamespace string `json:"csi.storage.k8s.io/pod.namespace,omitempty"`

	VaultID string  `json:"vaultID,omitempty"`
	Secrets Secrets `json:"secrets"`
}

func ParseConfig(data []byte) (*MountConfig, error) {
	var cfg MountConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mount config: %w", err)
	}
	return &cfg, nil
}

type Secrets []Secret

var _ json.Unmarshaler = (*Secrets)(nil)

func (m *Secrets) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if err := yaml.Unmarshal([]byte(raw), m); err != nil {
		return err
	}

	return nil
}

type Secret struct {
	VaultID string `yaml:"vaultID,omitempty"`
	Name    string `yaml:"name"`
	Version *int   `yaml:"version,omitempty"`
}
