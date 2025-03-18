package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const interval = 10 * time.Second

func main() {
	ctx := context.Background()

	if os.Getenv("DEBUG") == "true" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		slog.Error("failed to get docker client", "error", err)
		os.Exit(1)
	}

	volumeSizeMetric := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "volume_size_bytes",
		Help: "Number of bytes the volume is using on the file system",
	}, []string{"name", "path", "scope", "created_at"})

	go collectMetrics(volumeSizeMetric, cli, ctx)

	server := http.Server{
		Addr:         ":2112",
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	http.Handle("/metrics", promhttp.Handler())

	err = server.ListenAndServe()
	if err != nil {
		slog.Error("an error occurred while serving metrics", "error", err)
		os.Exit(1)
	}
}

func collectMetrics(volumeSizeMetric *prometheus.GaugeVec, cli *client.Client, ctx context.Context) {
	for {
		time.Sleep(interval)

		diskusage, err := cli.DiskUsage(ctx, types.DiskUsageOptions{
			Types: []types.DiskUsageObject{types.VolumeObject},
		})
		if err != nil {
			slog.Error("error getting volumes", "error", err)
			return
		}

		slog.Debug("gathered volumes", "volumes", len(diskusage.Volumes))

		for _, volume := range diskusage.Volumes {
			slog.Debug("updating metric for volume",
				"name", volume.Name,
				"size", volume.UsageData.Size,
				"created_at", volume.CreatedAt,
				"scope", volume.Scope,
				"path", volume.Mountpoint,
			)

			volumeSizeMetric.With(prometheus.Labels{
				"name":       volume.Name,
				"path":       volume.Mountpoint,
				"scope":      volume.Scope,
				"created_at": volume.CreatedAt,
			}).Set(float64(volume.UsageData.Size))
		}
	}
}
