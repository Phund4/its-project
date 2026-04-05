package utils

import "context"

// WithAppShutdown возвращает контекст, отменяемый при завершении req или app.
func WithAppShutdown(req, app context.Context) (context.Context, context.CancelFunc) {
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
