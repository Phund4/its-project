package services

import "context"

// withAppShutdown возвращает контекст, отменяемый при завершении HTTP-запроса или приложения.
func withAppShutdown(req, app context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(req)
	if app == nil {
		return ctx, cancel
	}
	go func() {
		select {
		case <-app.Done():
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}
