package main

import (
	"flag"

	"github.com/DITAS-Project/VDC-Request-Monitor/monitor"
	log "github.com/sirupsen/logrus"
)

func main() {

	var configDir string
	flag.StringVar(&configDir, "dir", ".config/", "main config directory")

	mon, err := monitor.NewManger(configDir)

	if err != nil {
		log.Fatalf("could not start request monitor %+v", err)
	}

	mon.Listen()
}
