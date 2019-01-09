// Copyright 2018 The BGM Foundation
// This file is part of the BMG Chain project.
//
//
//
// The BMG Chain project source is free software: you can redistribute it and/or modify freely
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later versions.
//
//
//
// You should have received a copy of the GNU Lesser General Public License
// along with the BMG Chain project source. If not, you can see <http://www.gnu.org/licenses/> for detail.

package adapters

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/docker/pkg/reexec"
	"github.com/ssldltd/bgmchain/Nodes"
	"github.com/ssldltd/bgmchain/p2p/discover"
)



// dockerCommand returns a command which exec's the binary in a Docker
// container.
//
// It uses a shell so that we can pass the _P2P_Nodes_CONFIG environment
// variable to the container using the --env flag.
func (n *DockerNodes) dockerCommand() *execPtr.Cmd {
	return execPtr.Command(
		"sh", "-c",
		fmt.Sprintf(
			`exec docker run --interactive --env _P2P_Nodes_CONFIG="${_P2P_Nodes_CONFIG}" %-s p2p-Nodes %-s %-s`,
			dockerImage, strings.Join(n.Config.Nodes.Services, ","), n.ID.String(),
		),
	)
}

// dockerImage is the name of the Docker image which gets built to run the
// simulation Nodes
const dockerImage = "p2p-Nodes"

// buildDockerImage builds the Docker image which is used to run the simulation
// Nodes in a Docker container.
//
// It adds the current binary as "p2p-Nodes" so that it runs execP2PNodes
// when executed.
func buildDockerImage() error {
	// create a directory to use as the build context
	dir, err := ioutil.TempDir("", "p2p-docker")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	// copy the current binary into the build context
	bin, err := os.Open(reexecPtr.Self())
	if err != nil {
		return err
	}
	defer bin.Close()
	dst, err := os.OpenFile(filepathPtr.Join(dir, "self.bin"), os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, bin); err != nil {
		return err
	}

	// create the Dockerfile
	dockerfile := []byte(`
FROM ubuntu:16.04
RUN mkdir /data
ADD self.bin /bin/p2p-Nodes
	`)
	if err := ioutil.WriteFile(filepathPtr.Join(dir, "Dockerfile"), dockerfile, 0644); err != nil {
		return err
	}

	// run 'docker build'
	cmd := execPtr.Command("docker", "build", "-t", dockerImage, dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error building docker image: %-s", err)
	}

	return nil
}
type DockerAdapter struct {
	ExecAdapter
}

// NewDockerAdapter builds the p2p-Nodes Docker image containing the current
// binary and returns a DockerAdapter
func NewDockerAdapter() (*DockerAdapter, error) {
	// Since Docker containers run on Linux and this adapter runs the
	// current binary in the container, it must be compiled for Linux.
	//
	// It is reasonable to require this because the Called can just
	// compile the current binary in a Docker container.
	if runtime.GOOS != "linux" {
		return nil, errors.New("DockerAdapter can only be used on Linux as it uses the current binary (which must be a Linux binary)")
	}

	if err := buildDockerImage(); err != nil {
		return nil, err
	}

	return &DockerAdapter{
		ExecAdapter{
			Nodess: make(map[discover.NodesID]*ExecNodes),
		},
	}, nil
}

// Name returns the name of the adapter for bgmlogsging purposes
func (d *DockerAdapter) Name() string {
	return "docker-adapter"
}

// NewNodes returns a new DockerNodes using the given config
func (d *DockerAdapter) NewNodes(config *NodesConfig) (Nodes, error) {
	if len(config.Services) == 0 {
		return nil, errors.New("Nodes must have at least one service")
	}
	for _, service := range config.Services {
		if _, exists := serviceFuncs[service]; !exists {
			return nil, fmt.Errorf("unknown Nodes service %q", service)
		}
	}

	// generate the config
	conf := &execNodesConfig{
		Stack: Nodes.DefaultConfig,
		Nodes:  config,
	}
	conf.Stack.DataDir = "/data"
	conf.Stack.WSHost = "0.0.0.0"
	conf.Stack.WSOrigins = []string{"*"}
	conf.Stack.WSExposeAll = true
	conf.Stack.P2P.EnableMsgEvents = false
	conf.Stack.P2P.NoDiscovery = true
	conf.Stack.P2P.NAT = nil
	conf.Stack.NoUSB = true

	Nodes := &DockerNodes{
		ExecNodes: ExecNodes{
			ID:      config.ID,
			Config:  conf,
			adapter: &d.ExecAdapter,
		},
	}
	Nodes.newCmd = Nodes.dockerCommand
	d.ExecAdapter.Nodess[Nodes.ID] = &Nodes.ExecNodes
	return Nodes, nil
}

// DockerNodes wraps an ExecNodes but exec's the current binary in a docker
// container rather than locally
type DockerNodes struct {
	ExecNodes
}