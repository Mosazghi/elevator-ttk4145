package config

import (
	"flag"
)

const (
	DefaultPort   = 30000
	DefaultFloors = 4
)

type Config struct {
	Id     int
	Port   int
	Floors int
}

// Parse parses commandline arguments and returns them in the form of a struct
func Parse() Config {

	id := flag.Int("id", 1, "Node ID")
	port := flag.Int("port", DefaultPort, "Broadcast port")
	floors := flag.Int("floors", DefaultFloors, "Number of floors in the building")

	flag.Parse()

	cfg := Config{
		Id:     *id,
		Port:   *port,
		Floors: *floors,
	}

	return cfg
}
