package duct

import (
	"context"
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	c := New(Manifest{
		"sleep": {
			Command: []string{"sleep", "1"},
			Image:   "debian:latest",
			PostCommands: [][]string{
				{"echo", "from post-command"},
				{"head", "-1", "/duct.go"},
			},
			BindMounts: map[string]string{
				"duct.go": "/duct.go",
			},
		},
	}, "duct-test-network")

	if err := c.Launch(context.Background()); err != nil {
		t.Fatal(err)
	}

	if err := c.Teardown(context.Background()); err != nil {
		t.Fatal(err)
	}

	c = New(Manifest{
		"early-terminator": {
			Command:  []string{"sleep", "1"},
			Image:    "debian:latest",
			BootWait: 2 * time.Second,
			PostCommands: [][]string{
				{"echo", "from post-command"},
			},
		},
	}, "duct-test-network")

	if err := c.Launch(context.Background()); err == nil {
		t.Fatal("launch succeeded; should not have")
	}

	if err := c.Teardown(context.Background()); err == nil {
		t.Fatal("teardown did not fail with an error")
	}

	// start it a second time to make sure it was cleaned up, this time it will
	// succeed to run
	c = New(Manifest{
		"early-terminator": {
			Command:  []string{"sleep", "3"},
			Image:    "debian:latest",
			BootWait: 2 * time.Second,
			PostCommands: [][]string{
				{"echo", "from post-command"},
			},
		},
	}, "duct-test-network")

	if err := c.Launch(context.Background()); err != nil {
		t.Fatal(err)
	}

	if err := c.Teardown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestNetwork(t *testing.T) {
	c := New(Manifest{
		"target": {
			Command: []string{"sleep", "infinity"},
			Image:   "debian:latest",
		},
		"pinger": {
			Command:      []string{"sleep", "infinity"},
			PostCommands: [][]string{{"ping", "-c", "1", "target"}},
			Image:        "debian:latest",
		},
	}, "duct-test-network")

	t.Cleanup(func() {
		if err := c.Teardown(context.Background()); err != nil {
			t.Fatal(err)
		}
	})

	if err := c.Launch(context.Background()); err != nil {
		t.Fatal(err)
	}
}
