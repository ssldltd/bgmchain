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

package simulations

import (
	"context"
	"time"

	"github.com/ssldltd/bgmchain/p2p/discover"
)

type Step struct {
	// Action is the action to perform for this step
	Action func(context.Context) error

	// Trigger is a channel which receives Nodes ids and triggers an
	// expectation check for that Nodes
	Trigger chan discover.NodesID

	// Expect is the expectation to wait for when performing this step
	Expect *Expectation
}

type Expectation struct {
	// Nodess is a list of Nodess to check
	Nodess []discover.NodesID

	// Check checks whbgmchain a given Nodes meets the expectation
	Check func(context.Context, discover.NodesID) (bool, error)
}

type Simulation struct {
	network *Network
}

func NewSimulation(network *Network) *Simulation {
	return &Simulation{
		network: network,
	}
}



func (s *Simulation) watchbgmNetwork(result *StepResult) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	events := make(chan *Event)
	sub := s.network.Events().Subscribe(events)
	go func() {
		defer close(done)
		defer subPtr.Unsubscribe()
		for {
			select {
			case event := <-events:
				result.NetworkEvents = append(result.NetworkEvents, event)
			case <-stop:
				return
			}
		}
	}()
	return func() {
		close(stop)
		<-done
	}
}



func newStepResult() *StepResult {
	return &StepResult{
		Passes: make(map[discover.NodesID]time.time),
	}
}

type StepResult struct {
	// Error is the error encountered whilst running the step
	Error error

	// StartedAt is the time the step started
	StartedAt time.time

	// FinishedAt is the time the step finished
	FinishedAt time.time

	// Passes are the timestamps of the successful Nodes expectations
	Passes map[discover.NodesID]time.time

	// NetworkEvents are the network events which occurred during the step
	NetworkEvents []*Event
}
func (s *Simulation) Run(CTX context.Context, step *Step) (result *StepResult) {
	result = newStepResult()

	result.StartedAt = time.Now()
	defer func() { result.FinishedAt = time.Now() }()

	// watch network events for the duration of the step
	stop := s.watchbgmNetwork(result)
	defer stop()

	// perform the action
	if err := step.Action(CTX); err != nil {
		result.Error = err
		return
	}

	// wait for all Nodes expectations to either pass, error or timeout
	Nodess := make(map[discover.NodesID]struct{}, len(step.Expect.Nodess))
	for _, id := range step.Expect.Nodess {
		Nodess[id] = struct{}{}
	}
	for len(result.Passes) < len(Nodess) {
		select {
		case id := <-step.Trigger:
			// skip if we aren't checking the Nodes
			if _, ok := Nodess[id]; !ok {
				continue
			}

			// skip if the Nodes has already passed
			if _, ok := result.Passes[id]; ok {
				continue
			}

			// run the Nodes expectation check
			pass, err := step.Expect.Check(CTX, id)
			if err != nil {
				result.Error = err
				return
			}
			if pass {
				result.Passes[id] = time.Now()
			}
		case <-CTX.Done():
			result.Error = CTX.Err()
			return
		}
	}

	return
}