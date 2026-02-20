package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"testing"
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

	if cfg.id != 1 {
		t.Errorf("expected id 1, got %d", cfg.id)
	}
	if cfg.port != 10000 {
		t.Errorf("expected port 10000, got %d", cfg.port)
	}
	slog.Info("[Default values]", "port", cfg.port, "id", cfg.id)
}




func TestParse_Id(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"program", "-id", "42"}

	cfg := Parse()

	if cfg.id != 42 {
		t.Errorf("expected id 42, got %d", cfg.id)
	}

	if cfg.port != 10000 {
		t.Errorf("expected port 10000, got %d", cfg.port)
	}
}

func TestParse_Port(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"program", "-port", "8080"}

	cfg := Parse()

	if cfg.port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.port)
	}
	if cfg.id != 1 {
		t.Errorf("expected id 1, got %d", cfg.id)
	}
	fmt.Printf("port: %v\n", cfg.port)
}

func TestParse_All(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"program", "-id", "3", "-port", "9999"}

	cfg := Parse()

	if cfg.id != 3 {
		t.Errorf("expected id 3, got %d", cfg.id)
	}
	if cfg.port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.port)
	}
}
