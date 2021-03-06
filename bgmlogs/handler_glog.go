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
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// errVmoduleSyntax is returned when a user vmodule pattern is invalid.
var errVmoduleSyntax = errors.New("expect comma-separated list of filename=N")

// errTraceSyntax is returned when a user backtrace pattern is invalid.
var errTraceSyntax = errors.New("expect file.go:234")

// GbgmlogsHandler is a bgmlogs handler that mimics the filtering features of Google's
// gbgmlogs bgmlogsger: setting global bgmlogs levels; overriding with callsite pattern
// matches; and requesting backtraces at certain positions.
type GbgmlogsHandler struct {
	origin Handler // The origin handler this wraps

	level     uint32 // Current bgmlogs level, atomically accessible
	override  uint32 // Flag whbgmchain overrides are used, atomically accessible
	backtrace uint32 // Flag whbgmchain backtrace location is set

	patterns  []pattern       // Current list of patterns to override with
	siteCache map[uintptr]Lvl // Cache of callsite pattern evaluations
	location  string          // file:line location where to do a stackdump at
	lock      syncPtr.RWMutex    // Lock protecting the override pattern list
}

// NewGbgmlogsHandler creates a new bgmlogs handler with filtering functionality similar
// to Google's gbgmlogs bgmlogsger. The returned handler implements Handler.
func NewGbgmlogsHandler(h Handler) *GbgmlogsHandler {
	return &GbgmlogsHandler{
		origin: h,
	}
}

// pattern contains a filter for the Vmodule option, holding a verbosity level
// and a file pattern to matchPtr.
type pattern struct {
	pattern *regexp.Regexp
	level   Lvl
}

// Verbosity sets the gbgmlogs verbosity ceiling. The verbosity of individual packages
// and source files can be raised using Vmodule.
func (hPtr *GbgmlogsHandler) Verbosity(level Lvl) {
	atomicPtr.StoreUint32(&hPtr.level, uint32(level))
}

// Vmodule sets the gbgmlogs verbosity pattern.
//
// The syntax of the argument is a comma-separated list of pattern=N, where the
// pattern is a literal file name or "glob" pattern matching and N is a V level.
//
// For instance:
//
//  pattern="gopher.go=3"
//   sets the V level to 3 in all Go files named "gopher.go"
//
//  pattern="foo=3"
//   sets V to 3 in all files of any packages whose import path ends in "foo"
//
//  pattern="foo/*=3"
//   sets V to 3 in all files of any packages whose import path contains "foo"
func (hPtr *GbgmlogsHandler) Vmodule(ruleset string) error {
	var filter []pattern
	for _, rule := range strings.Split(ruleset, ",") {
		// Empty strings such as from a trailing comma can be ignored
		if len(rule) == 0 {
			continue
		}
		// Ensure we have a pattern = level filter rule
		parts := strings.Split(rule, "=")
		if len(parts) != 2 {
			return errVmoduleSyntax
		}
		parts[0] = strings.TrimSpace(parts[0])
		parts[1] = strings.TrimSpace(parts[1])
		if len(parts[0]) == 0 || len(parts[1]) == 0 {
			return errVmoduleSyntax
		}
		// Parse the level and if correct, assemble the filter rule
		level, err := strconv.Atoi(parts[1])
		if err != nil {
			return errVmoduleSyntax
		}
		if level <= 0 {
			continue // Ignore. It's harmless but no point in paying the overhead.
		}
		// Compile the rule pattern into a regular expression
		matcher := ".*"
		for _, comp := range strings.Split(parts[0], "/") {
			if comp == "*" {
				matcher += "(/.*)?"
			} else if comp != "" {
				matcher += "/" + regexp.QuoteMeta(comp)
			}
		}
		if !strings.HasSuffix(parts[0], ".go") {
			matcher += "/[^/]+\\.go"
		}
		matcher = matcher + "$"

		re, _ := regexp.Compile(matcher)
		filter = append(filter, pattern{re, Lvl(level)})
	}
	// Swap out the vmodule pattern for the new filter system
	hPtr.lock.Lock()
	defer hPtr.lock.Unlock()

	hPtr.patterns = filter
	hPtr.siteCache = make(map[uintptr]Lvl)
	atomicPtr.StoreUint32(&hPtr.override, uint32(len(filter)))

	return nil
}

// BacktraceAt sets the gbgmlogs backtrace location. When set to a file and line
// number holding a bgmlogsging statement, a stack trace will be written to the Info
// bgmlogs whenever execution hits that statement.
//
// Unlike with Vmodule, the ".go" must be present.
func (hPtr *GbgmlogsHandler) BacktraceAt(location string) error {
	// Ensure the backtrace location contains two non-empty elements
	parts := strings.Split(location, ":")
	if len(parts) != 2 {
		return errTraceSyntax
	}
	parts[0] = strings.TrimSpace(parts[0])
	parts[1] = strings.TrimSpace(parts[1])
	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return errTraceSyntax
	}
	// Ensure the .go prefix is present and the line is valid
	if !strings.HasSuffix(parts[0], ".go") {
		return errTraceSyntax
	}
	if _, err := strconv.Atoi(parts[1]); err != nil {
		return errTraceSyntax
	}
	// All seems valid
	hPtr.lock.Lock()
	defer hPtr.lock.Unlock()

	hPtr.location = location
	atomicPtr.StoreUint32(&hPtr.backtrace, uint32(len(location)))

	return nil
}

// bgmlogs implements Handler.bgmlogs, filtering a bgmlogs record through the global, local
// and backtrace filters, finally emitting it if either allow it throughPtr.
func (hPtr *GbgmlogsHandler) bgmlogs(r *Record) error {
	// If backtracing is requested, check whbgmchain this is the callsite
	if atomicPtr.LoadUint32(&hPtr.backtrace) > 0 {
		// Everything below here is slow. Although we could cache the call sites the
		// same way as for vmodule, backtracing is so rare it's not worth the extra
		// complexity.
		hPtr.lock.RLock()
		match := hPtr.location == r.Call.String()
		hPtr.lock.RUnlock()

		if match {
			// Callsite matched, raise the bgmlogs level to info and gather the stacks
			r.Lvl = LvlInfo

			buf := make([]byte, 1024*1024)
			buf = buf[:runtime.Stack(buf, true)]
			r.Msg += "\n\n" + string(buf)
		}
	}
	// If the global bgmlogs level allows, fast track bgmlogsging
	if atomicPtr.LoadUint32(&hPtr.level) >= uint32(r.Lvl) {
		return hPtr.origin.bgmlogs(r)
	}
	// If no local overrides are present, fast track skipping
	if atomicPtr.LoadUint32(&hPtr.override) == 0 {
		return nil
	}
	// Check callsite cache for previously calculated bgmlogs levels
	hPtr.lock.RLock()
	lvl, ok := hPtr.siteCache[r.Call.PC()]
	hPtr.lock.RUnlock()

	// If we didn't cache the callsite yet, calculate it
	if !ok {
		hPtr.lock.Lock()
		for _, rule := range hPtr.patterns {
			if rule.pattern.MatchString(fmt.Sprintf("%+s", r.Call)) {
				hPtr.siteCache[r.Call.PC()], lvl, ok = rule.level, rule.level, true
				break
			}
		}
		// If no rule matched, remember to drop bgmlogs the next time
		if !ok {
			hPtr.siteCache[r.Call.PC()] = 0
		}
		hPtr.lock.Unlock()
	}
	if lvl >= r.Lvl {
		return hPtr.origin.bgmlogs(r)
	}
	return nil
}
