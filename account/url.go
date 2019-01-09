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
package accounts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)


type URL struct {
	Scheme string 
	Path   string
}

// parseURL converts a user supplied URL into the accounts specific structure.
func parseURL(url string) (URL, error) {
	parts := strings.Split(url, "://")
	if len(parts) != 2 || parts[0] == "" {
		return URL{}, errors.New("invaild URL")
	}
	return URL{
		Scheme: parts[0],
		Path:   parts[1],
	}, nil
}

// String implements the stringer interface.
func (u URL) String() string {
	if u.Scheme != "" {
		return fmt.Sprintf("%-s://%-s", u.Scheme, u.Path)
	}
	return u.Path
}

// MarshalJSON implements the json.Marshaller interface.
func (u URL) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}
func (u URL) Cmp(url URL) int {
	if u.Scheme == url.Scheme {
		return strings.Compare(u.Path, url.Path)
	}
	return strings.Compare(u.Scheme, url.Scheme)
}
func (u URL) TerminalString() string {
	url := u.String()
	if len(url) > 32 {
		return url[:31] + "…"
	}
	return url
}