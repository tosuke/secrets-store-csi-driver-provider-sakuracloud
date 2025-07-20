package server

import (
	"context"
	"fmt"
	"io/fs"
	"strconv"

	sacloudsm "github.com/sacloud/secretmanager-api-go"
	sacloudsmv1 "github.com/sacloud/secretmanager-api-go/apis/v1"
	"github.com/tosuke/secrets-store-csi-driver-provider-sakuracloud/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	providerv1alpha1 "sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

const providerName = "secrets-store-csi-driver-provider-sakuracloud"

type Server struct {
	version string
	client  *sacloudsmv1.Client
}

func NewServer(version string, client *sacloudsmv1.Client) *Server {
	return &Server{
		version: version,
		client:  client,
	}
}

var _ providerv1alpha1.CSIDriverProviderServer = (*Server)(nil)

func (s *Server) Version(ctx context.Context, req *providerv1alpha1.VersionRequest) (*providerv1alpha1.VersionResponse, error) {
	return &providerv1alpha1.VersionResponse{
		Version:        "v1alpha1",
		RuntimeName:    providerName,
		RuntimeVersion: s.version,
	}, nil
}

//nolint:funlen
func (s *Server) Mount(ctx context.Context, req *providerv1alpha1.MountRequest) (*providerv1alpha1.MountResponse, error) {
	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "targetPath is required")
	}

	permu64, err := strconv.ParseUint(req.GetPermission(), 10, 32)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unable to parse permission: %v", req.GetPermission())
	}
	permission := fs.FileMode(permu64)

	cfg, err := config.ParseMountRequest(req.GetAttributes())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse mount config: %v", err)
	}

	ovs := make([]*providerv1alpha1.ObjectVersion, 0, len(cfg.Secrets))
	files := make([]*providerv1alpha1.File, 0, len(cfg.Secrets))
	for _, secret := range cfg.Secrets {
		var vaultID string
		if secret.VaultID != "" {
			vaultID = secret.VaultID
		} else {
			vaultID = cfg.VaultID
		}

		if vaultID == "" {
			return nil, status.Error(codes.InvalidArgument, "vaultID is required")
		}

		secretOp := sacloudsm.NewSecretOp(s.client, vaultID)
		unveilRequest := sacloudsmv1.Unveil{
			Name: secret.Name,
		}

		// Set version if specified
		if secret.Version != nil {
			unveilRequest.SetVersion(sacloudsmv1.NewOptNilInt(*secret.Version))
		}

		unveilResult, err := secretOp.Unveil(ctx, unveilRequest)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unveil secret %q in vault %q: %v", secret.Name, vaultID, err)
		}

		var version string
		if ver, ok := unveilResult.GetVersion().Get(); ok {
			version = strconv.FormatInt(int64(ver), 10)
		}
		ovs = append(ovs, &providerv1alpha1.ObjectVersion{
			Id:      fmt.Sprintf("vaults/%s/secrets/%s", vaultID, secret.Name),
			Version: version,
		})
		files = append(files, &providerv1alpha1.File{
			Path:     secret.Name,
			Mode:     int32(permission),
			Contents: []byte(unveilResult.GetValue()),
		})
	}

	return &providerv1alpha1.MountResponse{
		ObjectVersion: ovs,
		Error:         nil,
		Files:         files,
	}, nil
}
