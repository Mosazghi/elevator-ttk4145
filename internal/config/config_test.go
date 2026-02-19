package config

import (
	"flag"
	"fmt"
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

	if cfg.bcastInterval != 200 {
		t.Errorf("expected bcastInterval 200, got %f", cfg.bcastInterval)
	}
	if cfg.nodeTimeout != 10 {
		t.Errorf("expected nodeTimeout 10, got %f", cfg.nodeTimeout)
	}
	if cfg.id != 1 {
		t.Errorf("expected id 1, got %d", cfg.id)
	}
	if cfg.port != 10000 {
		t.Errorf("expected port 10000, got %d", cfg.port)
	}
}

func TestParse_BcastInterval(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"program", "-bcastInterval", "150.5"}

	cfg := Parse()

	if cfg.bcastInterval != 150.5 {
		t.Errorf("expected bcastInterval 150.5, got %f", cfg.bcastInterval)
	}
	// Other fields should be defaults
	if cfg.nodeTimeout != 10 {
		t.Errorf("expected nodeTimeout 10, got %f", cfg.nodeTimeout)
	}
	if cfg.id != 1 {
		t.Errorf("expected id 1, got %d", cfg.id)
	}
	if cfg.port != 10000 {
		t.Errorf("expected port 10000, got %d", cfg.port)
	}
}

func TestParse_NodeTimeout(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"program", "-nodeTimeout", "5.5"}

	cfg := Parse()

	if cfg.nodeTimeout != 5.5 {
		t.Errorf("expected nodeTimeout 5.5, got %f", cfg.nodeTimeout)
	}
	if cfg.bcastInterval != 200 {
		t.Errorf("expected bcastInterval 200, got %f", cfg.bcastInterval)
	}
	if cfg.id != 1 {
		t.Errorf("expected id 1, got %d", cfg.id)
	}
	if cfg.port != 10000 {
		t.Errorf("expected port 10000, got %d", cfg.port)
	}
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
	if cfg.bcastInterval != 200 {
		t.Errorf("expected bcastInterval 200, got %f", cfg.bcastInterval)
	}
	if cfg.nodeTimeout != 10 {
		t.Errorf("expected nodeTimeout 10, got %f", cfg.nodeTimeout)
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
	if cfg.bcastInterval != 200 {
		t.Errorf("expected bcastInterval 200, got %f", cfg.bcastInterval)
	}
	if cfg.nodeTimeout != 10 {
		t.Errorf("expected nodeTimeout 10, got %f", cfg.nodeTimeout)
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

	os.Args = []string{"program", "-bcastInterval", "300", "-nodeTimeout", "20", "-id", "3", "-port", "9999"}

	cfg := Parse()

	if cfg.bcastInterval != 300 {
		t.Errorf("expected bcastInterval 300, got %f", cfg.bcastInterval)
	}
	if cfg.nodeTimeout != 20 {
		t.Errorf("expected nodeTimeout 20, got %f", cfg.nodeTimeout)
	}
	if cfg.id != 3 {
		t.Errorf("expected id 3, got %d", cfg.id)
	}
	if cfg.port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.port)
	}
}
