/*

Copyright © 2020 Andrey Kuvshinov. Contacts: <syslinux@protonmail.com>
Copyright © 2020 Eltaline OU. Contacts: <eltaline.ou@gmail.com>
All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

The cTRL project contains unmodified/modified libraries imports too with
separate copyright notices and license terms. Your use of the source code
this libraries is subject to the terms and conditions of licenses these libraries.

*/

package main

import (
	"os"
	"path/filepath"
)

// File Helpers

// FileExists : check existence of requested file
func FileExists(filename string) bool {

	if fi, err := os.Stat(filename); err == nil {

		if fi.Mode().IsRegular() {
			return true
		}

	}

	return false

}

// FileOrLinkExists : check existence of requested file or symlink
func FileOrLinkExists(filename string) bool {

	if fi, err := os.Stat(filename); err == nil {

		if fi.Mode().IsRegular() {
			return true
		}

	}

	if _, err := filepath.EvalSymlinks(filename); err == nil {
		return true
	}

	return false

}

// DirExists : check existence of requested directory
func DirExists(filename string) bool {

	if fi, err := os.Stat(filename); err == nil {

		if fi.Mode().IsDir() {
			return true
		}

	}

	return false

}

// RemoveSegment : remove NutsDB segment file if empty
func RemoveSegment(filename string) error {

	err := os.Remove(filename)
	if err != nil {
		return err
	}

	return nil

}
