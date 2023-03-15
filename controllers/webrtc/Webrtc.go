package webrtc

import (
	"backnet/components"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"

	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/pion/webrtc/v3/pkg/media/oggreader"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"

	"backnet/config"

	"github.com/at-wat/ebml-go/webm"
)

type webrtResp struct {
	Mutex  sync.Mutex
	Action string
	Data   *components.Data
}

type webrtObj struct {
	Mutex          sync.Mutex
	OpenChanSource bool
	ChanSource     chan *webrtResp
	Action         string
	Data           *components.Data
}

type webrtConnection struct {
	Mutex       sync.Mutex
	KeyI        uint64
	WItem       *webrtItem
	Connection  *webrtc.PeerConnection
	DataChannel *webrtc.DataChannel
}

type webrtItem struct {
	Mutex        sync.Mutex
	Key          uint64
	WHub         *webrtHub
	WObj         *webrtObj
	OfferChan    chan *webrtObj
	CompleteChan chan error
	Connections  map[uint64]*webrtConnection
}

type webrtHub struct {
	Mutex     sync.Mutex
	ChanStack chan *webrtObj
	Count     uint64
	Key       uint64
	Conn_i    uint64
	Stack     map[uint64]*webrtItem
}

type WebrtcApi struct {
	Mutex sync.Mutex
	Stack map[uint64]*webrtHub
	I     uint64
	valid bool
}

var webrtcApp WebrtcApi

type webmSaver struct {
	audioWriter, videoWriter       webm.BlockWriteCloser
	audioBuilder, videoBuilder     *samplebuilder.SampleBuilder
	audioTimestamp, videoTimestamp time.Duration
}

func newWebmSaver() *webmSaver {
	return &webmSaver{
		audioBuilder: samplebuilder.New(10, &codecs.OpusPacket{}, 48000),
		videoBuilder: samplebuilder.New(10, &codecs.VP8Packet{}, 90000),
	}
}

func (wItem *webrtItem) NewWebrtConnection(peerConnection *webrtc.PeerConnection) *webrtConnection {
	atomic.AddUint64(&wItem.WHub.Conn_i, 1)
	conn_i := wItem.WHub.Conn_i

	wItem.Connections[conn_i] = &webrtConnection{
		KeyI:       conn_i,
		Connection: peerConnection,
		WItem:      wItem,
	}

	return wItem.Connections[conn_i]
}

func (s *webmSaver) Close() {
	fmt.Printf("Finalizing webm...\n")
	if s.audioWriter != nil {
		if err := s.audioWriter.Close(); err != nil {
			fmt.Println(err)
			return
		}
	}
	if s.videoWriter != nil {
		if err := s.videoWriter.Close(); err != nil {
			fmt.Println(err)
			return
		}
	}
}
func (s *webmSaver) PushOpus(rtpPacket *rtp.Packet) {
	s.audioBuilder.Push(rtpPacket)

	for {
		sample := s.audioBuilder.Pop()
		if sample == nil {
			return
		}
		if s.audioWriter != nil {
			s.audioTimestamp += sample.Duration
			if _, err := s.audioWriter.Write(true, int64(s.audioTimestamp/time.Millisecond), sample.Data); err != nil {
				return
			}
		}
	}
}
func (s *webmSaver) PushVP8(file_out string, rtpPacket *rtp.Packet) {
	s.videoBuilder.Push(rtpPacket)

	for {
		sample := s.videoBuilder.Pop()
		if sample == nil {
			return
		}
		// Read VP8 header.
		videoKeyframe := (sample.Data[0]&0x1 == 0)
		if videoKeyframe {
			// Keyframe has frame information.
			raw := uint(sample.Data[6]) | uint(sample.Data[7])<<8 | uint(sample.Data[8])<<16 | uint(sample.Data[9])<<24
			width := int(raw & 0x3FFF)
			height := int((raw >> 16) & 0x3FFF)

			if s.videoWriter == nil || s.audioWriter == nil {
				// Initialize WebM saver using received frame size.
				s.InitWriter(file_out, width, height)
			}
		}
		if s.videoWriter != nil {
			s.videoTimestamp += sample.Duration
			if _, err := s.videoWriter.Write(videoKeyframe, int64(s.videoTimestamp/time.Millisecond), sample.Data); err != nil {
				return
			}
		}
	}
}
func (s *webmSaver) InitWriter(file_out string, width, height int) {
	w, err := os.OpenFile(file_out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return
	}

	ws, err := webm.NewSimpleBlockWriter(w,
		[]webm.TrackEntry{
			{
				Name:            "Audio",
				TrackNumber:     1,
				TrackUID:        12345,
				CodecID:         "A_OPUS",
				TrackType:       2,
				DefaultDuration: 20000000,
				Audio: &webm.Audio{
					SamplingFrequency: 48000.0,
					Channels:          2,
				},
			}, {
				Name:            "Video",
				TrackNumber:     2,
				TrackUID:        67890,
				CodecID:         "V_VP8",
				TrackType:       1,
				DefaultDuration: 33333333,
				Video: &webm.Video{
					PixelWidth:  uint64(width),
					PixelHeight: uint64(height),
				},
			},
		})
	if err != nil {
		return
	}
	fmt.Printf("WebM saver has started with video width=%d, height=%d\n", width, height)
	s.audioWriter = ws[0]
	s.videoWriter = ws[1]
}

func NewWebrtObj() *webrtObj {
	return &webrtObj{
		Action:         "",
		Data:           components.NewData(),
		OpenChanSource: true,
		ChanSource:     make(chan *webrtResp),
	}
}

func (WObj *webrtObj) CloseChanSource() {
	if WObj.OpenChanSource {
		WObj.Mutex.Lock()
		WObj.OpenChanSource = false
		WObj.Mutex.Unlock()
		close(WObj.ChanSource)
	}
}

func (WObj *webrtObj) SendChanSource(wResp *webrtResp) {
	if WObj.OpenChanSource {
		WObj.ChanSource <- wResp
	}
}

func (wr *WebrtcApi) webrtc(count int) *WebrtcApi {
	if !wr.valid {
		wr.Stack = map[uint64]*webrtHub{}

		wr.I = 1000

		wr.valid = true

		for i := 0; i < count; i++ {
			wr.neWHub()
		}
	}

	return wr
}

func (wrHub *webrtHub) isWebrtItemByObj(wrObj *webrtObj) bool {
	var key uint64
	components.СonvertAssign(&key, wrObj.Data.Get("key"))

	if _, ok := wrHub.Stack[key]; ok {
		return true
	}

	return false
}

func (wrHub *webrtHub) webrtItemByObj(wrObj *webrtObj) *webrtItem {
	var key uint64
	components.СonvertAssign(&key, wrObj.Data.Get("key"))

	if _, ok := wrHub.Stack[key]; ok {
		return wrHub.Stack[key]
	}

	wItem := &webrtItem{
		Key:          key,
		WHub:         wrHub,
		WObj:         wrObj,
		OfferChan:    make(chan *webrtObj),
		CompleteChan: make(chan error),
		Connections:  map[uint64]*webrtConnection{},
	}

	wrHub.Stack[key] = wItem

	wItem.run()

	return wItem
}

func (wItem *webrtItem) delConn(conn_i uint64) {
	if _, ok := wItem.Connections[conn_i]; ok {
		wItem.Connections[conn_i].Connection.Close()
		wItem.Mutex.Lock()
		delete(wItem.Connections, conn_i)
		wItem.Mutex.Unlock()
	}
}

func (wItem *webrtItem) runPeer(traks map[string]webrtc.TrackLocal, cb_connect func(), cb_close func()) {
	switch wItem.WObj.Action {
	default:
		for {
			select {
			case WObj := <-wItem.OfferChan:
				fmt.Println("runPeer OfferChan")

				offer := webrtc.SessionDescription{}
				var local_session string

				components.СonvertAssign(&local_session, WObj.Data.Get("local_session"))

				components.Decode(local_session, &offer)

				// Create a new PeerConnection
				peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
					ICEServers: []webrtc.ICEServer{
						{
							URLs: []string{"stun:stun.l.google.com:19302"},
						},
					},
				})
				if err != nil {
					WObj.CloseChanSource()
					continue
				}

				webrtConnection := wItem.NewWebrtConnection(peerConnection)

				isCbConnect := true
				isCbClose := true

				// Set the handler for ICE connection state
				// This will notify you when the peer has connected/disconnected
				webrtConnection.Connection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
					fmt.Printf("Connection State has changed %s \n", connectionState.String())
					if connectionState == webrtc.ICEConnectionStateConnected {
						if isCbConnect {
							isCbConnect = false
							cb_connect()
						}
					}
				})

				// Set the handler for Peer connection state
				// This will notify you when the peer has connected/disconnected
				webrtConnection.Connection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
					fmt.Printf("Peer Connection State has changed: %s\n", s.String())

					if s == webrtc.PeerConnectionStateDisconnected || s == webrtc.PeerConnectionStateFailed || s == webrtc.PeerConnectionStateFailed {
						if _, ok := wItem.Connections[webrtConnection.KeyI]; ok {
							if isCbClose {
								isCbClose = false
								WObj.CloseChanSource()
								wItem.delConn(webrtConnection.KeyI)
								cb_close()
							}
							return
						}
					}
				})

				for i, _ := range traks {
					rtpSender, trackErr := webrtConnection.Connection.AddTrack(traks[i])
					if trackErr == nil {
						go func() {
							rtcpBuf := make([]byte, 1500)
							for {
								if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
									return
								}
							}
						}()
					}
				}

				// Set the remote SessionDescription
				err = webrtConnection.Connection.SetRemoteDescription(offer)
				if err != nil {
					WObj.CloseChanSource()
					wItem.delConn(webrtConnection.KeyI)
					continue
				}

				// Create answer
				answer, err := webrtConnection.Connection.CreateAnswer(nil)
				if err != nil {
					WObj.CloseChanSource()
					wItem.delConn(webrtConnection.KeyI)
					continue
				}

				// Sets the LocalDescription, and starts our UDP listeners
				err = webrtConnection.Connection.SetLocalDescription(answer)
				if err != nil {
					WObj.CloseChanSource()
					wItem.delConn(webrtConnection.KeyI)
					continue
				}

				webrtConnection.Connection.OnICECandidate(func(c *webrtc.ICECandidate) {
					if c == nil {
						return
					}

					if WObj.OpenChanSource {
						if _, ok := wItem.Connections[webrtConnection.KeyI]; ok {
							wResp := &webrtResp{
								Action: "SessionDescription",
								Data:   components.NewData(),
							}

							wResp.Data.Set("remote_session", components.Encode(*webrtConnection.Connection.LocalDescription()))

							WObj.SendChanSource(wResp)

							WObj.CloseChanSource()
						}
					}
				})
			case <-wItem.CompleteChan:
				return
			}
		}
	}
}

func (wItem *webrtItem) start() {
	wItem.WHub.Mutex.Lock()
	wItem.WHub.Count++
	wItem.WHub.Mutex.Unlock()
}

func (wItem *webrtItem) complete() {
	var key uint64
	components.СonvertAssign(&key, wItem.WObj.Data.Get("key"))

	wItem.WHub.Mutex.Lock()
	wItem.WHub.Count--
	if _, ok := wItem.WHub.Stack[key]; ok {
		delete(wItem.WHub.Stack, key)
	}
	wItem.WHub.Mutex.Unlock()

	for i, _ := range wItem.Connections {
		wItem.delConn(i)
	}
}

func (wItem *webrtItem) run() {
	switch wItem.WObj.Action {
	case "cameraVideoSave":
		go func() {
			wItem.start()
			defer wItem.complete()

			var file_out string
			components.СonvertAssign(&file_out, wItem.WObj.Data.Get("file_out"))

			iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

			saver := newWebmSaver()

			// Create a MediaEngine object to configure the supported codec
			m := &webrtc.MediaEngine{}

			// Setup the codecs you want to use.
			// Only support VP8 and OPUS, this makes our WebM muxer code simpler
			if err := m.RegisterCodec(webrtc.RTPCodecParameters{
				RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/VP8", ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
				PayloadType:        96,
			}, webrtc.RTPCodecTypeVideo); err != nil {
				return
			}
			if err := m.RegisterCodec(webrtc.RTPCodecParameters{
				RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/opus", ClockRate: 48000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
				PayloadType:        111,
			}, webrtc.RTPCodecTypeAudio); err != nil {
				return
			}

			// Create the API object with the MediaEngine
			api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

			// Create a new RTCPeerConnection
			peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{
						URLs: []string{"stun:stun.l.google.com:19302"},
					},
				},
			})
			if err != nil {
				return
			}

			webrtConnection := wItem.NewWebrtConnection(peerConnection)

			// Set a handler for when a new remote track starts, this handler copies inbound RTP packets,
			// replaces the SSRC and sends them back
			webrtConnection.Connection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
				// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
				go func() {
					ticker := time.NewTicker(time.Second * 3)
					for range ticker.C {
						if _, ok := wItem.Connections[webrtConnection.KeyI]; ok {
							errSend := webrtConnection.Connection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
							if errSend != nil {
								fmt.Println(errSend)
							}
						} else {
							return
						}
					}
				}()

				fmt.Printf("Track has started, of type %d: %s \n", track.PayloadType(), track.Codec().RTPCodecCapability.MimeType)
				for {
					// Read RTP packets being sent to Pion
					rtp, _, readErr := track.ReadRTP()
					if readErr != nil {
						if readErr == io.EOF {
							return
						}
						return
					}
					switch track.Kind() {
					case webrtc.RTPCodecTypeAudio:
						saver.PushOpus(rtp)
					case webrtc.RTPCodecTypeVideo:
						saver.PushVP8(file_out, rtp)
					}
				}
			})
			// Set the handler for ICE connection state
			// This will notify you when the peer has connected/disconnected
			webrtConnection.Connection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
				fmt.Printf("Connection State has changed %s \n", connectionState.String())
			})

			webrtConnection.Connection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
				fmt.Printf("Peer Connection State has changed: %s\n", s.String())

				if s == webrtc.PeerConnectionStateDisconnected || s == webrtc.PeerConnectionStateFailed || s == webrtc.PeerConnectionStateFailed {
					if _, ok := wItem.Connections[webrtConnection.KeyI]; ok {
						wItem.WObj.CloseChanSource()
						wItem.delConn(webrtConnection.KeyI)
						iceConnectedCtxCancel()
						return
					}
				}
			})

			offer := webrtc.SessionDescription{}
			var local_session string

			components.СonvertAssign(&local_session, wItem.WObj.Data.Get("local_session"))

			components.Decode(local_session, &offer)
			// Set the remote SessionDescription
			err = webrtConnection.Connection.SetRemoteDescription(offer)
			if err != nil {
				return
			}

			// Create an answer
			answer, err := webrtConnection.Connection.CreateAnswer(nil)
			if err != nil {
				return
			}

			// Sets the LocalDescription, and starts our UDP listeners
			err = webrtConnection.Connection.SetLocalDescription(answer)
			if err != nil {
				return
			}

			webrtConnection.Connection.OnICECandidate(func(c *webrtc.ICECandidate) {
				if c == nil {
					return
				}

				if wItem.WObj.OpenChanSource {
					if _, ok := wItem.Connections[webrtConnection.KeyI]; ok {
						wResp := &webrtResp{
							Action: "SessionDescription",
							Data:   components.NewData(),
						}

						wResp.Data.Set("remote_session", components.Encode(*webrtConnection.Connection.LocalDescription()))

						wItem.WObj.SendChanSource(wResp)

						wItem.WObj.CloseChanSource()
					}
				}
			})

			var max_time time.Duration

			if wItem.WObj.Data.Is("max_time") {
				components.СonvertAssign(&max_time, wItem.WObj.Data.Get("max_time"))
			} else {
				max_time = 60 * 1 * time.Second
			}

			select {
			case <-iceConnectedCtx.Done():
			case <-time.After(max_time):
			}

			saver.Close()
			if _, ok := wItem.Connections[webrtConnection.KeyI]; ok {
				webrtConnection.Connection.Close()
				wItem.WObj.CloseChanSource()
			}

			fmt.Println("Track has stoped")
		}()
	case "storageVideoStream":
		go func() {
			wItem.start()
			defer wItem.complete()

			traks := map[string]webrtc.TrackLocal{}

			audioFileName := ""
			videoFileName := ""

			components.СonvertAssign(&audioFileName, wItem.WObj.Data.Get("audio"))
			components.СonvertAssign(&videoFileName, wItem.WObj.Data.Get("video"))

			// Assert that we have an audio or video file
			_, err := os.Stat(videoFileName)
			haveVideoFile := !os.IsNotExist(err)

			_, err = os.Stat(audioFileName)
			haveAudioFile := !os.IsNotExist(err)

			if !haveAudioFile && !haveVideoFile {
				return
			}

			iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

			var videoTrack *webrtc.TrackLocalStaticSample
			var videoTrackErr error

			var audioTrack *webrtc.TrackLocalStaticSample
			var audioTrackErr error

			if haveVideoFile {
				// Create a video track
				videoTrack, videoTrackErr = webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion")
				if videoTrackErr != nil {
					return
				}

				traks["video"] = videoTrack

				go func() {
					// Open a IVF file and start reading using our IVFReader
					file, ivfErr := os.Open(videoFileName)
					if ivfErr != nil {
						wItem.CompleteChan <- ivfErr
						return
					}

					ivf, header, ivfErr := ivfreader.NewWith(file)
					if ivfErr != nil {
						wItem.CompleteChan <- ivfErr
						return
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

							wItem.CompleteChan <- nil
							return
						}

						if ivfErr != nil {
							wItem.CompleteChan <- ivfErr
							return
						}

						if ivfErr = videoTrack.WriteSample(media.Sample{Data: frame, Duration: time.Second}); ivfErr != nil {
							wItem.CompleteChan <- ivfErr
							return
						}
					}
				}()
			}

			if haveAudioFile {
				// Create a audio track
				audioTrack, audioTrackErr = webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
				if audioTrackErr != nil {
					return
				}

				traks["audio"] = audioTrack

				go func() {
					// Open a OGG file and start reading using our OGGReader
					file, oggErr := os.Open(audioFileName)
					if oggErr != nil {
						wItem.CompleteChan <- oggErr
						return
					}

					// Open on oggfile in non-checksum mode.
					ogg, _, oggErr := oggreader.NewWith(file)
					if oggErr != nil {
						wItem.CompleteChan <- oggErr
						return
					}

					// Wait for connection established
					<-iceConnectedCtx.Done()

					// Keep track of last granule, the difference is the amount of samples in the buffer
					var lastGranule uint64

					// It is important to use a time.Ticker instead of time.Sleep because
					// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
					// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)
					ticker := time.NewTicker(time.Millisecond * 20)
					for ; true; <-ticker.C {
						pageData, pageHeader, oggErr := ogg.ParseNextPage()
						if errors.Is(oggErr, io.EOF) {
							fmt.Printf("All audio pages parsed and sent")

							wItem.CompleteChan <- nil
							return
						}

						if oggErr != nil {
							wItem.CompleteChan <- oggErr
							return
						}

						// The amount of samples is the difference between the last and current timestamp
						sampleCount := float64(pageHeader.GranulePosition - lastGranule)
						lastGranule = pageHeader.GranulePosition
						sampleDuration := time.Duration((sampleCount/48000)*1000) * time.Millisecond

						if oggErr = audioTrack.WriteSample(media.Sample{Data: pageData, Duration: sampleDuration}); oggErr != nil {
							wItem.CompleteChan <- oggErr
							return
						}
					}
				}()
			}

			wItem.runPeer(traks, func() {
				iceConnectedCtxCancel()
			}, func() {})
		}()
	case "cameraVideoStream":
		go func() {
			wItem.start()
			defer wItem.complete()

			traks := map[string]webrtc.TrackLocal{}

			iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())
			startConnectedCtx, startConnectedCtxCancel := context.WithCancel(context.Background())

			// Create a new RTCPeerConnection
			peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{
						URLs: []string{"stun:stun.l.google.com:19302"},
					},
				},
			})
			if err != nil {
				fmt.Println(err)
				return
			}

			webrtConnection := wItem.NewWebrtConnection(peerConnection)

			// Set a handler for when a new remote track starts, this handler copies inbound RTP packets,
			// replaces the SSRC and sends them back
			webrtConnection.Connection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
				// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
				// This can be less wasteful by processing incoming RTCP events, then we would emit a NACK/PLI when a viewer requests it
				go func() {
					ticker := time.NewTicker(time.Second * 3)
					for range ticker.C {
						if _, ok := wItem.Connections[webrtConnection.KeyI]; ok {
							if rtcpSendErr := webrtConnection.Connection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(remoteTrack.SSRC())}}); rtcpSendErr != nil {
								fmt.Println(rtcpSendErr)
							}
						} else {
							return
						}
					}
				}()

				fmt.Printf("Track has started, of type %d: %s \n", remoteTrack.PayloadType(), remoteTrack.Codec().RTPCodecCapability.MimeType)

				// Create a local track, all our SFU clients will be fed via this track
				localTrack, newTrackErr := webrtc.NewTrackLocalStaticRTP(remoteTrack.Codec().RTPCodecCapability, remoteTrack.Codec().RTPCodecCapability.MimeType, "pion")
				if newTrackErr != nil {
					fmt.Println(newTrackErr)
					return
				}

				traks[remoteTrack.Codec().RTPCodecCapability.MimeType] = localTrack

				if len(traks) == 1 {
					startConnectedCtxCancel()
				}

				rtpBuf := make([]byte, 1400)
				for {
					i, _, readErr := remoteTrack.Read(rtpBuf)
					if readErr != nil {
						fmt.Println(readErr)
						return
					}

					// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
					if _, err = localTrack.Write(rtpBuf[:i]); err != nil && !errors.Is(err, io.ErrClosedPipe) {
						fmt.Println(err)
						return
					}
				}
			})
			// Set the handler for ICE connection state
			// This will notify you when the peer has connected/disconnected
			webrtConnection.Connection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
				fmt.Printf("Connection State has changed %s \n", connectionState.String())
			})

			webrtConnection.Connection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
				fmt.Printf("Peer Connection State has changed: %s\n", s.String())

				if s == webrtc.PeerConnectionStateDisconnected || s == webrtc.PeerConnectionStateFailed || s == webrtc.PeerConnectionStateFailed {
					if _, ok := wItem.Connections[webrtConnection.KeyI]; ok {
						wItem.WObj.CloseChanSource()
						wItem.delConn(webrtConnection.KeyI)
						iceConnectedCtxCancel()
						return
					}
				}
			})

			offer := webrtc.SessionDescription{}
			var local_session string

			components.СonvertAssign(&local_session, wItem.WObj.Data.Get("local_session"))

			components.Decode(local_session, &offer)
			// Set the remote SessionDescription
			err = webrtConnection.Connection.SetRemoteDescription(offer)
			if err != nil {
				fmt.Println(err)
				return
			}

			// Create an answer
			answer, err := webrtConnection.Connection.CreateAnswer(nil)
			if err != nil {
				fmt.Println(err)
				return
			}

			// Sets the LocalDescription, and starts our UDP listeners
			err = webrtConnection.Connection.SetLocalDescription(answer)
			if err != nil {
				fmt.Println(err)
				return
			}

			webrtConnection.Connection.OnICECandidate(func(c *webrtc.ICECandidate) {
				if c == nil {
					return
				}

				if wItem.WObj.OpenChanSource {
					if _, ok := wItem.Connections[webrtConnection.KeyI]; ok {
						wResp := &webrtResp{
							Action: "SessionDescription",
							Data:   components.NewData(),
						}

						wResp.Data.Set("remote_session", components.Encode(*webrtConnection.Connection.LocalDescription()))

						wItem.WObj.SendChanSource(wResp)

						wItem.WObj.CloseChanSource()
					}
				}
			})

			var max_time time.Duration

			if wItem.WObj.Data.Is("max_time") {
				components.СonvertAssign(&max_time, wItem.WObj.Data.Get("max_time"))
			} else {
				max_time = 60 * 60 * time.Second
			}

			<-startConnectedCtx.Done()

			go func() {
				wItem.runPeer(traks, func() {
				}, func() {
					iceConnectedCtxCancel()
				})
			}()

			select {
			case <-iceConnectedCtx.Done():
			case <-time.After(max_time):
			}

			if _, ok := wItem.Connections[webrtConnection.KeyI]; ok {
				webrtConnection.Connection.Close()
				wItem.WObj.CloseChanSource()
			}

			fmt.Println("Track has stoped")
		}()
	case "WebrtcChannelsSessionGet":
		go func() {
			wItem.start()
			defer wItem.complete()

			iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

			// Create a new RTCPeerConnection
			peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{
						URLs: []string{"stun:stun.l.google.com:19302"},
					},
				},
			})
			if err != nil {
				fmt.Println(err)
				return
			}

			webrtConnection := wItem.NewWebrtConnection(peerConnection)

			dc := NewControllerDataChannel()

			// Set a handler for when a new remote track starts, this handler copies inbound RTP packets,
			// replaces the SSRC and sends them back
			webrtConnection.Connection.OnDataChannel(func(d *webrtc.DataChannel) {
				webrtConnection.DataChannel = d

				fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

				// Register channel opening handling
				d.OnOpen(func() {
					dc.OnConnect(webrtConnection)
				})

				// Register text message handling
				d.OnMessage(func(msg webrtc.DataChannelMessage) {
					dc.OnMessage(webrtConnection, msg.Data)
				})

				d.OnClose(func() {
					dc.OnClose(webrtConnection)
				})
			})

			// Set the handler for ICE connection state
			// This will notify you when the peer has connected/disconnected
			webrtConnection.Connection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
				fmt.Printf("Connection State has changed %s \n", connectionState.String())
			})

			webrtConnection.Connection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
				fmt.Printf("Peer Connection State has changed: %s\n", s.String())

				if s == webrtc.PeerConnectionStateDisconnected || s == webrtc.PeerConnectionStateFailed || s == webrtc.PeerConnectionStateFailed {
					if _, ok := wItem.Connections[webrtConnection.KeyI]; ok {
						wItem.WObj.CloseChanSource()
						wItem.delConn(webrtConnection.KeyI)
						iceConnectedCtxCancel()
						return
					}
				}
			})

			offer := webrtc.SessionDescription{}
			var local_session string

			components.СonvertAssign(&local_session, wItem.WObj.Data.Get("local_session"))

			components.Decode(local_session, &offer)
			// Set the remote SessionDescription
			err = webrtConnection.Connection.SetRemoteDescription(offer)
			if err != nil {
				fmt.Println(err)
				return
			}

			// Create an answer
			answer, err := webrtConnection.Connection.CreateAnswer(nil)
			if err != nil {
				fmt.Println(err)
				return
			}

			// Sets the LocalDescription, and starts our UDP listeners
			err = webrtConnection.Connection.SetLocalDescription(answer)
			if err != nil {
				fmt.Println(err)
				return
			}

			webrtConnection.Connection.OnICECandidate(func(c *webrtc.ICECandidate) {
				if c == nil {
					return
				}

				if wItem.WObj.OpenChanSource {
					if _, ok := wItem.Connections[webrtConnection.KeyI]; ok {
						wResp := &webrtResp{
							Action: "SessionDescription",
							Data:   components.NewData(),
						}

						wResp.Data.Set("remote_session", components.Encode(*webrtConnection.Connection.LocalDescription()))

						wItem.WObj.SendChanSource(wResp)

						wItem.WObj.CloseChanSource()
					}
				}
			})

			<-iceConnectedCtx.Done()
		}()
	default:
		wItem.WObj.CloseChanSource()
	}
}

func (wr *WebrtcApi) neWHub() {
	key := uint64(time.Now().Unix())

	wr.Stack[key] = &webrtHub{
		Count:     0,
		Key:       key,
		ChanStack: make(chan *webrtObj),
		Conn_i:    0,
		Stack:     map[uint64]*webrtItem{},
	}

	go wr.Stack[key].RunHub()
}

func (wrHub *webrtHub) RunHub() {
	for {
		select {
		case WObj := <-wrHub.ChanStack:
			switch WObj.Action {
			case "cameraVideoSave":
				wrHub.webrtItemByObj(WObj)
			case "cameraVideoStream":
				if WObj.Data.Is("action.get") {
					if wrHub.isWebrtItemByObj(WObj) {
						wItem := wrHub.webrtItemByObj(WObj)
						wItem.OfferChan <- WObj
					} else {
						wResp := &webrtResp{
							Action: "Error",
							Data:   components.NewData(),
						}

						wResp.Data.Set("error", "Camera no set")

						WObj.SendChanSource(wResp)

						WObj.CloseChanSource()
					}
				} else {
					wrHub.webrtItemByObj(WObj)
				}
			case "WebrtcChannelsSessionGet":
				wrHub.webrtItemByObj(WObj)
			default:
				wItem := wrHub.webrtItemByObj(WObj)

				wItem.OfferChan <- WObj
			}
		}
	}
}

func Webrtc() (*WebrtcApi, error) {
	if config.Env("WEBRTC_SERVER_CONNECT") == "true" || config.Env("WEBRTC_SERVER_CONNECT") == "1" {
		return webrtcApp.webrtc(50), nil
	}

	return nil, fmt.Errorf("Webrtc connection is prohibited on this server")
}

func WebrtHub() (*webrtHub, error) {
	wr, err := Webrtc()

	if err != nil {
		return nil, err
	}

	var hub *webrtHub

	for i, _ := range wr.Stack {
		hub = wr.Stack[i]
		break
	}

	for i, _ := range wr.Stack {
		if wr.Stack[i].Count < hub.Count {
			hub = wr.Stack[i]
		}
	}

	return hub, nil
}

func WebrtHubByObj(wrObj *webrtObj) (*webrtHub, error) {
	wr, err := Webrtc()

	if err != nil {
		return nil, err
	}

	var key uint64

	components.СonvertAssign(&key, wrObj.Data.Get("key"))

	if key > 0 {
		for i, _ := range wr.Stack {
			for j, _ := range wr.Stack[i].Stack {
				if j == key {
					return wr.Stack[i], nil
				}
			}
		}
	}

	atomic.AddUint64(&wr.I, 1)

	wrObj.Data.Set("key", wr.I)

	return WebrtHub()
}

func (wrConn *webrtConnection) Key() string {
	return fmt.Sprintf("webrtc:%d:%d:%d", wrConn.WItem.WHub.Key, wrConn.WItem.Key, wrConn.KeyI)
}

func (wrConn *webrtConnection) Send(key string, data any) {
	wr, err := Webrtc()

	if err != nil {
		return
	}

	wr.Send(key, data)
}

func (wrConn *webrtConnection) SendAll(data any) {
	wr, err := Webrtc()

	if err != nil {
		return
	}

	wr.SendAll(data)
}

func (wr *WebrtcApi) Send(key string, data any) {
	splitKey := strings.Split(key, ":")

	if len(splitKey) == 4 {
		if splitKey[0] == "webrtc" {
			if wHubKey, err := strconv.ParseUint(splitKey[1], 10, 64); err == nil {
				if wItemKey, err := strconv.ParseUint(splitKey[2], 10, 64); err == nil {
					if wConnKeyI, err := strconv.ParseUint(splitKey[3], 10, 64); err == nil {
						if _, ok := wr.Stack[wHubKey]; ok {
							if _, ok := wr.Stack[wHubKey].Stack[wItemKey]; ok {
								if _, ok := wr.Stack[wHubKey].Stack[wItemKey].Connections[wConnKeyI]; ok {
									if wr.Stack[wHubKey].Stack[wItemKey].Connections[wConnKeyI].DataChannel != nil {
										var databytes []byte

										components.СonvertAssign(&databytes, data)

										wr.Stack[wHubKey].Stack[wItemKey].Connections[wConnKeyI].DataChannel.Send(databytes)
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func (wr *WebrtcApi) SendAll(data any) {
	var databytes []byte

	components.СonvertAssign(&databytes, data)

	for wHubKey, _ := range wr.Stack {
		for wItemKey, _ := range wr.Stack[wHubKey].Stack {
			if wItemKey > 1000 {
				for wConnKeyI, _ := range wr.Stack[wHubKey].Stack[wItemKey].Connections {
					if wr.Stack[wHubKey].Stack[wItemKey].Connections[wConnKeyI].DataChannel != nil {
						wr.Stack[wHubKey].Stack[wItemKey].Connections[wConnKeyI].DataChannel.Send(databytes)
					}
				}
			}
		}
	}
}

func WebrtcSendAll(data any) {
	wr, err := Webrtc()

	if err != nil {
		return
	}

	wr.SendAll(data)
}

func WebrtcSend(key string, data any) {
	wr, err := Webrtc()

	if err != nil {
		return
	}

	wr.Send(key, data)
}
