package duct

import (
	"context"
	"testing"
)

func TestBuild(t *testing.T) {
	b := Builder{
		"test-image": {
			Dockerfile: "testdata/Dockerfile.test",
			Context:    ".",
		},
		"test-image2": {
			Dockerfile: "testdata/Dockerfile.test",
		},
	}

	if err := b.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	c := New(Manifest{
		{
			Name:       "test-image",
			Image:      "test-image",
			LocalImage: true,
		},
	}, WithNewNetwork("duct-test-network"))

	if err := c.Launch(context.Background()); err != nil {
		t.Fatal(err)
	}

	if err := c.Teardown(context.Background()); err != nil {
		t.Fatal(err)
	}
}
