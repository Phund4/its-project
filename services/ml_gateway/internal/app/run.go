package app

import "context"

// Run инициализирует зависимости и запускает HTTP-сервер до завершения rootCtx.
func Run(rootCtx context.Context) error {
	deps := InitializeDependencies()
	return RunHTTPServer(rootCtx, deps)
}
