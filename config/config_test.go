package config_test

import (
	"strings"
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

	runParseTestCases(t, cases)
}

func TestParseWithPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want *config.MountRequest
	}{
		{
			name: "with path",
			in: `{
				"csi.storage.k8s.io/pod.name": "test-pod",
				"csi.storage.k8s.io/pod.namespace": "default",
				"vaultID": "1234",
				"secrets": "- name: secret1\n  path: ./config/secret1\n- vaultID: \"5678\"\n  name: secret2\n  path: subdir/secret2.txt"
			}`,
			want: &config.MountRequest{
				PodName:      "test-pod",
				PodNamespace: "default",

				VaultID: "1234",
				Secrets: config.Secrets{
					{VaultID: "1234", Name: "secret1", Path: "./config/secret1"},
					{VaultID: "5678", Name: "secret2", Path: "subdir/secret2.txt"},
				},
			},
		},
		{
			name: "with path and version",
			in: `{
				"csi.storage.k8s.io/pod.name": "test-pod",
				"csi.storage.k8s.io/pod.namespace": "default",
				"vaultID": "1234",
				"secrets": "- name: secret1\n  version: 1\n  path: config/secret1.json\n- vaultID: \"5678\"\n  name: secret2\n  version: 2\n  path: secrets/database.conf"
			}`,
			want: &config.MountRequest{
				PodName:      "test-pod",
				PodNamespace: "default",

				VaultID: "1234",
				Secrets: config.Secrets{
					{VaultID: "1234", Name: "secret1", Version: ptr(1), Path: "config/secret1.json"},
					{VaultID: "5678", Name: "secret2", Version: ptr(2), Path: "secrets/database.conf"},
				},
			},
		},
	}

	runParseTestCases(t, cases)
}

func runParseTestCases(t *testing.T, cases []struct {
	name string
	in   string
	want *config.MountRequest
},
) {
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

func TestSecretFilePath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   config.Secret
		want string
	}{
		{
			name: "with path",
			in: config.Secret{
				Name: "secret1",
				Path: "config/secret1.json",
			},
			want: "config/secret1.json",
		},
		{
			name: "without path uses name",
			in: config.Secret{
				Name: "secret2",
			},
			want: "secret2",
		},
		{
			name: "empty path uses name",
			in: config.Secret{
				Name: "secret3",
				Path: "",
			},
			want: "secret3",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.in.FilePath(); got != tt.want {
				t.Errorf("FilePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseWithInvalidPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		in     string
		errMsg string
	}{
		{
			name: "absolute path",
			in: `{
				"vaultID": "1234",
				"secrets": "- name: secret1\n  path: /absolute/path"
			}`,
			errMsg: "secrets[0].path is invalid: path must not be absolute",
		},
		{
			name: "relative escape",
			in: `{
				"vaultID": "1234",
				"secrets": "- name: secret1\n  path: ../escape"
			}`,
			errMsg: "secrets[0].path is invalid: path must not contain relative path escape sequences like '../'",
		},
		{
			name: "relative escape in middle",
			in: `{
				"vaultID": "1234",
				"secrets": "- name: secret1\n  path: config/../escape"
			}`,
			errMsg: "secrets[0].path is invalid: path must not contain relative path escape sequences like '../'",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := config.ParseMountRequest(tt.in)
			if err == nil {
				t.Fatal("expected error but got none")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error to contain %q, got %v", tt.errMsg, err)
			}
		})
	}
}
