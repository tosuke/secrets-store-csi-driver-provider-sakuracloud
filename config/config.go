package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

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
		if err := validatePath(secret.Path); err != nil {
			errs = append(errs, fmt.Errorf("secrets[%d].path is invalid: %w", i, err))
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
	// Path is the relative path where the secret should be mounted. If not specified, the secret name will be used.
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

func (s *Secret) ID() string {
	version := "latest"
	if s.Version != nil {
		version = strconv.Itoa(*s.Version)
	}
	return fmt.Sprintf("vaults/%s/secrets/%s/versions/%s", s.VaultID, s.Name, version)
}

// FilePath returns the path where the secret should be mounted.
// If Path is specified, it will be used; otherwise, Name will be used.
func (s *Secret) FilePath() string {
	if s.Path != "" {
		return s.Path
	}
	return s.Name
}

// validatePath validates that the path is a valid relative path within the mount directory.
func validatePath(path string) error {
	// Empty path is valid (will use secret name)
	if path == "" {
		return nil
	}

	// Path must not be absolute
	if filepath.IsAbs(path) {
		return errors.New("path must not be absolute")
	}

	// Path must not contain relative path escape sequences
	if strings.Contains(path, "../") {
		return errors.New("path must not contain relative path escape sequences like '../'")
	}

	return nil
}
