package webrtc

import (
	"backnet/components"
	"backnet/controllers"
	"fmt"
	"log"
	"net/http"

	"time"

	"github.com/bitly/go-simplejson"
)

const (
	StorageVideoStream = 1
	CameraVideoStream  = 2
)

type ControllerMain struct {
	controllers.Controller
}

func NewControllerMain() ControllerMain {
	controller := ControllerMain{}

	return controller
}

func (сontroller ControllerMain) Index(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	request.View([]string{
		"views/layouts/main.html",
		"views/webrtc/index.html",
	}, 200, map[string]any{
		"Title": "Video",
	})
}

func (сontroller ControllerMain) Cam(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	request.View([]string{
		"views/layouts/main.html",
		"views/webrtc/cam.html",
	}, 200, map[string]any{
		"Title": "Camera",
	})
}

func (сontroller ControllerMain) CamStream(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	request.View([]string{
		"views/layouts/main.html",
		"views/webrtc/cam.stream.html",
	}, 200, map[string]any{
		"Title": "Camera",
	})
}

func (сontroller ControllerMain) WebrtcChannelsIndex(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	request.View([]string{
		"views/layouts/main.html",
		"views/webrtc/channels.index.html",
	}, 200, map[string]any{
		"Title": "Data Channels",
	})
}

func (сontroller ControllerMain) WebrtcSessionGet(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	r.ParseForm()

	if r.Form.Get("local_session") != "" {
		wrObj := NewWebrtObj()
		wrObj.Action = "storageVideoStream"

		wrObj.Data.Set("key", StorageVideoStream)
		wrObj.Data.Set("audio", "storage/video/output.ogg")
		wrObj.Data.Set("video", "storage/video/output.ivf")
		wrObj.Data.Set("local_session", r.Form.Get("local_session"))

		wrHub, err := WebrtHubByObj(wrObj)

		if err != nil {
			json := simplejson.New()
			json.Set("error", fmt.Sprint(err))

			payload, err := json.MarshalJSON()
			if err != nil {
				log.Println(err)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(payload)
		} else {
			wrHub.ChanStack <- wrObj

			select {
			case wrResp, ok := <-wrObj.ChanSource:
				if ok {
					switch wrResp.Action {
					case "SessionDescription":
						json := simplejson.New()

						var remote_session string

						components.СonvertAssign(&remote_session, wrResp.Data.Get("remote_session"))

						json.Set("remote_session", remote_session)

						payload, err := json.MarshalJSON()
						if err != nil {
							log.Println(err)
						}

						w.Header().Set("Content-Type", "application/json")
						w.Write(payload)
					}
				} else {
					fmt.Println("wrObj.ChanSource is close")
				}
			case <-time.After(10 * time.Second):
				wrObj.CloseChanSource()
			}
		}
	} else {
		json := simplejson.New()
		json.Set("error", "local_session not")

		payload, err := json.MarshalJSON()
		if err != nil {
			log.Println(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	}
}

func (сontroller ControllerMain) WebrtcCameraSet(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	r.ParseForm()

	if r.Form.Get("local_session") != "" {
		wrObj := NewWebrtObj()
		wrObj.Action = "cameraVideoSave"

		wrObj.Data.Set("file_out", "tmp/webrtc/video.webm")
		wrObj.Data.Set("local_session", r.Form.Get("local_session"))
		wrObj.Data.Set("max_time", 60*10*time.Second)

		wrHub, err := WebrtHubByObj(wrObj)

		if err != nil {
			json := simplejson.New()
			json.Set("error", fmt.Sprint(err))

			payload, err := json.MarshalJSON()
			if err != nil {
				log.Println(err)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(payload)
		} else {
			wrHub.ChanStack <- wrObj

			select {
			case wrResp, ok := <-wrObj.ChanSource:
				if ok {
					switch wrResp.Action {
					case "SessionDescription":
						json := simplejson.New()

						var remote_session string

						components.СonvertAssign(&remote_session, wrResp.Data.Get("remote_session"))

						json.Set("remote_session", remote_session)

						payload, err := json.MarshalJSON()
						if err != nil {
							log.Println(err)
						}

						w.Header().Set("Content-Type", "application/json")
						w.Write(payload)
					}
				} else {
					fmt.Println("wrObj.ChanSource is close")
				}
			case <-time.After(10 * time.Second):
				wrObj.CloseChanSource()
			}
		}
	} else {
		json := simplejson.New()
		json.Set("error", "local_session not")

		payload, err := json.MarshalJSON()
		if err != nil {
			log.Println(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	}
}

func (сontroller ControllerMain) WebrtcCameraStreamSet(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	r.ParseForm()

	if r.Form.Get("local_session") != "" {
		wrObj := NewWebrtObj()
		wrObj.Action = "cameraVideoStream"

		wrObj.Data.Set("key", CameraVideoStream)
		wrObj.Data.Set("action.set", true)
		wrObj.Data.Set("local_session", r.Form.Get("local_session"))
		wrObj.Data.Set("max_time", 60*60*10*time.Second)

		wrHub, err := WebrtHubByObj(wrObj)

		if err != nil {
			json := simplejson.New()
			json.Set("error", fmt.Sprint(err))

			payload, err := json.MarshalJSON()
			if err != nil {
				log.Println(err)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(payload)
		} else {
			wrHub.ChanStack <- wrObj

			select {
			case wrResp, ok := <-wrObj.ChanSource:
				if ok {
					switch wrResp.Action {
					case "SessionDescription":
						json := simplejson.New()

						var remote_session string

						components.СonvertAssign(&remote_session, wrResp.Data.Get("remote_session"))

						json.Set("remote_session", remote_session)

						payload, err := json.MarshalJSON()
						if err != nil {
							log.Println(err)
						}

						w.Header().Set("Content-Type", "application/json")
						w.Write(payload)
					}
				} else {
					fmt.Println("wrObj.ChanSource is close")
				}
			case <-time.After(10 * time.Second):
				wrObj.CloseChanSource()
			}
		}
	} else {
		json := simplejson.New()
		json.Set("error", "local_session not")

		payload, err := json.MarshalJSON()
		if err != nil {
			log.Println(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	}
}

func (сontroller ControllerMain) WebrtcCameraStreamGet(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	r.ParseForm()

	if r.Form.Get("local_session") != "" {
		wrObj := NewWebrtObj()
		wrObj.Action = "cameraVideoStream"

		wrObj.Data.Set("key", CameraVideoStream)
		wrObj.Data.Set("action.get", true)
		wrObj.Data.Set("local_session", r.Form.Get("local_session"))
		wrObj.Data.Set("max_time", 60*60*10*time.Second)

		wrHub, err := WebrtHubByObj(wrObj)

		if err != nil {
			json := simplejson.New()
			json.Set("error", fmt.Sprint(err))

			payload, err := json.MarshalJSON()
			if err != nil {
				log.Println(err)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(payload)
		} else {
			wrHub.ChanStack <- wrObj

			select {
			case wrResp, ok := <-wrObj.ChanSource:
				if ok {
					switch wrResp.Action {
					case "SessionDescription":
						json := simplejson.New()

						var remote_session string

						components.СonvertAssign(&remote_session, wrResp.Data.Get("remote_session"))

						json.Set("remote_session", remote_session)

						payload, err := json.MarshalJSON()
						if err != nil {
							log.Println(err)
						}

						w.Header().Set("Content-Type", "application/json")
						w.Write(payload)
					case "Error":
						json := simplejson.New()

						json.Set("error", wrResp.Data.Get("error"))

						payload, err := json.MarshalJSON()
						if err != nil {
							log.Println(err)
						}

						w.Header().Set("Content-Type", "application/json")
						w.Write(payload)
					}
				} else {
					fmt.Println("wrObj.ChanSource is close")
				}
			case <-time.After(10 * time.Second):
				wrObj.CloseChanSource()
			}
		}
	} else {
		json := simplejson.New()
		json.Set("error", "local_session not")

		payload, err := json.MarshalJSON()
		if err != nil {
			log.Println(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	}
}

func (сontroller ControllerMain) WebrtcChannelsSessionGet(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	r.ParseForm()

	if r.Form.Get("local_session") != "" {
		wrObj := NewWebrtObj()
		wrObj.Action = "WebrtcChannelsSessionGet"

		wrObj.Data.Set("local_session", r.Form.Get("local_session"))

		wrHub, err := WebrtHubByObj(wrObj)

		if err != nil {
			json := simplejson.New()
			json.Set("error", fmt.Sprint(err))

			payload, err := json.MarshalJSON()
			if err != nil {
				log.Println(err)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(payload)
		} else {
			wrHub.ChanStack <- wrObj

			select {
			case wrResp, ok := <-wrObj.ChanSource:
				if ok {
					switch wrResp.Action {
					case "SessionDescription":
						json := simplejson.New()

						var remote_session string

						components.СonvertAssign(&remote_session, wrResp.Data.Get("remote_session"))

						json.Set("remote_session", remote_session)

						payload, err := json.MarshalJSON()
						if err != nil {
							log.Println(err)
						}

						w.Header().Set("Content-Type", "application/json")
						w.Write(payload)
					}
				} else {
					fmt.Println("wrObj.ChanSource is close")
				}
			case <-time.After(10 * time.Second):
				wrObj.CloseChanSource()
			}
		}
	} else {
		json := simplejson.New()
		json.Set("error", "local_session not")

		payload, err := json.MarshalJSON()
		if err != nil {
			log.Println(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	}
}
