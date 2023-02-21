package main

import (
	"log"
	"time"

	"backnet/components"
	"backnet/config"
	"backnet/routes"

	"github.com/joho/godotenv"

	"net/http"

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

	_, err := components.DB()

	if err != nil {
		log.Fatal(err)
	}

	defer components.CloseDB()

	MuxRouterHTTP := components.RouteMux("http")

	routes.App.Http(MuxRouterHTTP)

	srv := &http.Server{
		Handler: MuxRouterHTTP,
		//Handler: controllers.RedirectToHTTPSRouter(MuxRouterHTTP), // Редирект на https
		Addr: ":" + config.Env("HTTP_PORT"),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}
	srvTLS := &http.Server{
		Handler: MuxRouterHTTP,
		Addr:    ":" + config.Env("HTTPS_PORT"),

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
	go func() {
		if err := srvTLS.ListenAndServeTLS("certs/ssl.cert", "certs/ssl.key"); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	srvTLS.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)
}
