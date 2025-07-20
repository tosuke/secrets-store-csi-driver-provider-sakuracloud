package config_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tosuke/secrets-store-csi-driver-provider-sakuracloud/config"
)

func TestParse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want *config.MountRequest
	}{
		{
			name: "basic",
			in: `{
				"csi.storage.k8s.io/pod.name": "test-pod",
				"csi.storage.k8s.io/pod.namespace": "default",
				"vaultID": "1234",
				"secrets": "- name: secret1\n- vaultID: \"5678\"\n  name: secret2"
			}`,
			want: &config.MountRequest{
				PodName:      "test-pod",
				PodNamespace: "default",

				VaultID: "1234",
				Secrets: config.Secrets{
					{VaultID: "1234", Name: "secret1"},
					{VaultID: "5678", Name: "secret2"},
				},
			},
		},
		{
			name: "with version",
			in: `{
				"csi.storage.k8s.io/pod.name": "test-pod",
				"csi.storage.k8s.io/pod.namespace": "default",
				"vaultID": "1234",
				"secrets": "- name: secret1\n  version: 1\n- vaultID: \"5678\"\n  name: secret2\n  version: 2"
			}`,
			want: &config.MountRequest{
				PodName:      "test-pod",
				PodNamespace: "default",

				VaultID: "1234",
				Secrets: config.Secrets{
					{VaultID: "1234", Name: "secret1", Version: ptr(1)},
					{VaultID: "5678", Name: "secret2", Version: ptr(2)},
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := config.ParseMountRequest(tt.in)
			if err != nil {
				t.Fatalf("parse config: %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unexpected diff (-want +got):\n%s", diff)
			}
		})
	}
}

func ptr[T any](x T) *T {
	return &x
}

func TestSecretID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   config.Secret
		want string
	}{
		{
			name: "with version",
			in: config.Secret{
				VaultID: "1234",
				Name:    "secret1",
				Version: ptr(1),
			},
			want: "vaults/1234/secrets/secret1/versions/1",
		},
		{
			name: "without version",
			in: config.Secret{
				VaultID: "5678",
				Name:    "secret2",
			},
			want: "vaults/5678/secrets/secret2/versions/latest",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.in.ID(); got != tt.want {
				t.Errorf("ID() = %v, want %v", got, tt.want)
			}
		})
	}
}
