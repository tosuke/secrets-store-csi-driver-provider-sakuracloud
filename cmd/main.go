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

func run(cfg config) int {
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.InfoContext(ctx, "Starting provider", "version", Version, "endpoint", cfg.endpoint)
	shutdownProvider, err := setupProvider(ctx, cfg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to setup provider", "error", err)
		return 1
	}
	defer func() {
		slog.InfoContext(ctx, "Stopping provider", "endpoint", cfg.endpoint)
		if err := shutdownProvider(ctx); err != nil && errors.Is(err, errForcedShutdown) {
			slog.WarnContext(ctx, "Provider shutdown was forced")
		} else if err != nil {
			slog.ErrorContext(ctx, "Failed to shutdown provider", "error", err)
		}
	}()

	shutdownHealthz := setupHealthzServer(ctx, cfg)
	defer func() {
		slog.InfoContext(ctx, "Stopping healthz server", "addr", cfg.healthzAddr)
		if err := shutdownHealthz(ctx); err != nil {
			slog.ErrorContext(ctx, "Failed to shutdown healthz server gracefully", "error", err)
		}
	}()

	<-ctx.Done()
	slog.InfoContext(ctx, "Shutting down gracefully", "timeout", gracefulShutdownTimeout)
	shutdownCtx, cancelShutdownTimeout := context.WithTimeout(context.WithoutCancel(ctx), gracefulShutdownTimeout)
	_ = cancelShutdownTimeout
	ctx = shutdownCtx

	return 0
}

type shutdownFunc func(context.Context) error

func setupProvider(ctx context.Context, cfg config) (shutdownFunc, error) {
	client, err := sacloudsm.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Secret Manager client: %w", err)
	}

	grpcServ := grpc.NewServer(grpc.UnaryInterceptor(logInterceptor()))
	providerv1alpha1.RegisterCSIDriverProviderServer(grpcServ, server.NewServer(Version, client))

	network, addr, err := parseEndpoint(cfg.endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint: %w", err)
	}
	if network == "unix" {
		if err := os.Remove(addr); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to remove existing UDS file: %w", err)
		}
	}

	listener, err := (&net.ListenConfig{}).Listen(ctx, network, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		if err := grpcServ.Serve(listener); err != nil {
			slog.ErrorContext(ctx, "Failed to start gRPC server", "error", err)
		}
	}()

	return func(ctx context.Context) error {
		stopErr := stopGRPCServer(ctx, grpcServ)

		var closeErr error
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			closeErr = err
		}
		return errors.Join(stopErr, closeErr)
	}, nil
}

func setupHealthzServer(ctx context.Context, cfg config) shutdownFunc {
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

	return func(ctx context.Context) error {
		if err := healthzServ.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown healthz server gracefully: %w", err)
		}
		return nil
	}
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

var errForcedShutdown = errors.New("forced shutdown")

func stopGRPCServer(ctx context.Context, serv *grpc.Server) error {
	stop := make(chan struct{})

	go func() {
		serv.GracefulStop()
		close(stop)
	}()

	select {
	case <-stop:
		return nil
	case <-ctx.Done():
		serv.Stop()
	}
	return errForcedShutdown
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
