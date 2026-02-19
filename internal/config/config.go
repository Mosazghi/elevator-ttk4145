package config

import (
	"flag"
)

type config struct {
	bcastInterval float64
	nodeTimeout   float64
	id            int
	port          int
}

// Parse parses commandline arguments and returns them in the form of a struct
func Parse() config {

	bcastInterval := flag.Float64("bcastInterval", 200, "Broadcast Interval")
	nodeTimeout := flag.Float64("nodeTimeout", 10, "Node timeout delay")
	id := flag.Int("id", 1, "Node ID")
	port := flag.Int("port", 10000, "Broadcast port")

	flag.Parse()

	cfg := config{
		bcastInterval: *bcastInterval,
		nodeTimeout:   *nodeTimeout,
		id:            *id,
		port:          *port,
	}
	return cfg
}
