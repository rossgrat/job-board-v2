package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

const (
	authCookieName = "session"
	authCookieMaxAge = 30 * 24 * 60 * 60 // 30 days
)

func authMiddleware(password string) func(http.Handler) http.Handler {
	token := generateToken(password)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(authCookieName)
			if err != nil || cookie.Value != token {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func generateToken(password string) string {
	mac := hmac.New(sha256.New, []byte(password))
	mac.Write([]byte("job-board-auth"))
	return hex.EncodeToString(mac.Sum(nil))
}

func slogRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		slog.Info("http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.Status()),
			slog.Int("bytes", ww.BytesWritten()),
			slog.Duration("duration", time.Since(start)),
		)
	})
}
