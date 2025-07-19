package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	sacloudsm "github.com/sacloud/secretmanager-api-go"
	"github.com/tosuke/secrets-store-csi-driver-provider-sakuracloud/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	providerv1alpha1 "sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

func main() {
	var cfg config
	flag.StringVar(&cfg.endpoint, "endpoint", "unix:///tmp/sakuracloud.sock", "gRPC endpoint to connect to the provider")
	flag.TextVar(&cfg.healthzAddr, "healthz-addr", netip.MustParseAddrPort("0.0.0.0:8080"), "Healthz addr")

	var versionFlag bool
	flag.BoolVar(&versionFlag, "version", false, "Print version and exit")

	flag.Parse()
	if versionFlag {
		//nolint: forbidigo
		fmt.Println("Version:", Version)
		return
	}

	os.Exit(run(cfg))
}

type config struct {
	endpoint    string
	healthzAddr netip.AddrPort
}

const (
	gracefulShutdownTimeout = 5 * time.Second
)

// nolint: funlen
func run(cfg config) int {
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	client, err := sacloudsm.NewClient()
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create Secret Manager client", "error", err)
		return 1
	}

	grpcServ := grpc.NewServer(grpc.UnaryInterceptor(logInterceptor()))
	providerv1alpha1.RegisterCSIDriverProviderServer(grpcServ, server.NewServer(Version, client))

	slog.InfoContext(ctx, "Starting provider", "version", Version)

	network, addr, err := parseEndpoint(cfg.endpoint)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to parse endpoint", "endpoint", cfg.endpoint, "error", err)
		return 1
	}
	if network == "unix" {
		if err := os.Remove(addr); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.ErrorContext(ctx, "Failed to remove existing UDS file", "endpoint", cfg.endpoint, "error", err)
			return 1
		}
	}

	listener, err := net.Listen(network, addr)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to listen on UDS", "endpoint", cfg.endpoint, "error", err)
		return 1
	}
	defer func() {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			slog.ErrorContext(ctx, "Failed to close listener", "endpoint", cfg.endpoint, "error", err)
		}
	}()

	go func() {
		slog.InfoContext(ctx, "gRPC server is starting", "endpoint", cfg.endpoint)
		if err := grpcServ.Serve(listener); err != nil {
			cancel(fmt.Errorf("failed to start gRPC server: %w", err))
		}
	}()
	defer func() {
		slog.InfoContext(ctx, "Stopping gRPC server", "endpoint", cfg.endpoint)
		gracefully := stopGRPCServer(ctx, grpcServ)
		if gracefully {
			slog.InfoContext(ctx, "gRPC server stopped gracefully", "endpoint", cfg.endpoint)
		} else {
			slog.WarnContext(ctx, "gRPC server did not stop gracefully", "endpoint", cfg.endpoint)
		}
	}()

	// Start healthz server
	healthzMux := http.NewServeMux()
	healthzMux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	healthzServ := &http.Server{
		Addr:    cfg.healthzAddr.String(),
		Handler: healthzMux,
	}
	go func() {
		if err := healthzServ.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.ErrorContext(ctx, "Failed to start healthz server", "addr", cfg.healthzAddr, "error", err)
		}
	}()
	defer func() {
		if err := healthzServ.Shutdown(ctx); err != nil {
			slog.ErrorContext(ctx, "Failed to shutdown healthz server gracefully", "error", err)
		}
	}()

	<-ctx.Done()
	if err := context.Cause(ctx); !errors.Is(err, ctx.Err()) {
		slog.ErrorContext(ctx, "Failed to run provider", "error", err)
		return 1
	}

	slog.InfoContext(ctx, "Shutting down gracefully", "timeout", gracefulShutdownTimeout)
	shutdownCtx, cancelShutdownTimeout := context.WithTimeout(context.WithoutCancel(ctx), gracefulShutdownTimeout)
	_ = cancelShutdownTimeout
	ctx = shutdownCtx

	return 0
}

func parseEndpoint(endpoint string) (network, addr string, err error) {
	lowerEp := strings.ToLower(endpoint)
	if strings.HasPrefix(lowerEp, "unix://") || strings.HasPrefix(lowerEp, "tcp://") {
		network, addr, ok := strings.Cut(endpoint, "://")
		if ok {
			return network, addr, nil
		}
	}

	return "", "", fmt.Errorf("invalid endpoint format: %s", endpoint)
}

func stopGRPCServer(ctx context.Context, serv *grpc.Server) (gracefully bool) {
	stop := make(chan struct{})

	go func() {
		serv.GracefulStop()
		close(stop)
	}()

	select {
	case <-stop:
		return true
	case <-ctx.Done():
		serv.Stop()
	}
	return false
}

func logInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		slog.InfoContext(ctx, "gRPC request received", "method", info.FullMethod)
		resp, err := handler(ctx, req)
		status, _ := status.FromError(err)
		slog.InfoContext(ctx, "gRPC request processed", "method", info.FullMethod, "duration", time.Since(start).String(), "status.code", status.Code().String(), "status.message", status.Message())
		return resp, err
	}
}
