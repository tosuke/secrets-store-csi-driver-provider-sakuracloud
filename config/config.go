package config

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-faster/yaml"
)

func ParseMountRequest(data string) (*MountRequest, error) {
	var cfg MountRequest
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mount config: %w", err)
	}

	var errs []error
	for i, secret := range cfg.Secrets {
		if secret.Name == "" {
			errs = append(errs, fmt.Errorf("secrets[%d].name is required", i))
		}
		if secret.VaultID == "" {
			if cfg.VaultID != "" {
				cfg.Secrets[i].VaultID = cfg.VaultID
			} else {
				errs = append(errs, fmt.Errorf("secrets[%d].vaultID is required", i))
			}
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return &cfg, nil
}

type MountRequest struct {
	PodName      string `json:"csi.storage.k8s.io/pod.name,omitempty"`
	PodNamespace string `json:"csi.storage.k8s.io/pod.namespace,omitempty"`

	// VaultID is the default vault ID to use for the secrets.
	VaultID string  `json:"vaultID,omitempty"`
	Secrets Secrets `json:"secrets"`
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
	// VaultID is the ID of the vault where the secret is stored.
	VaultID string `json:"vaultID" yaml:"vaultID,omitempty"`
	// Name is the name of the secret.
	Name string `json:"name" yaml:"name"`
	// Version is the version of the secret. If not specified, the latest version will be used.
	Version *int `json:"version,omitempty" yaml:"version,omitempty"`
}
