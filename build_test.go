package duct

import (
	"context"
	"testing"
	"time"
)

func TestBuild(t *testing.T) {
	b := Builder{
		"test-image": {
			Dockerfile: "testdata/Dockerfile.test",
			Context:    ".",
		},
	}

	if err := b.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	c := New(Manifest{
		"test-image": {
			Image:      "test-image",
			LocalImage: true,
		},
	})

	if err := c.Launch(time.Minute); err != nil {
		t.Fatal(err)
	}

	if err := c.Teardown(time.Minute); err != nil {
		t.Fatal(err)
	}
}
