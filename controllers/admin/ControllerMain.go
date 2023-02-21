package admin

import (
	"net/http"
	"backnet/components"
	"backnet/controllers"

	"backnet/components/monitor"
)

type ControllerMain struct {
	controllers.Controller
}

func NewControllerMain() ControllerMain {
	controller := ControllerMain{}

	return controller
}

func (сontroller ControllerMain) Index(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r).Admin()
	defer request.Store()

	if !request.Valid {
		return
	}

	request.View([]string{
		"views/admin/layouts/main.html",
		"views/admin/main/index.html",
	}, 200, map[string]any{
		"Title":          "Admin Panel",
		"UrlAdminLogout": components.Route("admin.auth.logout"),
	})
}

func (сontroller ControllerMain) Monitor() func(http.ResponseWriter, *http.Request) {
	handler := monitor.New()

	return func(w http.ResponseWriter, r *http.Request) {
		request := controllers.NewRequest(w, r).Admin()
		defer request.Store()

		if !request.Valid {
			return
		}

		handler(request)
		return
	}
}
