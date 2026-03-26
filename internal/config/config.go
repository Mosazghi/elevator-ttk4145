package config

import (
	"flag"
	"log/slog"
)

// Config holds the configuration for the elevator node.
type Config struct {
	Id        int
	Port      int
	NumFloors int
	LogLevel  slog.Leveler
}

// Parse parses commandline arguments and returns them in the form of a struct
func Parse() Config {

	id := flag.Int("id", 1, "Node ID")
	port := flag.Int("port", defaultPort, "Broadcast port")
	numFloors := flag.Int("floors", defaultNumFloors, "Number of floors in the building")
	logLevel := flag.Int("loglevel", -4, "Log level (debug=-4, info=0, warn=4, error=8)")

	flag.Parse()

	cfg := Config{
		Id:        *id,
		Port:      *port,
		NumFloors: *numFloors,
		LogLevel:  slog.Level(*logLevel),
	}

	return cfg
}
