package api

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
)

func RecoverMiddleware() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					var recoveredError error
					switch v := recovered.(type) {
					case error:
						recoveredError = v
					case string:
						recoveredError = errors.New(v)
					default:
						recoveredError = fmt.Errorf("panic")
					}
					if rr, ok := w.(*responseRecorder); ok {
						rr.panicError = recoveredError
						rr.panicStack = debug.Stack()
						http.Error(rr, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					} else {
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
