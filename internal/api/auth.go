package api

import (
	"net/http"

	"github.com/rossgrat/job-board-v2/internal/api/templates"
)

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	templates.LoginPage("").Render(r.Context(), w)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")

	if password != s.cfg.Auth.Password {
		templates.LoginPage("Invalid password").Render(r.Context(), w)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    generateToken(password),
		Path:     "/",
		MaxAge:   authCookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}
