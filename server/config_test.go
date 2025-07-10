package server_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tosuke/secrets-store-csi-driver-provider-sakuracloud/server"
)

func TestParse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want *server.MountConfig
	}{
		{
			name: "basic",
			in: `{
				"csi.storage.k8s.io/pod.name": "test-pod",
				"csi.storage.k8s.io/pod.namespace": "default",
				"vaultID": "1234",
				"secrets": "- name: secret1\n- vaultID: \"5678\"\n  name: secret2"
			}`,
			want: &server.MountConfig{
				PodName:      "test-pod",
				PodNamespace: "default",

				VaultID: "1234",
				Secrets: server.Secrets{
					{Name: "secret1"},
					{VaultID: "5678", Name: "secret2"},
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := server.ParseConfig([]byte(tt.in))
			if err != nil {
				t.Fatalf("parse config: %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unexpected diff (-want +got):\n%s", diff)
			}
		})
	}
}
