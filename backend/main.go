package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"crowdfund-app/server"
)

func main() {
	server.LoadDotEnv()
	e, addr, err := server.BuildHTTP(nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	e.Logger.Info("shutdown server")
	_ = e.Shutdown(context.Background())
}
