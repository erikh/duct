package duct

import (
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	c := New(Manifest{
		"sleep": {
			Command: []string{"sleep", "1"},
			Image:   "alpine:latest",
		},
	})

	if err := c.Launch(time.Minute); err != nil {
		t.Fatal(err)
	}

	if err := c.Teardown(time.Minute); err != nil {
		t.Fatal(err)
	}
}
