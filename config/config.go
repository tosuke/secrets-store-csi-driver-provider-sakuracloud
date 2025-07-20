package config

import (
	"encoding/json"
	"fmt"

	"github.com/go-faster/yaml"
)

type MountRequest struct {
	PodName      string `json:"csi.storage.k8s.io/pod.name,omitempty"`
	PodNamespace string `json:"csi.storage.k8s.io/pod.namespace,omitempty"`

	VaultID string  `json:"vaultID,omitempty"`
	Secrets Secrets `json:"secrets"`
}

func ParseMountRequest(data string) (*MountRequest, error) {
	var cfg MountRequest
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
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
	VaultID string `json:"vaultID" yaml:"vaultID,omitempty"`
	Name    string `json:"name" yaml:"name"`
	Version *int   `json:"version,omitempty" yaml:"version,omitempty"`
}
