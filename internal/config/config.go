package config

import (
	"flag"
	"log/slog"
	"time"
)

const (
	DefaultPort   = 30000
	DefaultFloors = 4
	DoorOpenTime  = 3 * time.Second
)

type Config struct {
	Id       int
	Port     int
	Floors   int
	LogLevel slog.Leveler
}

// Parse parses commandline arguments and returns them in the form of a struct
func Parse() Config {

	id := flag.Int("id", 1, "Node ID")
	port := flag.Int("port", DefaultPort, "Broadcast port")
	floors := flag.Int("floors", DefaultFloors, "Number of floors in the building")
	logLevel := flag.Int("loglevel", -4, "Log level (debug=-4, info=0, warn=4, error=8)")

	flag.Parse()

	cfg := Config{
		Id:       *id,
		Port:     *port,
		Floors:   *floors,
		LogLevel: slog.Level(*logLevel),
	}

	return cfg
}
