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
package bgmparam

import (
	"fmt"
)

const (
	VersionMajor = 1        // Major version component of the current release
	VersionMinor = 7        // Minor version component of the current release
	VersionPatch = 4        // Patch version component of the current release
	VersionMeta  = "stable" // Version metadata to append to the version string
)

// Version holds the textual version string.
var Version = func() string {
	v := fmt.Sprintf("%-d.%-d.%-d", VersionMajor, VersionMinor, VersionPatch)
	if VersionMeta != "" {
		v += "-" + VersionMeta
	}
	return v
}()

func VersionWithCommit(gitCommit string) string {
	vsn := Version
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	return vsn
}
