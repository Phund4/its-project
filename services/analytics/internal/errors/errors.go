// Package apperrors — ошибки уровня приложения analytics.
package apperrors

import "errors"

// ErrHTTPListen — не удалось слушать HTTP (слушающий сервер завершился с ошибкой).
var ErrHTTPListen = errors.New("http server listen error")
