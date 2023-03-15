package routes

import (
	"backnet/controllers/webrtc"

	"github.com/gorilla/mux"
)

func (route Route) Webrtc(router *mux.Router) {
	controllerWebrtc := webrtc.NewControllerMain()

	router.Name("webrtc.video.index").Methods("GET").Path("/video").HandlerFunc(controllerWebrtc.Index)
	router.Name("webrtc.video.webrtc.session.get").Methods("POST").Path("/video/webrtc/session/get").HandlerFunc(controllerWebrtc.WebrtcSessionGet)

	router.Name("webrtc.video.cam").Methods("GET").Path("/cam").HandlerFunc(controllerWebrtc.Cam)
	router.Name("webrtc.video.webrtc.camera.set").Methods("POST").Path("/video/webrtc/camera/set").HandlerFunc(controllerWebrtc.WebrtcCameraSet)
	router.Name("webrtc.video.cam.stream").Methods("GET").Path("/cam/stream").HandlerFunc(controllerWebrtc.CamStream)
	router.Name("webrtc.video.webrtc.camera.stream.set").Methods("POST").Path("/video/webrtc/camera/stream/set").HandlerFunc(controllerWebrtc.WebrtcCameraStreamSet)
	router.Name("webrtc.video.webrtc.camera.stream.get").Methods("POST").Path("/video/webrtc/camera/stream/get").HandlerFunc(controllerWebrtc.WebrtcCameraStreamGet)

	router.Name("webrtc.channels.index").Methods("GET").Path("/channels/index").HandlerFunc(controllerWebrtc.WebrtcChannelsIndex)
	router.Name("webrtc.channels.session.get").Methods("POST").Path("/webrtc/channels/session/get").HandlerFunc(controllerWebrtc.WebrtcChannelsSessionGet)
}
