package main

import (
	"AstralTest/internal/app"
	"log"
)

func main() {
	app, err := app.InitApp()
	if err != nil {
		log.Fatal("can't init app ", err)
	}

	app.Run()
}
