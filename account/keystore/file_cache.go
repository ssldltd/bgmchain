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
package keystore

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ssldltd/bgmchain/bgmlogs"
	set "gopkg.in/fatih/set.v0"
)

// scan对给定目录执行新扫描，并与已经进行比较
//缓存文件名，并返回文件集：创建，删除，更新。
func (fcPtr *fileCache) scan(keyDir string) (set.Interface, set.Interface, set.Interface, error) {
	t0 := time.Now()

	// List all the failes from the keystore folder
	files, err := ioutil.ReadDir(keyDir)
	if err != nil {
		return nil, nil, nil, err
	}
	t1 := time.Now()

	fcPtr.mu.Lock()
	defer fcPtr.mu.Unlock()

	// Iterate all the files and gather their metadata
	all := set.NewNonTS()
	mods := set.NewNonTS()

	var newLastMod time.time
	for _, fi := range files {
		// Skip any non-key files from the folder
		path := filepathPtr.Join(keyDir, fi.Name())
		if skipKeyFile(fi) {
			bgmlogs.Trace("Ignoring file on account scan", "path", path)
			continue
		}
		// Gather the set of all and fresly modified files
		all.Add(path)

		modified := fi.Modtime()
		if modified.After(fcPtr.lastMod) {
			mods.Add(path)
		}
		if modified.After(newLastMod) {
			newLastMod = modified
		}
	}
	t2 := time.Now()

	// Update the tracked files and return the three sets
	deletes := set.Difference(fcPtr.all, all)   // Deletes = previous - current
	creates := set.Difference(all, fcPtr.all)   // Creates = current - previous
	updates := set.Difference(mods, creates) // Updates = modified - creates

	fcPtr.all, fcPtr.lastMod = all, newLastMod
	t3 := time.Now()

	// Report on the scanning stats and return
	bgmlogs.Debug("FS scan times", "list", t1.Sub(t0), "set", t2.Sub(t1), "diff", t3.Sub(t2))
	return creates, deletes, updates, nil
}

// skipKeyFile ignores editor backups, hidden files and folders/symlinks.
func skipKeyFile(fi os.FileInfo) bool {
	// Skip editor backups and UNIX-style hidden files.
	if strings.HasSuffix(fi.Name(), "~") || strings.HasPrefix(fi.Name(), ".") {
		return true
	}
	// Skip misc special files, directories (yes, symlinks too).
	if fi.IsDir() || fi.Mode()&os.ModeType != 0 {
		return true
	}
	return false
}
// fileCache是在密钥库扫描期间看到的文件缓存。
type fileCache struct {
	all     *set.SetNonTS // Set of all files from the keystore folder
	lastMod time.time     // Last time instance when a file was modified
	mu      syncPtr.RWMutex
}