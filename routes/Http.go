package routes

import (
	"net/http"

	"backnet/controllers"
	"backnet/controllers/admin"
	"backnet/controllers/frontend"
	"backnet/controllers/sse"
	"backnet/controllers/ws"

	"github.com/gorilla/mux"
)

func (route Route) Http(router *mux.Router) {
	adminControllerAuth := admin.NewControllerAuth()
	adminControllerMain := admin.NewControllerMain()
	wsControllerMain := ws.NewControllerMain()
	sseControllerMain := sse.NewControllerMain()

	frontendControllerMain := frontend.NewControllerMain()

	router.Name("static").PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./public"))))
	router.Name("static.favicon.ico").Methods("GET").Path("/favicon.ico").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/favicon.ico")
	})

	router.Name("admin.index").Methods("GET").Path("/admin").HandlerFunc(adminControllerMain.Index)
	router.Name("admin.auth.login").Methods("GET").Path("/admin/login").HandlerFunc(adminControllerAuth.Login)
	router.Name("admin.auth.authorize").Methods("POST").Path("/admin/authorize").HandlerFunc(adminControllerAuth.Authorize)
	router.Name("admin.auth.logout").Methods("GET").Path("/admin/logout").HandlerFunc(adminControllerAuth.Logout)
	router.Name("admin.auth.monitor").Path("/admin/monitor").HandlerFunc(adminControllerMain.Monitor())

	router.Name("main.index").Methods("GET").Path("/").HandlerFunc(frontendControllerMain.Index)

	router.Name("video.mediadata").Methods("GET").Path("/video/mediadata").HandlerFunc(frontendControllerMain.MediaData)
	router.Name("video.mediadata.preload").Methods("POST").Path("/video/mediadata/preload").HandlerFunc(frontendControllerMain.MediaDataPreload)

	router.Name("websocket.index").Methods("GET").Path("/chat").HandlerFunc(wsControllerMain.Index)

	router.Name("sse.index").Methods("GET").Path("/sse/index").HandlerFunc(sseControllerMain.Index)
	router.Name("sse.message").Methods("POST").Path("/sse/message").HandlerFunc(controllers.SseOnMessage)

	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, router *http.Request) {
		controllers.Abort404(w, router)
	})

	router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, router *http.Request) {
		controllers.Abort404(w, router)
	})
}
