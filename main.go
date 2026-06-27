package main

import (
	"log"

	"crowdfund-app/bootstrap"

	"github.com/labstack/echo/v4"
)

func main() {
	var (
		e    *echo.Echo
		addr string
		err  error
	)
	bootstrap.LoadDotEnv()
	e, addr, err = buildApp()
	if err != nil {
		log.Fatal(err)
	}
	err = e.Start(addr)
	if err != nil {
		log.Fatal(err)
	}
}
