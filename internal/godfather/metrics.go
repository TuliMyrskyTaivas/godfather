package godfather

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsServer struct {
	Registry *prometheus.Registry
	Server   *http.Server
}

// ----------------------------------------------------------------
func StartMetricsServer(ctx context.Context, url string, port int) (*MetricsServer, error) {
	slog.Info(fmt.Sprintf("Starting Prometheus metrics server at http://localhost:%d%s...", port, url))

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	http.Handle(url, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		ReadHeaderTimeout: 5 * time.Second, // Prevent Slowloris attacks
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Prometheus metrics server failed", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("Prometheus metrics server stopped")

	// Gracefully shutdown the server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)

	return &MetricsServer{
		Registry: reg,
		Server:   server,
	}, nil
}

// ----------------------------------------------------------------
func (ms *MetricsServer) Stop() error {
	if ms.Server != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		slog.Info("Prometheus metrics server stopped")
		return ms.Server.Shutdown(shutdownCtx)
	}
	return nil
}

// ----------------------------------------------------------------
func (ms *MetricsServer) RegisterCounter(counter prometheus.Counter) {
	if ms.Registry != nil {
		ms.Registry.MustRegister(counter)
	}
}
