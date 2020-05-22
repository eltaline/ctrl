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
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/eltaline/gron"
	"github.com/eltaline/mmutex"
	"github.com/eltaline/nutsdb"
	"github.com/eltaline/toml"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/recover"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Global Configuration

// Config : Global configuration type
type Config struct {
	Global global
	Server map[string]server
}

type global struct {
	BINDADDR          string
	BINDADDRSSL       string
	ONLYSSL           bool
	READTIMEOUT       int
	READHEADERTIMEOUT int
	WRITETIMEOUT      int
	IDLETIMEOUT       int
	KEEPALIVE         bool
	REALHEADER        string
	CHARSET           string
	DEBUGMODE         bool
	GCPERCENT         int
	DBDIR             string
	SCHTIME           int
	LOGDIR            string
	LOGMODE           uint32
	PIDFILE           string
}

type server struct {
	HOST       string
	SSLCRT     string
	SSLKEY     string
	USSALLOW   string
	IPSALLOW   string
	SHELL      string
	RTHREADS   uint32
	VTHREADS   uint32
	VTIMEOUT   uint32
	VTTLTIME   uint32
	VINTERVAL  uint32
	VREPEATERR []string
	VREPEATCNT uint32
}

// UssAllow : type for key and slice pairs of a virtual host and user/hash allowable pairs
type UssAllow struct {
	Vhost string
	PAIR  []strPAIR
}

type strPAIR struct {
	User string
	Hash string
}

// IpsAllow : type for key and slice pairs of a virtual host and CIDR allowable networks
type IpsAllow struct {
	Vhost string
	CIDR  []strCIDR
}

type strCIDR struct {
	Addr string
}

// GetTask : type task for get video tasks
type GetTask struct {
	Key       string   `json:"key"`
	Time      int64    `json:"time"`
	Type      string   `json:"type"`
	Path      string   `json:"path"`
	Lock      string   `json:"lock"`
	Command   string   `json:"command"`
	Threads   uint32   `json:"threads"`
	Timeout   uint32   `json:"timeout"`
	Ttltime   uint32   `json:"ttltime"`
	Interval  uint32   `json:"interval"`
	Repeaterr []string `json:"repeaterr"`
	Repeatcnt uint32   `json:"repeatcnt"`
	Replace   bool     `json:"replace"`
	Stdcode   int      `json:"stdcode"`
	Stdout    string   `json:"stdout"`
	Errcode   int      `json:"errcode"`
	Stderr    string   `json:"stderr"`
	Runtime   float64  `json:"runtime"`
}

// PostTask : type task for post video tasks
type PostTask struct {
	Key       string   `json:"key"`
	Type      string   `json:"type"`
	Path      string   `json:"path"`
	Lock      string   `json:"lock"`
	Command   string   `json:"command"`
	Threads   uint32   `json:"threads"`
	Timeout   uint32   `json:"timeout"`
	Ttltime   uint32   `json:"ttltime"`
	Interval  uint32   `json:"interval"`
	Repeaterr []string `json:"repeaterr"`
	Repeatcnt uint32   `json:"repeatcnt"`
	Replace   bool     `json:"replace"`
}

// DelTask : type task for delete video tasks
type DelTask struct {
	Key       string   `json:"key"`
	Time      int64    `json:"time"`
	Type      string   `json:"type"`
	Path      string   `json:"path"`
	Lock      string   `json:"lock"`
	Command   string   `json:"command"`
	Threads   uint32   `json:"threads"`
	Timeout   uint32   `json:"timeout"`
	Ttltime   uint32   `json:"ttltime"`
	Interval  uint32   `json:"interval"`
	Repeaterr []string `json:"repeaterr"`
	Repeatcnt uint32   `json:"repeatcnt"`
	Replace   bool     `json:"replace"`
	Stdcode   int      `json:"stdcode"`
	Stdout    string   `json:"stdout"`
	Errcode   int      `json:"errcode"`
	Stderr    string   `json:"stderr"`
	Runtime   float64  `json:"runtime"`
	Delcode   int      `json:"delcode"`
	Delerr    string   `json:"delerr"`
}

// RawTask : raw type task for video tasks
type RawTask struct {
	Time      int64
	Type      string
	Path      string
	Lock      string
	Command   string
	Threads   uint32
	Timeout   uint32
	Ttltime   uint32
	Interval  uint32
	Repeaterr []string
	Repeatcnt uint32
	Replace   bool
	Stdcode   int
	Stdout    string
	Errcode   int
	Stderr    string
	Runtime   float64
}

// FullTask : full type task with key for video tasks
type FullTask struct {
	Key       []byte
	Time      int64
	Type      string
	Path      string
	Lock      string
	Command   string
	Threads   uint32
	Timeout   uint32
	Ttltime   uint32
	Interval  uint32
	Repeaterr []string
	Repeatcnt uint32
	Replace   bool
	Stdcode   int
	Stdout    string
	Errcode   int
	Stderr    string
	Runtime   float64
}

// ResetTask : reset type task with key only
type ResetTask struct {
	Key []byte
}

// Global Variables

var (

	// Shutdown Mutex

	lsht = &sync.Mutex{}

	// Config

	config     Config
	configfile string = "/etc/ctrl/ctrl.conf"

	onlyssl bool = false

	ussallow []UssAllow
	ipsallow []IpsAllow

	readtimeout       time.Duration = 60 * time.Second
	readheadertimeout time.Duration = 5 * time.Second
	writetimeout      time.Duration = 60 * time.Second
	idletimeout       time.Duration = 60 * time.Second
	keepalive         bool          = false

	// machid string = "nomachineid"

	shutdown bool = false

	debugmode bool = false

	gcpercent int = 25

	dbdir string = "/var/lib/ctrl/db"

	schtime time.Duration = 5

	vc = struct {
		sync.RWMutex
		vcounter map[string]int
	}{vcounter: make(map[string]int)}

	rc = struct {
		sync.RWMutex
		rcounter map[string]int
	}{rcounter: make(map[string]int)}

	rcnt = struct {
		sync.RWMutex
		trycounter map[string]int
	}{trycounter: make(map[string]int)}

	logdir  string = "/var/log/ctrl"
	logmode os.FileMode

	pidfile string = "/run/ctrl/ctrl.pid"

	rgxpair = regexp.MustCompile("^(.+):(.+)$")
	rgxcln  = regexp.MustCompile(":")
)

// Init Function

func init() {

	var err error

	var version string = "1.1.2"
	var vprint bool = false
	var help bool = false

	// Command Line Options

	flag.StringVar(&configfile, "config", configfile, "--config=/etc/ctrl/ctrl.conf")
	flag.BoolVar(&debugmode, "debug", debugmode, "--debug - debug mode")
	flag.BoolVar(&vprint, "version", vprint, "--version - print version")
	flag.BoolVar(&help, "help", help, "--help - displays help")

	flag.Parse()

	switch {
	case vprint:
		fmt.Printf("cTRL Version: %s\n", version)
		os.Exit(0)
	case help:
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Load Configuration

	if _, err = toml.DecodeFile(configfile, &config); err != nil {
		fmt.Printf("Can`t decode config file error | File [%s] | %v\n", configfile, err)
		os.Exit(1)
	}

	// Check Global Options

	rgxonlyssl := regexp.MustCompile("^(?i)(true|false)$")
	mchonlyssl := rgxonlyssl.MatchString(fmt.Sprintf("%t", config.Global.ONLYSSL))
	Check(mchonlyssl, "[global]", "onlyssl", fmt.Sprintf("%t", config.Global.ONLYSSL), "true or false", DoExit)

	mchreadtimeout := RBInt(config.Global.READTIMEOUT, 0, 86400)
	Check(mchreadtimeout, "[global]", "readtimeout", fmt.Sprintf("%d", config.Global.READTIMEOUT), "from 0 to 86400", DoExit)

	mchreadheadertimeout := RBInt(config.Global.READHEADERTIMEOUT, 0, 86400)
	Check(mchreadheadertimeout, "[global]", "readheadertimeout", fmt.Sprintf("%d", config.Global.READHEADERTIMEOUT), "from 0 to 86400", DoExit)

	mchwritetimeout := RBInt(config.Global.WRITETIMEOUT, 0, 86400)
	Check(mchwritetimeout, "[global]", "writetimeout", fmt.Sprintf("%d", config.Global.WRITETIMEOUT), "from 0 to 86400", DoExit)

	mchidletimeout := RBInt(config.Global.IDLETIMEOUT, 0, 86400)
	Check(mchidletimeout, "[global]", "idletimeout", fmt.Sprintf("%d", config.Global.IDLETIMEOUT), "from 0 to 86400", DoExit)

	rgxkeepalive := regexp.MustCompile("^(?i)(true|false)$")
	mchkeepalive := rgxkeepalive.MatchString(fmt.Sprintf("%t", config.Global.KEEPALIVE))
	Check(mchkeepalive, "[global]", "keepalive", fmt.Sprintf("%t", config.Global.KEEPALIVE), "true or false", DoExit)

	rgxrealheader := regexp.MustCompile("^([a-zA-Z0-9-_]+)")
	mchrealheader := rgxrealheader.MatchString(config.Global.REALHEADER)
	Check(mchrealheader, "[global]", "realheader", config.Global.REALHEADER, "ex. X-Real-IP", DoExit)

	rgxcharset := regexp.MustCompile("^([a-zA-Z0-9-])+")
	mchcharset := rgxcharset.MatchString(config.Global.CHARSET)
	Check(mchcharset, "[global]", "charset", config.Global.CHARSET, "ex. UTF-8", DoExit)

	rgxdebugmode := regexp.MustCompile("^(?i)(true|false)$")
	mchdebugmode := rgxdebugmode.MatchString(fmt.Sprintf("%t", config.Global.DEBUGMODE))
	Check(mchdebugmode, "[global]", "debugmode", fmt.Sprintf("%t", config.Global.DEBUGMODE), "true or false", DoExit)

	mchgcpercent := RBInt(config.Global.GCPERCENT, -1, 100)
	Check(mchgcpercent, "[global]", "gcpercent", fmt.Sprintf("%d", config.Global.GCPERCENT), "from -1 to 100", DoExit)

	rgxdbdir := regexp.MustCompile("^(/?[^/\x00]*)+/?$")
	mchdbdir := rgxdbdir.MatchString(config.Global.DBDIR)
	Check(mchdbdir, "[global]", "dbdir", config.Global.DBDIR, "ex. /var/lib/ctrl/db", DoExit)

	mchschtime := RBInt(config.Global.SCHTIME, 1, 3600)
	Check(mchschtime, "[global]", "schtime", fmt.Sprintf("%d", config.Global.SCHTIME), "from 1 to 3600", DoExit)

	rgxlogdir := regexp.MustCompile("^(/?[^/\x00]*)+/?$")
	mchlogdir := rgxlogdir.MatchString(config.Global.LOGDIR)
	Check(mchlogdir, "[global]", "logdir", config.Global.LOGDIR, "ex. /var/log/ctrl", DoExit)

	rgxlogmode := regexp.MustCompile("^([0-7]{3})")
	mchlogmode := rgxlogmode.MatchString(fmt.Sprintf("%d", config.Global.LOGMODE))
	Check(mchlogmode, "[global]", "logmode", fmt.Sprintf("%d", config.Global.LOGMODE), "from 0600 to 0666", DoExit)

	rgxpidfile := regexp.MustCompile("^(/?[^/\x00]*)+/?$")
	mchpidfile := rgxpidfile.MatchString(config.Global.PIDFILE)
	Check(mchpidfile, "[global]", "pidfile", config.Global.PIDFILE, "ex. /run/ctrl/ctrl.pid", DoExit)

	// Log Directory

	logdir = config.Global.LOGDIR

	// Log Mode

	clogmode, err := strconv.ParseUint(fmt.Sprintf("%d", config.Global.LOGMODE), 8, 32)
	switch {
	case err != nil || clogmode == 0:
		logmode = os.FileMode(0640)
	default:
		logmode = os.FileMode(clogmode)
	}

	// Output Important Global Configuration Options

	appLogger, applogfile := AppLogger()
	defer applogfile.Close()

	appLogger.Warnf("| Starting cTRL Server [%s]", version)

	switch {
	case config.Global.KEEPALIVE:
		appLogger.Warnf("| KeepAlive [ENABLED]")
	default:
		appLogger.Warnf("| KeepAlive [DISABLED]")
	}

	appLogger.Warnf("| Scheduler Interval Seconds [%d]", config.Global.SCHTIME)
	appLogger.Warnf("| Garbage Collection Percentage [%d]", config.Global.GCPERCENT)

	// Check Server Options

	var section string

	rgxsslcrt := regexp.MustCompile("^(/?[^/\x00]*)+/?$")
	rgxsslkey := regexp.MustCompile("^(/?[^/\x00]*)+/?$")
	rgxshell := regexp.MustCompile("^(/?[^/\x00]*)+/?$")

	for _, Server := range config.Server {

		section = "[server] | Host ["
		section = fmt.Sprintf("%s%s%s", section, Server.HOST, "]")

		if Server.HOST == "" {
			fmt.Printf("Server host cannot be empty error | %s%s\n", section, " | ex. host=\"localhost\"")
			os.Exit(1)
		}

		if Server.SSLCRT != "" {
			mchsslcrt := rgxsslcrt.MatchString(filepath.Clean(Server.SSLCRT))
			Check(mchsslcrt, section, "sslcrt", Server.SSLCRT, "/path/to/sslcrt.pem", DoExit)

			if Server.SSLKEY == "" {
				appLogger.Errorf("| SSL key cannot be empty error | %s | File [%s]", section, Server.SSLKEY)
				fmt.Printf("SSL key cannot be empty error | %s | File [%s]\n", section, Server.SSLKEY)
				os.Exit(1)
			}

			if !FileOrLinkExists(Server.SSLCRT) {
				appLogger.Errorf("| SSL certificate not exists/permission denied error | %s | File [%s]", section, Server.SSLCRT)
				fmt.Printf("SSL certificate not exists/permission denied error | %s | File [%s]\n", section, Server.SSLCRT)
				os.Exit(1)
			}

		}

		if Server.SSLKEY != "" {
			mchsslkey := rgxsslkey.MatchString(filepath.Clean(Server.SSLKEY))
			Check(mchsslkey, section, "sslkey", Server.SSLKEY, "/path/to/sslkey.pem", DoExit)

			if Server.SSLCRT == "" {
				appLogger.Errorf("| SSL certificate cannot be empty error | %s | File [%s]", section, Server.SSLCRT)
				fmt.Printf("SSL certificate cannot be empty error | %s | File [%s]\n", section, Server.SSLCRT)
				os.Exit(1)
			}

			if !FileOrLinkExists(Server.SSLKEY) {
				appLogger.Errorf("| SSL key not exists/permission denied error | %s | File [%s]", section, Server.SSLKEY)
				fmt.Printf("SSL key not exists/permission denied error | %s | File [%s]\n", section, Server.SSLKEY)
				os.Exit(1)
			}

		}

		if Server.USSALLOW != "" {

			var uss UssAllow

			ussfile, err := os.OpenFile(filepath.Clean(Server.USSALLOW), os.O_RDONLY, os.ModePerm)
			if err != nil {
				appLogger.Errorf("Can`t open uss allow file error | %s | File [%s] | %v", section, Server.USSALLOW, err)
				fmt.Printf("Can`t open uss allow file error | %s | File [%s] | %v\n", section, Server.USSALLOW, err)
				os.Exit(1)
			}
			// No need to defer in loop

			uss.Vhost = Server.HOST

			sussallow := bufio.NewScanner(ussfile)
			for sussallow.Scan() {

				line := sussallow.Text()

				mchpair := rgxpair.MatchString(line)
				Check(mchpair, section, "ussallow", Server.USSALLOW, "Bad format user:sha512 pairs", DoExit)

				user := strings.Split(line, ":")[0]
				phsh := strings.Split(line, ":")[1]

				uss.PAIR = append(uss.PAIR, struct {
					User string
					Hash string
				}{user, phsh})

			}

			err = ussfile.Close()
			if err != nil {
				appLogger.Errorf("Close after read uss allow file error | %s | File [%s] | %v\n", section, Server.USSALLOW, err)
				fmt.Printf("Close after read uss allow file error | %s | File [%s] | %v\n", section, Server.USSALLOW, err)
				os.Exit(1)
			}

			err = sussallow.Err()
			if err != nil {
				fmt.Printf("Read lines from a uss allow file error | %s | File [%s] | %v\n", section, Server.USSALLOW, err)
				return
			}

			ussallow = append(ussallow, uss)

		}

		if Server.IPSALLOW != "" {

			var ips IpsAllow

			ipsfile, err := os.OpenFile(filepath.Clean(Server.IPSALLOW), os.O_RDONLY, os.ModePerm)
			if err != nil {
				appLogger.Errorf("Can`t open ips allow file error | %s | File [%s] | %v", section, Server.IPSALLOW, err)
				fmt.Printf("Can`t open ips allow file error | %s | File [%s] | %v\n", section, Server.IPSALLOW, err)
				os.Exit(1)
			}
			// No need to defer in loop

			ips.Vhost = Server.HOST

			sipsallow := bufio.NewScanner(ipsfile)
			for sipsallow.Scan() {

				line := sipsallow.Text()

				_, _, err = net.ParseCIDR(line)
				if err != nil {
					appLogger.Errorf("| Bad CIDR line format in a ips allow file error | %s | File [%s] | Line [%s]", section, Server.IPSALLOW, line)
					fmt.Printf("Bad CIDR line format in a ips allow file error | %s | File [%s] | Line [%s]\n", section, Server.IPSALLOW, line)
					os.Exit(1)
				}

				ips.CIDR = append(ips.CIDR, struct{ Addr string }{line})

			}

			err = ipsfile.Close()
			if err != nil {
				appLogger.Errorf("Close after read ips allow file error | %s | File [%s] | %v\n", section, Server.IPSALLOW, err)
				fmt.Printf("Close after read ips allow file error | %s | File [%s] | %v\n", section, Server.IPSALLOW, err)
				os.Exit(1)
			}

			err = sipsallow.Err()
			if err != nil {
				fmt.Printf("Read lines from a ips allow file error | %s | File [%s] | %v\n", section, Server.IPSALLOW, err)
				return
			}

			ipsallow = append(ipsallow, ips)

		}

		mchshell := rgxshell.MatchString(Server.SHELL)
		Check(mchshell, section, "shell", Server.SHELL, "ex. /bin/bash", DoExit)

		mchrthreads := RBUint(Server.RTHREADS, 1, 4096)
		Check(mchrthreads, section, "rthreads", fmt.Sprintf("%d", Server.RTHREADS), "from 1 to 4096", DoExit)

		mchvthreads := RBUint(Server.VTHREADS, 1, 4096)
		Check(mchvthreads, section, "vthreads", fmt.Sprintf("%d", Server.VTHREADS), "from 1 to 4096", DoExit)

		mchvtimeout := RBUint(Server.VTIMEOUT, 1, 2592000)
		Check(mchvtimeout, section, "vtimeout", fmt.Sprintf("%d", Server.VTIMEOUT), "from 1 to 2592000", DoExit)

		mchvttltime := RBUint(Server.VTTLTIME, 1, 2592000)
		Check(mchvttltime, section, "vttltime", fmt.Sprintf("%d", Server.VTTLTIME), "from 1 to 2592000", DoExit)

		mchvinterval := RBUint(Server.VINTERVAL, 0, 60)
		Check(mchvinterval, section, "vinterval", fmt.Sprintf("%d", Server.VINTERVAL), "from 0 to 60", DoExit)

		mchvrepeatcnt := RBUint(Server.VREPEATCNT, 0, 1000)
		Check(mchvrepeatcnt, section, "vrepeatcnt", fmt.Sprintf("%d", Server.VREPEATCNT), "from 0 to 1000", DoExit)

		// Output Important Server Configuration Options

		appLogger.Warnf("| Host [%s] | Shell [%s]", Server.HOST, Server.SHELL)
		appLogger.Warnf("| Host [%s] | Scheduler Run Threads Count [%d]", Server.HOST, Server.RTHREADS)
		appLogger.Warnf("| Host [%s] | Scheduler Task Threads Count [%d]", Server.HOST, Server.VTHREADS)
		appLogger.Warnf("| Host [%s] | Scheduler Task Timeout Seconds [%d]", Server.HOST, Server.VTIMEOUT)
		appLogger.Warnf("| Host [%s] | Scheduler Task TTL Seconds [%d]", Server.HOST, Server.VTTLTIME)
		appLogger.Warnf("| Host [%s] | Scheduler Task Interval Seconds [%d]", Server.HOST, Server.VINTERVAL)
		appLogger.Warnf("| Host [%s] | Scheduler Task Repeat Tries [%d]", Server.HOST, Server.VREPEATCNT)

	}

	// Debug Option

	if !debugmode {
		debugmode = config.Global.DEBUGMODE
	}

}

// Main Function

func main() {

	var err error

	// Main WaitGroup

	var wg sync.WaitGroup

	// System Handling

	// Get Pid

	gpid, fpid := GetPID()

	// Log Directory

	logdir = filepath.Clean(config.Global.LOGDIR)

	if !DirExists(logdir) {
		fmt.Printf("Log directory not exists error | Path: [%s]\n", logdir)
		os.Exit(1)
	}

	appLogger, applogfile := AppLogger()
	defer applogfile.Close()

	// PID File

	pidfile = filepath.Clean(config.Global.PIDFILE)

	switch {
	case FileExists(pidfile):
		err = os.Remove(pidfile)
		if err != nil {
			appLogger.Errorf("| Can`t remove pid file error | File [%s] | %v", pidfile, err)
			fmt.Printf("Can`t remove pid file error | File [%s] | %v\n", pidfile, err)
			os.Exit(1)
		}
		fallthrough
	default:
		err = ioutil.WriteFile(pidfile, []byte(fpid), 0644)
		if err != nil {
			appLogger.Errorf("| Can`t create pid file error | File [%s] | %v", pidfile, err)
			fmt.Printf("Can`t create pid file error | File [%s] | %v\n", pidfile, err)
			os.Exit(1)
		}

	}

	// Application Options

	schtime = time.Duration(config.Global.SCHTIME)

	// Cron Scheduler

	cron := gron.New()

	// Only SSL

	onlyssl = config.Global.ONLYSSL

	// KeepAlive

	keepalive = config.Global.KEEPALIVE

	// Pid Handling

	appLogger.Warnf("cTRL server running with a pid: %s", gpid)

	// Key Mapped Mutex

	keymutex := mmutex.NewMMutex()

	// Close Mapped Mutex

	clsmutex := mmutex.NewMMutex()

	// Database

	dbdir = filepath.Clean(config.Global.DBDIR)

	if !DirExists(dbdir) {

		err = os.MkdirAll(dbdir, 0700)
		if err != nil {
			appLogger.Errorf("| Can`t create db directory error | DB Directory [%s] | %v", dbdir, err)
			fmt.Printf("Can`t create db directory error | DB Directory [%s] | %v\n", dbdir, err)
			os.Exit(1)
		}

	}

	clopt := nutsdb.DefaultOptions
	clopt.Dir = dbdir
	// clopt.EntryIdxMode = nutsdb.HintKeyValAndRAMIdxMode
	clopt.EntryIdxMode = nutsdb.HintKeyAndRAMIdxMode
	// clopt.EntryIdxMode = nutsdb.HintBPTSparseIdxMode
	clopt.SegmentSize = 67108864
	clopt.NodeNum = 1
	clopt.StartFileLoadingMode = nutsdb.MMap

	clopt.RWMode = nutsdb.FileIO
	clopt.SyncEnable = true

	cldb, err := nutsdb.Open(clopt)
	if err != nil {
		appLogger.Errorf("| Can`t open/create db error | DB Directory [%s] | %v", dbdir, err)
		fmt.Printf("Can`t open/create db error | DB Directory [%s] | %v\n", dbdir, err)
		os.Exit(1)
	}
	defer cldb.Close()

	// Shell

	//	shell = filepath.Clean(config.Global.SHELL)

	// Reset working queue

	ResetWorking(cldb, &wg)

	cron.AddFunc(gron.Every(schtime*time.Second), func() {
		wg.Add(1)
		CtrlScheduler(cldb, keymutex)
		wg.Done()
	})

	// Garbage Collection Percent

	gcpercent = config.Global.GCPERCENT
	debug.SetGCPercent(gcpercent)

	// Web Server

	app := iris.New()

	switch debugmode {
	case true:
		app.Logger().SetLevel("debug")
		app.Logger().AddOutput(applogfile)
	case false:
		app.Logger().SetLevel("warn")
		app.Logger().SetOutput(applogfile)
	}

	app.Use(logger.New())
	app.Use(recover.New())

	// Web Routing

	app.Get("/show", CtrlShow(cldb, &wg))
	app.Get("/del", CtrlDel(cldb, keymutex, &wg))

	app.Post("/run", CtrlRun(clsmutex, &wg))
	app.Post("/task", CtrlTask(cldb, &wg))

	// Interrupt Handler

	iris.RegisterOnInterrupt(func() {

		// Shutdown Server

		appLogger.Warnf("Stop receive new requests")
		appLogger.Warnf("Capture interrupt")
		appLogger.Warnf("Notify go routines about interrupt")

		lsht.Lock()
		shutdown = true
		lsht.Unlock()

		// Wait Go Routines

		appLogger.Warnf("Awaiting all go routines")

		wg.Wait()

		appLogger.Warnf("Finished all go routines")

		timeout := 5 * time.Second

		// Merge DB

		appLogger.Warnf("Merging db")

		err = NDBMerge(cldb, dbdir)
		if err != nil {
			appLogger.Errorf("Merge db error | %v", err)
		}

		appLogger.Warnf("Finished merge db")

		// Stop Iris

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		appLogger.Warnf("Shutdown cTRL server completed")

		err = app.Shutdown(ctx)
		if err != nil {
			fmt.Printf("Something wrong when shutdown cTRL Server | %v\n", err)
			os.Exit(1)
		}

		// Remove PID File

		if FileExists(pidfile) {
			err = os.Remove(pidfile)
			if err != nil {
				appLogger.Errorf("| Can`t remove pid file error | File [%s] | %v", pidfile, err)
				fmt.Printf("Can`t remove pid file error | File [%s] | %v\n", pidfile, err)
				os.Exit(1)
			}
		}

	})

	// TLS Configuration

	var sstd bool = false
	var stls bool = false
	var stdcount int = 0
	var tlscount int = 0

	tlsConfig := &tls.Config{}

	for _, Server := range config.Server {

		if Server.SSLCRT != "" && Server.SSLKEY != "" {
			tlscount++
		}

		if !onlyssl {
			stdcount++
		}

	}

	if tlscount > 0 {

		tlsConfig.Certificates = make([]tls.Certificate, tlscount)

		s := 0

		for _, Server := range config.Server {

			if Server.SSLCRT != "" && Server.SSLKEY != "" {

				tlsConfig.Certificates[s], err = tls.LoadX509KeyPair(Server.SSLCRT, Server.SSLKEY)
				if err != nil {
					appLogger.Errorf("| Can`t apply ssl certificate/key error | Certificate [%s] | Key [%s] | %v", Server.SSLCRT, Server.SSLKEY, err)
					fmt.Printf("Can`t apply ssl certificate/key error | Certificate [%s] | Key [%s] | %v\n", Server.SSLCRT, Server.SSLKEY, err)
					os.Exit(1)
				}

				s++

			}

		}

		tlsConfig.BuildNameToCertificate()

		stls = true

	}

	if stdcount > 0 {
		sstd = true
	}

	// Configure App

	charset := config.Global.CHARSET
	realheader := config.Global.REALHEADER

	app.Configure(iris.WithoutInterruptHandler, iris.WithoutBodyConsumptionOnUnmarshal, iris.WithCharset(charset), iris.WithRemoteAddrHeader(realheader), iris.WithOptimizations, iris.WithConfiguration(iris.Configuration{
		DisablePathCorrection: false,
		EnablePathEscape:      false,
		TimeFormat:            "Mon, 02 Jan 2006 15:04:05 GMT",
		Charset:               charset,
	}))

	// Build App

	err = app.Build()
	if err != nil {
		fmt.Printf("Something wrong when building cTRL Server | %v\n", err)
		os.Exit(1)
	}

	// Timeouts

	readtimeout = time.Duration(config.Global.READTIMEOUT) * time.Second
	readheadertimeout = time.Duration(config.Global.READHEADERTIMEOUT) * time.Second
	writetimeout = time.Duration(config.Global.WRITETIMEOUT) * time.Second
	idletimeout = time.Duration(config.Global.IDLETIMEOUT) * time.Second

	// Start Cron Scheduler

	cron.Start()

	// Start WebServer

	switch {

	case sstd && !stls:

		bindaddr := config.Global.BINDADDR
		switch {
		case bindaddr == "":
			bindaddr = "127.0.0.1:9691"
		}

		srv := &http.Server{
			Addr:              bindaddr,
			ReadTimeout:       readtimeout,
			ReadHeaderTimeout: readheadertimeout,
			IdleTimeout:       idletimeout,
			WriteTimeout:      writetimeout,
			MaxHeaderBytes:    1 << 20,
		}

		srv.SetKeepAlivesEnabled(keepalive)

		err = app.Run(iris.Server(srv))
		if err != nil && !shutdown {
			fmt.Printf("Something wrong when starting cTRL Server | %v\n", err)
			os.Exit(1)
		}

	case !sstd && stls:

		bindaddrssl := config.Global.BINDADDRSSL
		switch {
		case bindaddrssl == "":
			bindaddrssl = "127.0.0.1:9791"
		}

		srvssl := &http.Server{
			Addr:              bindaddrssl,
			ReadTimeout:       readtimeout,
			ReadHeaderTimeout: readheadertimeout,
			IdleTimeout:       idletimeout,
			WriteTimeout:      writetimeout,
			MaxHeaderBytes:    1 << 20,
			TLSConfig:         tlsConfig,
		}

		srvssl.SetKeepAlivesEnabled(keepalive)

		err = app.Run(iris.Server(srvssl))
		if err != nil && !shutdown {
			fmt.Printf("Something wrong when starting cTRL Server | %v\n", err)
			os.Exit(1)
		}

	case sstd && stls:

		bindaddr := config.Global.BINDADDR
		switch {
		case bindaddr == "":
			bindaddr = "127.0.0.1:9691"
		}

		bindaddrssl := config.Global.BINDADDRSSL
		switch {
		case bindaddrssl == "":
			bindaddrssl = "127.0.0.1:9791"
		}

		srv := &http.Server{
			Handler:           app,
			Addr:              bindaddr,
			ReadTimeout:       readtimeout,
			ReadHeaderTimeout: readheadertimeout,
			IdleTimeout:       idletimeout,
			WriteTimeout:      writetimeout,
			MaxHeaderBytes:    1 << 20,
		}

		srvssl := &http.Server{
			Addr:              bindaddrssl,
			ReadTimeout:       readtimeout,
			ReadHeaderTimeout: readheadertimeout,
			IdleTimeout:       idletimeout,
			WriteTimeout:      writetimeout,
			MaxHeaderBytes:    1 << 20,
			TLSConfig:         tlsConfig,
		}

		srv.SetKeepAlivesEnabled(keepalive)
		srvssl.SetKeepAlivesEnabled(keepalive)

		go srv.ListenAndServe()

		if debugmode {
			appLogger.Debugf("Application: running using 1 host(s)")
			appLogger.Debugf("Host: addr is %s", bindaddr)
			appLogger.Debugf("Host: virtual host is %s", bindaddr)
			appLogger.Debugf("Host: register startup notifier")
			appLogger.Debugf("Now listening on: http://%s", bindaddr)
		} else {
			appLogger.Warnf("Now listening on: http://%s", bindaddr)
		}

		err = app.Run(iris.Server(srvssl))
		if err != nil && !shutdown {
			fmt.Printf("Something wrong when starting cTRL Server | %v\n", err)
			os.Exit(1)
		}

	default:

		appLogger.Errorf("| Not configured any virtual host. Must check config [%s]", configfile)
		fmt.Printf("Not configured any virtual host. Must check config [%s]\n", configfile)
		os.Exit(1)

	}

}
