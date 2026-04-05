package app

import "errors"

// ErrHTTPListen — сервер завершил Listen с ошибкой.
var ErrHTTPListen = errors.New("http server listen error")
