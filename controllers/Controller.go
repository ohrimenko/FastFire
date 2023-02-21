package controllers

import (
	"fmt"
	"net/http"
	"backnet/components"
	"backnet/config"
)

type Controller struct {
}

func Abort404(w http.ResponseWriter, r *http.Request) {
	components.View(w, []string{
		"views/layouts/main.html",
		"views/errors/404.html",
	}, 404, map[string]any{
		"Title":     "404 Not Found",
		"Error":     "Error 404",
		"TextError": "404 Not Found",
	})
}

func Abort500(w http.ResponseWriter, r *http.Request) {
	components.View(w, []string{
		"views/layouts/main.html",
		"views/errors/500.html",
	}, 500, map[string]any{
		"Title":     "500 Internal Server Error",
		"Error":     "Error 500",
		"TextError": "500 Internal Server Error",
	})
}

func RedirectToHTTPSRouter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		port := ""

		if config.Env("HTTPS_PORT") != "443" {
			port = ":" + config.Env("HTTPS_PORT")
		}

		http.Redirect(w, r, fmt.Sprintf("https://%s%s%s", config.Env("HOST"), port, r.URL), http.StatusPermanentRedirect)
		return

		next.ServeHTTP(w, r)
	})
}
