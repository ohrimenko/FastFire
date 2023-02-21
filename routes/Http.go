package routes

import (
	"net/http"

	"backnet/controllers"
	"backnet/controllers/admin"
	"backnet/controllers/frontend"

	"github.com/gorilla/mux"
)

func (route Route) Http(router *mux.Router) {
	adminControllerAuth := admin.NewControllerAuth()
	adminControllerMain := admin.NewControllerMain()

	frontendControllerMain := frontend.NewControllerMain()
	frontendControllerVideo := frontend.NewControllerVideo()

	router.Name("static").PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./public"))))

	router.Name("admin.index").Methods("GET").Path("/admin").HandlerFunc(adminControllerMain.Index)
	router.Name("admin.auth.login").Methods("GET").Path("/admin/login").HandlerFunc(adminControllerAuth.Login)
	router.Name("admin.auth.authorize").Methods("POST").Path("/admin/authorize").HandlerFunc(adminControllerAuth.Authorize)
	router.Name("admin.auth.logout").Methods("GET").Path("/admin/logout").HandlerFunc(adminControllerAuth.Logout)
	router.Name("admin.auth.monitor").Path("/admin/monitor").HandlerFunc(adminControllerMain.Monitor())

	router.Name("main.index").Methods("GET").Path("/").HandlerFunc(frontendControllerMain.Index)

	router.Name("video.index").Methods("GET").Path("/video").HandlerFunc(frontendControllerVideo.Index)
	router.Name("video.webrtc.session.get").Methods("POST").Path("/video/webrtc/session/get").HandlerFunc(frontendControllerVideo.WebrtcSessionGet)

	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, router *http.Request) {
		controllers.Abort404(w, router)
	})

	router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, router *http.Request) {
		controllers.Abort404(w, router)
	})
}
