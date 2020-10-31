package duct

import (
	"context"
	"testing"
)

func TestBasic(t *testing.T) {
	c := New(Manifest{
		"sleep": {
			Command: []string{"sleep", "1"},
			Image:   "alpine:latest",
		},
	})

	if err := c.Launch(context.Background()); err != nil {
		t.Fatal(err)
	}

	if err := c.Teardown(context.Background()); err != nil {
		t.Fatal(err)
	}
}
