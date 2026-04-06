package app

import (
	"context"
	"log/slog"
	"sync"

	"data-ingestion/internal/config"
	"data-ingestion/internal/core/services"
)

// StartCameraWorkers запускает воркеры захвата по всем камерам из конфигурации (неблокирующий вызов).
func StartCameraWorkers(ctx context.Context, deps *Deps, wg *sync.WaitGroup) {
	StartCameraWorkersWithCameras(ctx, deps, deps.Config.Cameras, wg)
}

func StartCameraWorkersWithCameras(ctx context.Context, deps *Deps, cameras []config.Camera, wg *sync.WaitGroup) {
	for _, cam := range cameras {
		cam := cam
		wg.Add(1)
		go func() {
			defer wg.Done()
			services.RunCamera(ctx, cam, deps.Store, deps.ML, deps.Config.S3.Prefix, deps.Config.Ingest.FFmpegPath, deps.Config.Ingest.TargetFPS, deps.Config.Ingest.ProcessWorkers)
		}()
	}
}

// WaitWorkers ждёт завершения воркеров после отмены контекста.
func WaitWorkers(wg *sync.WaitGroup) {
	slog.Info("waiting for background workers")
	wg.Wait()
}
