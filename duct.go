package duct

import (
	"context"
	"errors"
	"log"
	"time"

	dc "github.com/fsouza/go-dockerclient"
)

// Container describes a container
type Container struct {
	Env        []string
	Command    []string
	Entrypoint []string
	Image      string
	LocalImage bool

	id string
}

// Manifest is the mapping of name -> container
type Manifest map[string]*Container

// Composer is the interface to launching manifests
type Composer struct {
	manifest Manifest
	launched bool
}

// New constructs a new Composer from a manifest
func New(manifest Manifest) *Composer {
	return &Composer{manifest: manifest}
}

// Launch launches the manifest
func (c *Composer) Launch(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, err := dc.NewClientFromEnv()
	if err != nil {
		return err
	}

	for name, cont := range c.manifest {
		if !cont.LocalImage {
			if err := client.PullImage(dc.PullImageOptions{Repository: cont.Image}, dc.AuthConfiguration{}); err != nil {
				c.Teardown(timeout)
				return err
			}
		}

		ctr, err := client.CreateContainer(dc.CreateContainerOptions{
			Name: name,
			Config: &dc.Config{
				Hostname:   name,
				Image:      cont.Image,
				Env:        cont.Env,
				Cmd:        cont.Command,
				Entrypoint: cont.Entrypoint,
			},
			Context: ctx,
		})
		if err != nil {
			c.Teardown(timeout)
			return err
		}

		c.launched = true
		cont.id = ctr.ID
	}

	for _, cont := range c.manifest {
		if err := client.StartContainerWithContext(cont.id, nil, ctx); err != nil {
			c.Teardown(timeout)
			return err
		}
	}

	return nil
}

// Teardown kills the container processes in the manifest and removes their containers
func (c *Composer) Teardown(timeout time.Duration) error {
	if !c.launched {
		return errors.New("containers have not launched")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, err := dc.NewClientFromEnv()
	if err != nil {
		return err
	}

	var errs bool

	for _, cont := range c.manifest {
		if cont.id != "" {
			err := client.KillContainer(dc.KillContainerOptions{
				ID:      cont.id,
				Signal:  dc.SIGKILL,
				Context: ctx,
			})
			if err != nil {
				log.Println(err)
				errs = true
				continue
			}

			if err := client.RemoveContainer(dc.RemoveContainerOptions{ID: cont.id, Force: true, Context: ctx}); err != nil {
				log.Println(err)
				errs = true
				continue
			}
		}
	}

	if errs {
		return errors.New("there were errors (see log)")
	}

	return nil
}
