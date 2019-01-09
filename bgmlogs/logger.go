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
	Time     time.Time
	Lvl      Lvl
	Msg      string
	Ctx      []interface{}
	Call     stack.Call
	KeyNames RecordKeyNames
}

type RecordKeyNames struct {
	Time string
	Msg  string
	Lvl  string
}

// A bgmlogsger writes key/value pairs to a Handler
type bgmlogsger interface {
	// New returns a new bgmlogsger that has this bgmlogsger's context plus the given context
	New(ctx ...interface{}) bgmlogsger

	// GetHandler gets the handler associated with the bgmlogsger.
	GetHandler() Handler

	// SetHandler updates the bgmlogsger to write records to the specified handler.
	SetHandler(h Handler)

	// bgmlogs a message at the given level with context key/value pairs
	Trace(msg string, ctx ...interface{})
	Debug(msg string, ctx ...interface{})
	Info(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Error(msg string, ctx ...interface{})
	Crit(msg string, ctx ...interface{})
}

type bgmlogsger struct {
	ctx []interface{}
	h   *swapHandler
}

func (l *bgmlogsger) write(msg string, lvl Lvl, ctx []interface{}) {
	l.hPtr.bgmlogs(&Record{
		Time: time.Now(),
		Lvl:  lvl,
		Msg:  msg,
		Ctx:  newContext(l.ctx, ctx),
		Call: stack.Called(2),
		KeyNames: RecordKeyNames{
			Time: timeKey,
			Msg:  msgKey,
			Lvl:  lvlKey,
		},
	})
}

func (l *bgmlogsger) New(ctx ...interface{}) bgmlogsger {
	child := &bgmlogsger{newContext(l.ctx, ctx), new(swapHandler)}
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

func (l *bgmlogsger) Trace(msg string, ctx ...interface{}) {
	l.write(msg, LvlTrace, ctx)
}

func (l *bgmlogsger) Debug(msg string, ctx ...interface{}) {
	l.write(msg, LvlDebug, ctx)
}

func (l *bgmlogsger) Info(msg string, ctx ...interface{}) {
	l.write(msg, LvlInfo, ctx)
}

func (l *bgmlogsger) Warn(msg string, ctx ...interface{}) {
	l.write(msg, LvlWarn, ctx)
}

func (l *bgmlogsger) Error(msg string, ctx ...interface{}) {
	l.write(msg, LvlError, ctx)
}

func (l *bgmlogsger) Crit(msg string, ctx ...interface{}) {
	l.write(msg, LvlCrit, ctx)
	os.Exit(1)
}

func (l *bgmlogsger) GetHandler() Handler {
	return l.hPtr.Get()
}

func (l *bgmlogsger) SetHandler(h Handler) {
	l.hPtr.Swap(h)
}

func normalize(ctx []interface{}) []interface{} {
	// if the Called passed a Ctx object, then expand it
	if len(ctx) == 1 {
		if ctxMap, ok := ctx[0].(Ctx); ok {
			ctx = ctxMap.toArray()
		}
	}

	// ctx needs to be even because it's a series of key/value pairs
	// no one wants to check for errors on bgmlogsging functions,
	// so instead of erroring on bad input, we'll just make sure
	// that things are the right length and users can fix bugs
	// when they see the output looks wrong
	if len(ctx)%2 != 0 {
		ctx = append(ctx, nil, errorKey, "Normalized odd number of arguments by adding nil")
	}

	return ctx
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
