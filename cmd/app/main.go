package main

import (
	"log"

	config "mdata/configs"
	"mdata/internal/app"

	_ "net/http/pprof"
)

func main() {

	// Configuration
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	app.Run(cfg)

}
