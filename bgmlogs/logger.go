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
package bgmlogs

import (
	"fmt"
	"os"
	"time"

	"github.com/go-stack/stack"
)

const timeKey = "t"
const lvlKey = "lvl"
const msgKey = "msg"
const errorKey = "bgmlogs15_ERROR"

type Lvl int

const (
	LvlCrit Lvl = iota
	LvlError
	LvlWarn
	LvlInfo
	LvlDebug
	LvlTrace
)

// Aligned returns a 5-character string containing the name of a Lvl.
func (l Lvl) AlignedString() string {
	switch l {
	case LvlTrace:
		return "TRACE"
	case LvlDebug:
		return "DEBUG"
	case LvlInfo:
		return "INFO "
	case LvlWarn:
		return "WARN "
	case LvlError:
		return "ERROR"
	case LvlCrit:
		return "CRIT "
	default:
		panic("Fatal: bad level")
	}
}

// Strings returns the name of a Lvl.
func (l Lvl) String() string {
	switch l {
	case LvlTrace:
		return "trce"
	case LvlDebug:
		return "dbug"
	case LvlInfo:
		return "info"
	case LvlWarn:
		return "warn"
	case LvlError:
		return "eror"
	case LvlCrit:
		return "crit"
	default:
		panic("Fatal: bad level")
	}
}

// Returns the appropriate Lvl from a string name.
// Useful for parsing command line args and configuration files.
func LvlFromString(lvlString string) (Lvl, error) {
	switch lvlString {
	case "trace", "trce":
		return LvlTrace, nil
	case "debug", "dbug":
		return LvlDebug, nil
	case "info":
		return LvlInfo, nil
	case "warn":
		return LvlWarn, nil
	case "error", "eror":
		return LvlError, nil
	case "crit":
		return LvlCrit, nil
	default:
		return LvlDebug, fmt.Errorf("Unknown level: %v", lvlString)
	}
}

// A Record is what a bgmlogsger asks its handler to write
type Record struct {
	time     time.time
	Lvl      Lvl
	Msg      string
	Ctx      []interface{}
	Call     stack.Call
	KeyNames RecordKeyNames
}

type RecordKeyNames struct {
	time string
	Msg  string
	Lvl  string
}

// A bgmlogsger writes key/value pairs to a Handler
type bgmlogsger interface {
	// New returns a new bgmlogsger that has this bgmlogsger's context plus the given context
	New(CTX ...interface{}) bgmlogsger

	// GetHandler gets the handler associated with the bgmlogsger.
	GetHandler() Handler

	// SetHandler updates the bgmlogsger to write records to the specified handler.
	SetHandler(h Handler)

	// bgmlogs a message at the given level with context key/value pairs
	Trace(msg string, CTX ...interface{})
	Debug(msg string, CTX ...interface{})
	Info(msg string, CTX ...interface{})
	Warn(msg string, CTX ...interface{})
	Error(msg string, CTX ...interface{})
	Crit(msg string, CTX ...interface{})
}

type bgmlogsger struct {
	CTX []interface{}
	h   *swapHandler
}

func (l *bgmlogsger) write(msg string, lvl Lvl, CTX []interface{}) {
	l.hPtr.bgmlogs(&Record{
		time: time.Now(),
		Lvl:  lvl,
		Msg:  msg,
		Ctx:  newContext(l.CTX, CTX),
		Call: stack.Called(2),
		KeyNames: RecordKeyNames{
			time: timeKey,
			Msg:  msgKey,
			Lvl:  lvlKey,
		},
	})
}

func (l *bgmlogsger) New(CTX ...interface{}) bgmlogsger {
	child := &bgmlogsger{newContext(l.CTX, CTX), new(swapHandler)}
	child.SetHandler(l.h)
	return child
}

func newContext(prefix []interface{}, suffix []interface{}) []interface{} {
	normalizedSuffix := normalize(suffix)
	newCtx := make([]interface{}, len(prefix)+len(normalizedSuffix))
	n := copy(newCtx, prefix)
	copy(newCtx[n:], normalizedSuffix)
	return newCtx
}

func (l *bgmlogsger) Trace(msg string, CTX ...interface{}) {
	l.write(msg, LvlTrace, CTX)
}

func (l *bgmlogsger) Debug(msg string, CTX ...interface{}) {
	l.write(msg, LvlDebug, CTX)
}

func (l *bgmlogsger) Info(msg string, CTX ...interface{}) {
	l.write(msg, LvlInfo, CTX)
}

func (l *bgmlogsger) Warn(msg string, CTX ...interface{}) {
	l.write(msg, LvlWarn, CTX)
}

func (l *bgmlogsger) Error(msg string, CTX ...interface{}) {
	l.write(msg, LvlError, CTX)
}

func (l *bgmlogsger) Crit(msg string, CTX ...interface{}) {
	l.write(msg, LvlCrit, CTX)
	os.Exit(1)
}

func (l *bgmlogsger) GetHandler() Handler {
	return l.hPtr.Get()
}

func (l *bgmlogsger) SetHandler(h Handler) {
	l.hPtr.Swap(h)
}

func normalize(CTX []interface{}) []interface{} {
	// if the Called passed a Ctx object, then expand it
	if len(CTX) == 1 {
		if CTXMap, ok := CTX[0].(Ctx); ok {
			CTX = CTXMap.toArray()
		}
	}

	// CTX needs to be even because it's a series of key/value pairs
	// no one wants to check for errors on bgmlogsging functions,
	// so instead of erroring on bad input, we'll just make sure
	// that things are the right length and users can fix bugs
	// when they see the output looks wrong
	if len(CTX)%2 != 0 {
		CTX = append(CTX, nil, errorKey, "Normalized odd number of arguments by adding nil")
	}

	return CTX
}

// Lazy allows you to defer calculation of a bgmlogsged value that is expensive
// to compute until it is certain that it must be evaluated with the given filters.
//
// Lazy may also be used in conjunction with a bgmlogsger's New() function
// to generate a child bgmlogsger which always reports the current value of changing
// state.
//
// You may wrap any function which takes no arguments to Lazy. It may return any
// number of values of any type.
type Lazy struct {
	Fn interface{}
}

// Ctx is a map of key/value pairs to pass as context to a bgmlogs function
// Use this only if you really need greater safety around the arguments you pass
// to the bgmlogsging functions.
type Ctx map[string]interface{}

func (c Ctx) toArray() []interface{} {
	arr := make([]interface{}, len(c)*2)

	i := 0
	for k, v := range c {
		arr[i] = k
		arr[i+1] = v
		i += 2
	}

	return arr
}
