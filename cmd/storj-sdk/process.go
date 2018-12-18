// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"

	"storj.io/storj/internal/processgroup"
	"storj.io/storj/pkg/utils"
)

// Processes contains list of processes
type Processes struct {
	List []*Process
}

// NewProcesses creates a process-set with satellites and storage nodes
func NewProcesses(dir string, satelliteCount, storageNodeCount int) (*Processes, error) {
	processes := &Processes{}

	const (
		host            = "127.0.0.1"
		gatewayPort     = 9000
		satellitePort   = 10000
		storageNodePort = 11000
	)

	defaultSatellite := net.JoinHostPort(host, strconv.Itoa(satellitePort+0))

	arguments := func(name, command string, port int, rest ...string) []string {
		return append([]string{
			"--log.level", "debug",
			"--log.prefix", name,
			"--config-dir", ".",
			command,
			"--identity.server.address", net.JoinHostPort(host, strconv.Itoa(port)),
		}, rest...)
	}

	for i := 0; i < satelliteCount; i++ {
		name := fmt.Sprintf("satellite/%d", i)

		dir := filepath.Join(dir, "satellite", fmt.Sprint(i))
		if err := os.MkdirAll(dir, 0644); err != nil {
			return nil, err
		}

		process, err := NewProcess(name, "satellite", dir)
		if err != nil {
			return nil, utils.CombineErrors(err, processes.Close())
		}
		processes.List = append(processes.List, process)

		process.Arguments["setup"] = arguments(name, "setup", satellitePort+i)
		process.Arguments["run"] = arguments(name,
			"run", satellitePort+i,
			"--kademlia.bootstrap-addr", defaultSatellite,
		)
	}

	gatewayArguments := func(name, command string, index int, rest ...string) []string {
		return append([]string{
			"--log.level", "debug",
			"--log.prefix", name,
			"--config-dir", ".",
			command,
			// "--satellite-addr", net.JoinHostPort(host, strconv.Itoa(satellitePort+index)),
			"--identity.server.address", net.JoinHostPort(host, strconv.Itoa(gatewayPort+index)),
		}, rest...)
	}

	for i := 0; i < satelliteCount; i++ {
		name := fmt.Sprintf("gateway/%d", i)

		dir := filepath.Join(dir, "gateway", fmt.Sprint(i))
		if err := os.MkdirAll(dir, 0644); err != nil {
			return nil, err
		}

		process, err := NewProcess(name, "gateway", dir)
		if err != nil {
			return nil, utils.CombineErrors(err, processes.Close())
		}
		processes.List = append(processes.List, process)

		process.Arguments["setup"] = gatewayArguments(name, "setup", i)
		process.Arguments["run"] = gatewayArguments(name, "run", i)
	}

	for i := 0; i < storageNodeCount; i++ {
		name := fmt.Sprintf("storage/%d", i)

		dir := filepath.Join(dir, "storagenode", fmt.Sprint(i))
		if err := os.MkdirAll(dir, 0644); err != nil {
			return nil, err
		}

		process, err := NewProcess(name, "storagenode", dir)
		if err != nil {
			return nil, utils.CombineErrors(err, processes.Close())
		}
		processes.List = append(processes.List, process)

		process.Arguments["setup"] = arguments(name, "setup", storageNodePort+i,
			"--piecestore.agreementsender.overlay-addr", defaultSatellite,
		)
		process.Arguments["run"] = arguments(name, "run", storageNodePort+i,
			"--piecestore.agreementsender.overlay-addr", defaultSatellite,
			"--kademlia.bootstrap-addr", defaultSatellite,
		)
	}

	return processes, nil
}

// Exec executes a command on all processes
func (processes *Processes) Exec(ctx context.Context, command string) error {
	var group errgroup.Group
	for _, p := range processes.List {
		process := p
		group.Go(func() error {
			return process.Exec(ctx, command)
		})
	}

	return group.Wait()
}

// Close closes all the processes and their resources
func (processes *Processes) Close() error {
	var errs []error
	for _, process := range processes.List {
		err := process.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return utils.CombineErrors(errs...)
}

// Process is a type for monitoring the process
type Process struct {
	Name       string
	Directory  string
	Executable string

	Arguments map[string][]string

	Stdout io.Writer
	Stderr io.Writer

	stdout *os.File
	stderr *os.File
}

// NewProcess creates a process which can be run in the specified directory
func NewProcess(name, executable, directory string) (*Process, error) {
	stdout, err1 := os.OpenFile(filepath.Join(directory, "stderr.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	stderr, err2 := os.OpenFile(filepath.Join(directory, "stdout.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	return &Process{
		Name:       name,
		Directory:  directory,
		Executable: executable,

		Arguments: map[string][]string{},

		Stdout: io.MultiWriter(os.Stdout, stdout),
		Stderr: io.MultiWriter(os.Stderr, stderr),

		stdout: stdout,
		stderr: stderr,
	}, utils.CombineErrors(err1, err2)
}

// Exec runs the process using the arguments for a given command
func (process *Process) Exec(ctx context.Context, command string) error {
	cmd := exec.CommandContext(ctx, process.Executable, process.Arguments[command]...)
	cmd.Dir = process.Directory
	cmd.Stdout, cmd.Stderr = process.Stdout, process.Stderr

	processgroup.Setup(cmd)

	if printCommands {
		fmt.Println("exec: ", strings.Join(cmd.Args, " "))
	}
	return cmd.Run()
}

// Close closes process resources
func (process *Process) Close() error {
	return utils.CombineErrors(
		process.stdout.Close(),
		process.stderr.Close(),
	)
}
