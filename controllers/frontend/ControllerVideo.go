package frontend

import (
	"log"
	"net/http"
	"backnet/controllers"
	"backnet/webrtc"

	"time"

	"github.com/bitly/go-simplejson"
)

const (
	audioFileName   = "storage/video/audio-1.ogg"
	videoFileName   = "storage/video/video-1.ivf"
	oggPageDuration = time.Millisecond * 20
)

type ControllerVideo struct {
	controllers.Controller
}

func NewControllerVideo() ControllerVideo {
	controller := ControllerVideo{}

	return controller
}

func (сontroller ControllerVideo) Index(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	request.View([]string{
		"views/layouts/main.html",
		"views/video/index.html",
	}, 200, map[string]any{
		"Title": "Video",
	})
}

func (сontroller ControllerVideo) WebrtcSessionGet(w http.ResponseWriter, r *http.Request) {
	request := controllers.NewRequest(w, r)
	defer request.Store()

	if !request.Valid {
		return
	}

	r.ParseForm()

	if r.Form.Get("local_session") != "" {
		wrHub := webrtc.WebrtHub()

		wrObj := &webrtc.WebrtObj{
			Action:     "getSessionDescription",
			Message:    r.Form.Get("local_session"),
			ChanSource: make(chan *webrtc.WebrtResp),
		}

		wrHub.ChanStack <- wrObj

		select {
		case wrResp := <-wrObj.ChanSource:
			switch wrResp.Action {
			case "SessionDescription":
				json := simplejson.New()
				json.Set("remote_session", wrResp.Message)

				payload, err := json.MarshalJSON()
				if err != nil {
					log.Println(err)
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write(payload)
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
