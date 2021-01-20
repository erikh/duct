package duct

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	dc "github.com/fsouza/go-dockerclient"
	"golang.org/x/sys/unix"
)

// Container is the description of a single container. Usually several of these
// are composed in a Manifest and sent to the New() call. Please see the fields
// below for more information.
type Container struct {
	// Name is required; it is the independent name of the container. It maps
	// directly to the name on the docker installation, so be mindful of
	// collisions.
	Name string

	// Env is the array of key=value string pairs in `man 7 environ` fashion.
	Env []string

	// PostCommands is a series of argvs for running commands after the container
	// is booted, and after the bootwait is consumed.
	PostCommands [][]string

	// Command is the command to run as the booted container.
	Command []string

	// Entrypoint maps directly to Docker's entrypoint.
	Entrypoint []string

	// Image is the docker image; it uses repository syntax, and will attempt to
	// pull it unless LocalImage is set true.
	Image string

	// BindMounts is a map of absolute path -> absolute path for host ->
	// container bind mounting.
	BindMounts map[string]string

	// LocalImage indicates this image is not to be pulled.
	LocalImage bool

	// BootWait is how long to wait after booting the container before moving
	// forward with PostCommands and other orchestration.
	BootWait time.Duration

	// AliveFunc is a locally run golang function for testing the availability of
	// the container. The client is passed in as well as the container ID to
	// assist with this process.
	AliveFunc func(context.Context, *dc.Client, string) error

	// PortForwards are a simple mapping of host -> container port mappings that
	// forward the port on 0.0.0.0 automatically.
	PortForwards map[int]int

	id string // the container id
}

// Manifest is the containers to run, in order. Passed to New().
type Manifest []*Container

// Composer is the interface to launching manifests. This is returned from
// New()
type Composer struct {
	manifest  Manifest
	options   Options
	netID     string
	sigCancel context.CancelFunc
}

// New constructs a new Composer from a Manifest. A network name must also be
// provided; it will be created and cleaned up when Run and Teardown are
// called.
func New(manifest Manifest, options ...Options) *Composer {

	// Gather options
	opts := Options{}

	for _, o := range options {
		for k, v := range o {
			opts[k] = v
		}
	}

	return &Composer{manifest: manifest, options: opts}
}

// Options is a generic type for options.
type Options map[string]interface{}

const (
	optionCreateNetwork   = "create_network"
	optionExistingNetwork = "existing_network"
	optionLogWriter       = "log_writer"
)

// WithNewNetwork creates a network for use with the manifest.
func WithNewNetwork(name string) Options {
	return Options{optionCreateNetwork: name}
}

// WithExistingNetwork uses an existing network by ID (*not* name, since
// network names are not unique!)
func WithExistingNetwork(id string) Options {
	return Options{optionExistingNetwork: id}
}

// WithLogWriter routes all logging output to the specified writer, or to none
// if nil is pecified
func WithLogWriter(writer io.Writer) Options {
	return Options{optionLogWriter: writer}
}

// HandleSignals handles SIGINT and SIGTERM to ensure that containers get
// cleaned up. It is expected that no other signal handler will be installed
// afterwards. If the forward argument is true, it will forward the signal back
// to its own process after deregistering itself as the signal handler,
// allowing your test suite to exit gracefully. Set it to false to stay out of
// your way.
func (c *Composer) HandleSignals(forward bool) {
	ctx, cancel := context.WithCancel(context.Background())
	c.sigCancel = cancel
	sigChan := make(chan os.Signal, 2)

	go func() {
		select {
		case sig := <-sigChan:
			log.Println("Signalled; will terminate containers now")
			c.Teardown(context.Background())
			signal.Stop(sigChan) // stop letting us get notified
			if forward {
				unix.Kill(os.Getpid(), sig.(syscall.Signal))
			}
		case <-ctx.Done():
			signal.Stop(sigChan)
		}
	}()

	signal.Notify(sigChan, unix.SIGINT, unix.SIGTERM)
}

// GetNetworkID returns the network identifier of the created network from
// Launch. If this composition is not launched yet, it will return an empty
// string.
func (c *Composer) GetNetworkID() string {
	return c.netID
}

// Launch launches the manifest. On error containers are automatically cleaned
// up.
func (c *Composer) Launch(ctx context.Context) error {
	client, err := dc.NewClientFromEnv()
	if err != nil {
		return err
	}

	if w, ok := c.options[optionLogWriter]; ok {
		var writer io.Writer = w.(io.Writer)
		if writer == nil {
			writer = ioutil.Discard
		}
		log.SetOutput(writer)
	}

	if c.options[optionCreateNetwork] != nil {
		net, err := client.CreateNetwork(dc.CreateNetworkOptions{
			Name:    c.options[optionCreateNetwork].(string),
			Driver:  "bridge",
			Context: ctx,
		})
		if err != nil {
			return err
		}
		c.netID = net.ID
	} else if c.options[optionExistingNetwork] != nil {
		c.netID = c.options[optionExistingNetwork].(string)
	} else {
		return errors.New("compositions must have a network specified")
	}

	for _, cont := range c.manifest {
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

		exposed := map[dc.Port]struct{}{}
		bindings := map[dc.Port][]dc.PortBinding{}

		for from, to := range cont.PortForwards {
			port := dc.Port(fmt.Sprintf("%d/tcp", to))
			exposed[port] = struct{}{}
			bindings[port] = []dc.PortBinding{{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", from),
			}}
		}

		log.Printf("Creating container: [%s]", cont.Name)
		ctr, err := client.CreateContainer(dc.CreateContainerOptions{
			Name: cont.Name,
			Config: &dc.Config{
				Hostname:     cont.Name,
				Image:        cont.Image,
				Env:          cont.Env,
				Cmd:          cont.Command,
				Entrypoint:   cont.Entrypoint,
				ExposedPorts: exposed,
			},
			HostConfig: &dc.HostConfig{
				Mounts:       mounts,
				PortBindings: bindings,
			},
			NetworkingConfig: &dc.NetworkingConfig{
				EndpointsConfig: map[string]*dc.EndpointConfig{
					cont.Name: {
						NetworkID: c.netID,
						Aliases:   []string{cont.Name},
					},
				},
			},
			Context: ctx,
		})
		if err != nil {
			c.Teardown(ctx)
			return err
		}

		cont.id = ctr.ID
	}

	for _, cont := range c.manifest {
		log.Printf("Starting container: [%s]", cont.Name)
		if err := client.StartContainerWithContext(cont.id, nil, ctx); err != nil {
			c.Teardown(ctx)
			return err
		}

		if cont.BootWait != 0 {
			log.Printf("Sleeping for %v (requested by %q bootWait parameter)", cont.BootWait, cont.Name)
			time.Sleep(cont.BootWait)
		}

		if cont.AliveFunc != nil {
			log.Printf("Running aliveFunc for %v", cont.Name)
			if err := cont.AliveFunc(ctx, client, cont.id); err != nil {
				c.Teardown(ctx)
				return err
			}
			log.Printf("AliveFunc for %v completed", cont.Name)
		}

		for _, command := range cont.PostCommands {
			log.Printf("Running post-command [%s] in container: [%s]", strings.Join(command, " "), cont.Name)
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
			ins, err := client.InspectExec(exec.ID)
			if err != nil {
				c.Teardown(ctx)
				return err
			}

			if ins.ExitCode != 0 {
				c.Teardown(ctx)
				return fmt.Errorf("[%s] invalid exit code from postcommand: [%s]", cont.Name, strings.Join(command, " "))
			}
		}
	}

	return nil
}

// Teardown kills the container processes in the manifest and removes their
// containers. In the event of errors, this will continue to attempt to stop
// and remove everything before returning. It will log the error to stderr.
func (c *Composer) Teardown(ctx context.Context) error {
	if c.sigCancel != nil {
		c.sigCancel()
	}

	client, err := dc.NewClientFromEnv()
	if err != nil {
		return err
	}

	var errs bool

	for _, cont := range c.manifest {
		if cont.id != "" {
			log.Printf("Killing container: [%s]", cont.Name)
			err := client.KillContainer(dc.KillContainerOptions{
				ID:      cont.id,
				Signal:  dc.SIGKILL,
				Context: ctx,
			})
			if err != nil {
				log.Println(err)
				errs = true
			}

			log.Printf("Removing container: [%s]", cont.Name)
			if err := client.RemoveContainer(dc.RemoveContainerOptions{ID: cont.id, Force: true, Context: ctx}); err != nil {
				log.Printf("Error shutting down container: [%s] %v", cont.Name, err)
				errs = true
			}
		} else {
			log.Printf("Skipping unstarted container: [%s]", cont.Name)
		}
	}

	if c.options[optionCreateNetwork] != nil {
		if err := client.RemoveNetwork(c.netID); err != nil {
			log.Println(err)
			errs = true
		}
	}

	if errs {
		return errors.New("there were errors (see log)")
	}

	return nil
}
