package main

import (
	"log"

	"iq-home/backend/internal/app"
	"iq-home/backend/internal/config"
)

func main() {
	cfg := config.MustLoad()
	if err := app.New(cfg).Run(); err != nil {
		log.Fatal(err)
	}
}
