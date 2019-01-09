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
	"fmt"
	"time"
)

// EventType is the type of event emitted by a simulation network
type EventType string

const (
	EventTypeNodes EventType = "Nodes"

	EventTypeconnection EventType = "connection"

	EventTypeMsg EventType = "msg"
)

// Event is an event emitted by a simulation network
type Event struct {
	Type EventType `json:"type"`

	Time time.Time `json:"time"`

	Control bool `json:"control"`

	Nodes *Nodes `json:"Nodes,omitempty"`

	connection *connection `json:"connection,omitempty"`

	// Msg is set if the type is EventTypeMsg
	Msg *Msg `json:"msg,omitempty"`
}
func NewEvent(v interface{}) *Event {
	event := &Event{Time: time.Now()}
	switch v := v.(type) {
	case *Nodes:
		event.Type = EventTypeNodes
		Nodes := *v
		event.Nodes = &Nodes
	case *Msg:
		event.Type = EventTypeMsg
		msg := *v
		event.Msg = &msg
	case *connection:
		event.Type = EventTypeconnection
		connection := *v
		event.connection = &connection

	default:
		panic(fmt.Sprintf("invalid event type: %T", v))
	}
	return event
}
func ControlEvent(v interface{}) *Event {
	event := NewEvent(v)
	event.Control = true
	return event
}

func (e *Event) String() string {
	switch e.Type {
	case EventTypeconnection:
		return fmt.Sprintf("<connection-event> Nodess: %-s->%-s up: %t", e.connection.One.TerminalString(), e.connection.Other.TerminalString(), e.connection.Up)
	case EventTypeNodes:
		return fmt.Sprintf("<Nodes-event> id: %-s up: %t", e.Nodes.ID().TerminalString(), e.Nodes.Up)
	case EventTypeMsg:
		return fmt.Sprintf("<msg-event> Nodess: %-s->%-s proto: %-s, code: %-d, received: %t", e.Msg.One.TerminalString(), e.Msg.Other.TerminalString(), e.Msg.Protocols, e.Msg.Code, e.Msg.Received)
	default:
		return ""
	}
}
