package duct

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	dc "github.com/fsouza/go-dockerclient"
)

// Container describes a container
type Container struct {
	Env          []string
	PostCommands [][]string // run in container image after the command is called
	Command      []string
	Entrypoint   []string
	Image        string
	BindMounts   map[string]string
	LocalImage   bool
	BootWait     time.Duration

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
func (c *Composer) Launch(ctx context.Context) error {
	client, err := dc.NewClientFromEnv()
	if err != nil {
		return err
	}

	for name, cont := range c.manifest {
		if !cont.LocalImage {
			log.Printf("Pulling docker image: [%s]", cont.Image)

			if err := client.PullImage(dc.PullImageOptions{Repository: cont.Image}, dc.AuthConfiguration{}); err != nil {
				c.Teardown(ctx)
				return err
			}
		}

		mounts := []dc.HostMount{}
		for host, target := range cont.BindMounts {
			if !filepath.IsAbs(host) {
				host, err = filepath.Abs(host)
				if err != nil {
					c.Teardown(ctx)
					return err
				}
			}

			mounts = append(mounts, dc.HostMount{
				Source: host,
				Type:   "bind",
				Target: target,
			})
		}

		log.Printf("Creating container: [%s]", name)
		ctr, err := client.CreateContainer(dc.CreateContainerOptions{
			Name: name,
			Config: &dc.Config{
				Hostname:   name,
				Image:      cont.Image,
				Env:        cont.Env,
				Cmd:        cont.Command,
				Entrypoint: cont.Entrypoint,
			},
			HostConfig: &dc.HostConfig{
				Mounts: mounts,
			},
			Context: ctx,
		})
		if err != nil {
			c.Teardown(ctx)
			return err
		}

		c.launched = true
		cont.id = ctr.ID
	}

	for name, cont := range c.manifest {
		log.Printf("Starting container: [%s]", name)
		if err := client.StartContainerWithContext(cont.id, nil, ctx); err != nil {
			c.Teardown(ctx)
			return err
		}

		if cont.BootWait != 0 {
			log.Printf("Sleeping for %v (requested by %q bootWait parameter)", cont.BootWait, name)
			time.Sleep(cont.BootWait)
		}

		for _, command := range cont.PostCommands {
			log.Printf("Running post-command [%s] in container: [%s]", strings.Join(command, " "), name)
			exec, err := client.CreateExec(dc.CreateExecOptions{
				Context:      ctx,
				Container:    cont.id,
				Cmd:          command,
				AttachStderr: true,
				AttachStdout: true,
			})
			if err != nil {
				c.Teardown(ctx)
				return err
			}

			err = client.StartExec(exec.ID, dc.StartExecOptions{
				OutputStream: os.Stdout,
				ErrorStream:  os.Stderr,
				Context:      ctx,
			})
			if err != nil {
				c.Teardown(ctx)
				return err
			}
		}
	}

	return nil
}

// Teardown kills the container processes in the manifest and removes their containers
func (c *Composer) Teardown(ctx context.Context) error {
	if !c.launched {
		return errors.New("containers have not launched")
	}

	client, err := dc.NewClientFromEnv()
	if err != nil {
		return err
	}

	var errs bool

	for name, cont := range c.manifest {
		if cont.id != "" {
			log.Printf("Killing container: [%s]", name)
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

			log.Printf("Removing container: [%s]", name)
			if err := client.RemoveContainer(dc.RemoveContainerOptions{ID: cont.id, Force: true, Context: ctx}); err != nil {
				log.Println(err)
				errs = true
				continue
			}
		} else {
			log.Printf("Skipping unstarted container: [%s]", name)
		}
	}

	if errs {
		return errors.New("there were errors (see log)")
	}

	c.launched = false
	return nil
}
