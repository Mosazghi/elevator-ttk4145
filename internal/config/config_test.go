package config

import (
	"flag"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse_Defaults(t *testing.T) {
	// Reset flag set for testing
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	// Save and restore os.Args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// No additional args, use defaults
	os.Args = []string{"program"}

	cfg := Parse()

	assert.Equal(t, 1, cfg.Id)
	assert.Equal(t, 30000, cfg.Port)
	slog.Info("[Default values]", "port", cfg.Port, "id", cfg.Id)
}

func TestParse_Id(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"program", "-id", "42"}

	cfg := Parse()

	assert.Equal(t, 42, cfg.Id)
	assert.Equal(t, 30000, cfg.Port)
}

func TestParse_Port(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"program", "-port", "8080"}

	cfg := Parse()

	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, 1, cfg.Id)
}

func TestParse_All(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"program", "-id", "3", "-port", "9999"}

	cfg := Parse()

	assert.Equal(t, 3, cfg.Id)
	assert.Equal(t, 9999, cfg.Port)
}
