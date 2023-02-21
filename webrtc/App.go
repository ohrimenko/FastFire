package webrtc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"backnet/components"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"

	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/pion/webrtc/v3/pkg/media/oggreader"
)

const (
	audioFileName   = "storage/video/audio-1.ogg"
	videoFileName   = "storage/video/video-1.ivf"
	oggPageDuration = time.Millisecond * 20
)

type WebrtResp struct {
	Action  string
	Message string
}

type WebrtObj struct {
	ChanSource chan *WebrtResp
	Action     string
	Message    string
}

type webrtHub struct {
	Mutex     sync.Mutex
	ChanStack chan *WebrtObj
	Count     uint64
	Key       uint64
}

type webrtcStruct struct {
	I     uint64
	Mutex sync.Mutex
	Stack map[uint64]*webrtHub

	valid bool
}

var webrtcApp webrtcStruct

func (wr *webrtcStruct) webrtc(count int) *webrtcStruct {
	if !wr.valid {
		wr.I = 0
		wr.Stack = map[uint64]*webrtHub{}

		wr.valid = true

		for i := 0; i < count; i++ {
			wr.newHub()
		}
	}

	return wr
}

func (wr *webrtcStruct) newHub() {
	wr.I++

	wr.Stack[wr.I] = &webrtHub{
		Count:     0,
		Key:       wr.I,
		ChanStack: make(chan *WebrtObj),
	}

	go wr.Stack[wr.I].RunHub()
}

func (wrHub *webrtHub) RunHub() {
	for {
		select {
		case wObj := <-wrHub.ChanStack:
			go func() {
				chanComplete := make(chan error)

				switch wObj.Action {
				case "getSessionDescription":
					wrHub.Mutex.Lock()
					wrHub.Count++
					wrHub.Mutex.Unlock()

					defer func() {
						wrHub.Mutex.Lock()
						wrHub.Count--
						wrHub.Mutex.Unlock()
					}()

					// Assert that we have an audio or video file
					_, err := os.Stat(videoFileName)
					haveVideoFile := !os.IsNotExist(err)

					_, err = os.Stat(audioFileName)
					haveAudioFile := !os.IsNotExist(err)

					if !haveAudioFile && !haveVideoFile {
						return
						panic("Could not find `" + audioFileName + "` or `" + videoFileName + "`")
					}

					// Create a new RTCPeerConnection
					peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
						ICEServers: []webrtc.ICEServer{
							{
								URLs: []string{"stun:stun.l.google.com:19302"},
							},
						},
					})
					if err != nil {
						return
						panic(err)
					}
					defer func() {
						if cErr := peerConnection.Close(); cErr != nil {
							fmt.Printf("cannot close peerConnection: %v\n", cErr)
						}
					}()

					iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

					if haveVideoFile {
						// Create a video track
						videoTrack, videoTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion")
						if videoTrackErr != nil {
							return
							panic(videoTrackErr)
						}

						rtpSender, videoTrackErr := peerConnection.AddTrack(videoTrack)
						if videoTrackErr != nil {
							return
							panic(videoTrackErr)
						}

						// Read incoming RTCP packets
						// Before these packets are returned they are processed by interceptors. For things
						// like NACK this needs to be called.
						go func() {
							rtcpBuf := make([]byte, 1500)
							for {
								if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
									return
								}
							}
						}()

						go func() {
							// Open a IVF file and start reading using our IVFReader
							file, ivfErr := os.Open(videoFileName)
							if ivfErr != nil {
								chanComplete <- ivfErr
								return
								panic(ivfErr)
							}

							ivf, header, ivfErr := ivfreader.NewWith(file)
							if ivfErr != nil {
								chanComplete <- ivfErr
								return
								panic(ivfErr)
							}

							// Wait for connection established
							<-iceConnectedCtx.Done()

							// Send our video file frame at a time. Pace our sending so we send it at the same speed it should be played back as.
							// This isn't required since the video is timestamped, but we will such much higher loss if we send all at once.
							//
							// It is important to use a time.Ticker instead of time.Sleep because
							// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
							// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)
							ticker := time.NewTicker(time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*1000))
							for ; true; <-ticker.C {
								frame, _, ivfErr := ivf.ParseNextFrame()
								if errors.Is(ivfErr, io.EOF) {
									fmt.Printf("All video frames parsed and sent")

									chanComplete <- nil
									return
									os.Exit(0)
								}

								if ivfErr != nil {
									chanComplete <- ivfErr
									return
									panic(ivfErr)
								}

								if ivfErr = videoTrack.WriteSample(media.Sample{Data: frame, Duration: time.Second}); ivfErr != nil {
									chanComplete <- ivfErr
									return
									panic(ivfErr)
								}
							}
						}()
					}

					if haveAudioFile {
						// Create a audio track
						audioTrack, audioTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
						if audioTrackErr != nil {
							return
							panic(audioTrackErr)
						}

						rtpSender, audioTrackErr := peerConnection.AddTrack(audioTrack)
						if audioTrackErr != nil {
							return
							panic(audioTrackErr)
						}

						// Read incoming RTCP packets
						// Before these packets are returned they are processed by interceptors. For things
						// like NACK this needs to be called.
						go func() {
							rtcpBuf := make([]byte, 1500)
							for {
								if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
									return
								}
							}
						}()

						go func() {
							// Open a OGG file and start reading using our OGGReader
							file, oggErr := os.Open(audioFileName)
							if oggErr != nil {
								chanComplete <- oggErr
								return
								panic(oggErr)
							}

							// Open on oggfile in non-checksum mode.
							ogg, _, oggErr := oggreader.NewWith(file)
							if oggErr != nil {
								chanComplete <- oggErr
								return
								panic(oggErr)
							}

							// Wait for connection established
							<-iceConnectedCtx.Done()

							// Keep track of last granule, the difference is the amount of samples in the buffer
							var lastGranule uint64

							// It is important to use a time.Ticker instead of time.Sleep because
							// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
							// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)
							ticker := time.NewTicker(oggPageDuration)
							for ; true; <-ticker.C {
								pageData, pageHeader, oggErr := ogg.ParseNextPage()
								if errors.Is(oggErr, io.EOF) {
									fmt.Printf("All audio pages parsed and sent")

									chanComplete <- nil
									return
									os.Exit(0)
								}

								if oggErr != nil {
									chanComplete <- oggErr
									return
									panic(oggErr)
								}

								// The amount of samples is the difference between the last and current timestamp
								sampleCount := float64(pageHeader.GranulePosition - lastGranule)
								lastGranule = pageHeader.GranulePosition
								sampleDuration := time.Duration((sampleCount/48000)*1000) * time.Millisecond

								if oggErr = audioTrack.WriteSample(media.Sample{Data: pageData, Duration: sampleDuration}); oggErr != nil {
									chanComplete <- oggErr
									return
									panic(oggErr)
								}
							}
						}()
					}

					// Set the handler for ICE connection state
					// This will notify you when the peer has connected/disconnected
					peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
						fmt.Printf("Connection State has changed %s \n", connectionState.String())
						if connectionState == webrtc.ICEConnectionStateConnected {
							iceConnectedCtxCancel()
						}
					})

					// Set the handler for Peer connection state
					// This will notify you when the peer has connected/disconnected
					peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
						fmt.Printf("Peer Connection State has changed: %s\n", s.String())

						if s == webrtc.PeerConnectionStateFailed {
							// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
							// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
							// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
							fmt.Println("Peer Connection has gone to failed exiting")
							return
							os.Exit(0)
						}
					})

					// Wait for the offer to be pasted
					offer := webrtc.SessionDescription{}
					components.Decode(wObj.Message, &offer)

					// Set the remote SessionDescription
					if err = peerConnection.SetRemoteDescription(offer); err != nil {
						return
						panic(err)
					}

					// Create answer
					answer, err := peerConnection.CreateAnswer(nil)
					if err != nil {
						return
						panic(err)
					}

					// Create channel that is blocked until ICE Gathering is complete
					gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

					// Sets the LocalDescription, and starts our UDP listeners
					if err = peerConnection.SetLocalDescription(answer); err != nil {
						return
						panic(err)
					}

					// Block until ICE Gathering is complete, disabling trickle ICE
					// we do this because we only can exchange one signaling message
					// in a production application you should exchange ICE Candidates via OnICECandidate
					<-gatherComplete

					wResp := &WebrtResp{
						Action:  "SessionDescription",
						Message: components.Encode(*peerConnection.LocalDescription()),
					}

					wObj.ChanSource <- wResp

					select {
					case <-chanComplete:
					}
				}
			}()
		}
	}
}

func Webrtc() *webrtcStruct {
	return webrtcApp.webrtc(50)
}

func WebrtHub() *webrtHub {
	wr := Webrtc()

	if _, ok := wr.Stack[1]; ok {
		hub := wr.Stack[1]

		for i, _ := range wr.Stack {
			if wr.Stack[i].Count < hub.Count {
				hub = wr.Stack[i]
			}
		}

		return hub
	}

	return nil
}
