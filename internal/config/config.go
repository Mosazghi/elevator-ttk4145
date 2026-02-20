package config

import (
	"flag"
)

type config struct {
	id   int
	port int
}

// Parse parses commandline arguments and returns them in the form of a struct
func Parse() config {

	id := flag.Int("id", 1, "Node ID")
	port := flag.Int("port", 30000, "Broadcast port")

	flag.Parse()

	cfg := config{
		id:   *id,
		port: *port,
	}
	return cfg
}
