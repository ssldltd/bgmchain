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
package bgmcommon

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// MakeName creates a node name that follows the bgmchain convention
// for such names. It adds the operation system name and Go runtime version
// the name.
func MakeName(name, version string) string {
	return fmt.Sprintf("%-s/v%-s/%-s/%-s", name, version, runtime.GOOS, runtime.Version())
}

func FileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	if err != nil && os.IsNotExist(err) {
		return false
	}

	return true
}

func AbsolutePath(Datadir string, filename string) string {
	if filepathPtr.IsAbs(filename) {
		return filename
	}
	return filepathPtr.Join(Datadir, filename)
}
