// Package apperrors — ошибки приложения ml_gateway.
package apperrors

import "errors"

// ErrHTTPListen — ошибка прослушивания HTTP-сервера.
var ErrHTTPListen = errors.New("http server listen error")
