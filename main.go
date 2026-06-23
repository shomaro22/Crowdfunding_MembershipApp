package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

func main() {
	var (
		e    *echo.Echo
		addr string
		err  error
	)
	loadDotEnv()
	e, addr, err = buildApp()
	if err != nil {
		log.Fatal(err)
	}
	err = e.Start(addr)
	if err != nil {
		log.Fatal(err)
	}
}

func loadDotEnv() {
	const envFile = ".env"
	var err error
	_, err = os.Stat(envFile)
	if err != nil {
		return
	}
	err = godotenv.Load(envFile)
	if err != nil {
		log.Printf("warning: load .env: %v", err)
	}
}
