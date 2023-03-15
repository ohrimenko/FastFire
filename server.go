package main

import (
	"log"
	"strconv"
	"syscall"
	"time"

	"backnet/components"
	"backnet/config"
	"backnet/controllers/webrtc"
	"backnet/routes"

	"github.com/joho/godotenv"

	"net/http"

	"fmt"

	"math/rand"

	"context"
	"os"
	"os/signal"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	components.InitSerialize()

	// .env Variables validation
	if err := godotenv.Load("./.env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	if config.Env("WS_PONG_WAIT") != "" {
		n, err := strconv.Atoi(config.Env("WS_PONG_WAIT"))
		if err == nil {
			config.PongWait = time.Duration(n) * time.Second

			if config.PongWait > 0 {
				config.PingPeriod = int64((config.PongWait * 9) / 10)
			} else {
				config.PingPeriod = 50
			}
		}
	}

	_, err := components.DB()

	if err != nil {
		log.Fatal(err)
	}

	defer components.CloseDB()

	MuxRouterHTTP := components.RouteMux("http")
	MuxRouterWs := components.RouteMux("ws")
	MuxRouterSse := components.RouteMux("sse")

	routes.App.Http(MuxRouterHTTP)
	routes.App.Websocket(MuxRouterWs)
	routes.App.Sse(MuxRouterSse)
	routes.App.Webrtc(MuxRouterHTTP)

	if config.Env("HTTP_PORT") == config.Env("WS_PORT") {
		if config.Env("WS_SERVER_START") == "true" || config.Env("WS_SERVER_START") == "1" {
			routes.App.Websocket(MuxRouterHTTP)
		}
	}

	if config.Env("HTTPS_PORT") == config.Env("WSS_PORT") {
		if config.Env("WSS_SERVER_START") == "true" || config.Env("WSS_SERVER_START") == "1" {
			routes.App.Websocket(MuxRouterHTTP)
		}
	}

	if config.Env("HTTP_PORT") == config.Env("SSE_PORT") {
		if config.Env("SSE_SERVER_START") == "true" || config.Env("SSE_SERVER_START") == "1" {
			routes.App.Sse(MuxRouterHTTP)
		}
	}

	if config.Env("HTTPS_PORT") == config.Env("SSES_PORT") {
		if config.Env("SSES_SERVER_START") == "true" || config.Env("SSES_SERVER_START") == "1" {
			routes.App.Sse(MuxRouterHTTP)
		}
	}

	var srv *http.Server
	var srvTLS *http.Server
	var srvWs *http.Server
	var srvWss *http.Server
	var srvSse *http.Server
	var srvSses *http.Server

	if config.Env("HTTP_SERVER_START") == "true" || config.Env("HTTP_SERVER_START") == "1" {
		srv = &http.Server{
			Handler: MuxRouterHTTP,
			//Handler: controllers.RedirectToHTTPSRouter(MuxRouterHTTP), // Редирект на https
			Addr: ":" + config.Env("HTTP_PORT"),
			// Good practice: enforce timeouts for servers you create!
			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			IdleTimeout:  time.Second * 60,
		}

		// Run our server in a goroutine so that it doesn't block.
		go func() {
			if err := srv.ListenAndServe(); err != nil {
				log.Println(err)
			}
		}()
	}

	if config.Env("HTTPS_SERVER_START") == "true" || config.Env("HTTPS_SERVER_START") == "1" {
		srvTLS = &http.Server{
			Handler: MuxRouterHTTP,
			Addr:    ":" + config.Env("HTTPS_PORT"),

			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			IdleTimeout:  time.Second * 60,
		}

		go func() {
			if err := srvTLS.ListenAndServeTLS("certs/ssl.cert", "certs/ssl.key"); err != nil {
				log.Println(err)
			}
		}()
	}

	if config.Env("HTTP_PORT") != config.Env("WS_PORT") {
		if config.Env("WS_SERVER_START") == "true" || config.Env("WS_SERVER_START") == "1" {
			srvWs = &http.Server{
				Handler: MuxRouterWs,
				//Handler: controllers.RedirectToHTTPSRouter(MuxRouterHTTP), // Редирект на https
				Addr: ":" + config.Env("WS_PORT"),
				// Good practice: enforce timeouts for servers you create!
				WriteTimeout: time.Second * 15,
				ReadTimeout:  time.Second * 15,
				IdleTimeout:  time.Second * 60,
			}

			// Run our server in a goroutine so that it doesn't block.
			go func() {
				if err := srvWs.ListenAndServe(); err != nil {
					log.Println(err)
				}
			}()
		}
	}

	if config.Env("HTTPS_PORT") != config.Env("WSS_PORT") {
		if config.Env("WSS_SERVER_START") == "true" || config.Env("WSS_SERVER_START") == "1" {
			srvWss = &http.Server{
				Handler: MuxRouterWs,
				//Handler: controllers.RedirectToHTTPSRouter(MuxRouterHTTP), // Редирект на https
				Addr: ":" + config.Env("WSS_PORT"),
				// Good practice: enforce timeouts for servers you create!
				WriteTimeout: time.Second * 15,
				ReadTimeout:  time.Second * 15,
				IdleTimeout:  time.Second * 60,
			}

			// Run our server in a goroutine so that it doesn't block.
			go func() {
				if err := srvWss.ListenAndServeTLS("certs/ssl.cert", "certs/ssl.key"); err != nil {
					log.Println(err)
				}
			}()
		}
	}

	if config.Env("HTTP_PORT") != config.Env("SSE_PORT") {
		if config.Env("WS_SERVER_START") == "true" || config.Env("WS_SERVER_START") == "1" {
			srvSse = &http.Server{
				Handler: MuxRouterSse,
				//Handler: controllers.RedirectToHTTPSRouter(MuxRouterHTTP), // Редирект на https
				Addr: ":" + config.Env("SSE_PORT"),
				// Good practice: enforce timeouts for servers you create!
				WriteTimeout: time.Second * 3600 * 24 * 30,
				ReadTimeout:  0,
				IdleTimeout:  0,
			}

			// Run our server in a goroutine so that it doesn't block.
			go func() {
				if err := srvSse.ListenAndServe(); err != nil {
					log.Println(err)
				}
			}()
		}
	}

	if config.Env("HTTPS_PORT") != config.Env("SSES_PORT") {
		if config.Env("WSS_SERVER_START") == "true" || config.Env("WSS_SERVER_START") == "1" {
			srvSses = &http.Server{
				Handler: MuxRouterSse,
				//Handler: controllers.RedirectToHTTPSRouter(MuxRouterHTTP), // Редирект на https
				Addr: ":" + config.Env("SSES_PORT"),
				// Good practice: enforce timeouts for servers you create!
				WriteTimeout: time.Second * 3600 * 24 * 30,
				ReadTimeout:  0,
				IdleTimeout:  0,
			}

			// Run our server in a goroutine so that it doesn't block.
			go func() {
				if err := srvSses.ListenAndServeTLS("certs/ssl.cert", "certs/ssl.key"); err != nil {
					log.Println(err)
				}
			}()
		}
	}

	if config.Env("TURN_SERVER_START") == "true" || config.Env("TURN_SERVER_START") == "1" {
		portUdp, _ := strconv.Atoi(config.Env("TURN_SERVER_PORT_UDP"))
		portTcp, _ := strconv.Atoi(config.Env("TURN_SERVER_PORT_TCP"))
		portTls, _ := strconv.Atoi(config.Env("TURN_SERVER_PORT_TLS"))

		turnServer := webrtc.NewTurnServer(
			config.Env("TURN_SERVER_IP"),
			portUdp,
			portTcp,
			portTls,
			config.Env("TURN_SERVER_USERS"),
			config.Env("TURN_SERVER_REALM"),
			config.Env("TURN_SERVER_CERT_FILE"),
			config.Env("TURN_SERVER_KEY_FILE"))

		go turnServer.Run()
	}

	fmt.Println("Server start")

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGQUIT)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	if srv != nil {
		srv.Shutdown(ctx)
	}

	if srvTLS != nil {
		srvTLS.Shutdown(ctx)
	}

	if srvWs != nil {
		srvWs.Shutdown(ctx)
	}

	if srvWss != nil {
		srvWss.Shutdown(ctx)
	}

	if srvSse != nil {
		srvSse.Shutdown(ctx)
	}

	if srvSses != nil {
		srvSses.Shutdown(ctx)
	}

	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)
}
