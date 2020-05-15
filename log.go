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
	"fmt"
	"github.com/kataras/golog"
	"os"
	"path/filepath"
)

// Loggers

// AppLogger : logger
func AppLogger() (*golog.Logger, *os.File) {

	appLogger := golog.New()

	applogfile := appLogFile()

	if debugmode {
		appLogger.SetLevel("debug")
		appLogger.AddOutput(applogfile)
	} else {
		appLogger.SetLevel("warn")
		appLogger.SetOutput(applogfile)
	}

	return appLogger, applogfile

}

// GetLogger : logger
func GetLogger() (*golog.Logger, *os.File) {

	getLogger := golog.New()

	getlogfile := getLogFile()

	if debugmode {
		getLogger.SetLevel("debug")
		getLogger.AddOutput(getlogfile)
	} else {
		getLogger.SetLevel("warn")
		getLogger.SetOutput(getlogfile)
	}

	return getLogger, getlogfile

}

// PostLogger : logger
func PostLogger() (*golog.Logger, *os.File) {

	postLogger := golog.New()

	postlogfile := postLogFile()

	if debugmode {
		postLogger.SetLevel("debug")
		postLogger.AddOutput(postlogfile)
	} else {
		postLogger.SetLevel("warn")
		postLogger.SetOutput(postlogfile)
	}

	return postLogger, postlogfile

}

// Log Paths

func todayAppFilename() string {
	logfile := filepath.Clean(logdir + "/app.log")
	return logfile
}

func todayGetFilename() string {
	logfile := filepath.Clean(logdir + "/get.log")
	return logfile
}

func todayPostFilename() string {
	logfile := filepath.Clean(logdir + "/post.log")
	return logfile
}

// Log Files

func appLogFile() *os.File {

	filename := todayAppFilename()
	applogfile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, logmode)
	if err != nil {
		fmt.Printf("Can`t open/create 'app' log file error | File [%s] | %v", filename, err)
		os.Exit(1)
	}

	err = os.Chmod(filename, logmode)
	if err != nil {
		fmt.Printf("Can`t chmod log file error | File [%s] | %v", filename, err)
		os.Exit(1)
	}

	return applogfile
}

func getLogFile() *os.File {
	filename := todayGetFilename()
	getlogfile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, logmode)
	if err != nil {
		fmt.Printf("Can`t open/create 'get' log file error | File [%s] | %v", filename, err)
		os.Exit(1)
	}

	err = os.Chmod(filename, logmode)
	if err != nil {
		fmt.Printf("Can`t chmod log file error | File [%s] | %v", filename, err)
		os.Exit(1)
	}

	return getlogfile
}

func postLogFile() *os.File {
	filename := todayPostFilename()
	postlogfile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, logmode)
	if err != nil {
		fmt.Printf("Can`t open/create 'post' log file error | File [%s] | %v", filename, err)
		os.Exit(1)
	}

	err = os.Chmod(filename, logmode)
	if err != nil {
		fmt.Printf("Can`t chmod log file error | File [%s] | %v", filename, err)
		os.Exit(1)
	}

	return postlogfile
}
