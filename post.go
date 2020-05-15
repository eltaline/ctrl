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
	"bytes"
	"encoding/gob"
	//	"encoding/json"
	"fmt"
	"github.com/eltaline/nutsdb"
	"github.com/gobuffalo/uuid"
	"github.com/kataras/iris"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Post

// CtrlRun : Start realtime task command or commands method
func CtrlRun(wg *sync.WaitGroup) iris.Handler {
	return func(ctx iris.Context) {
		defer wg.Done()

		var err error

		// Wait Group

		wg.Add(1)

		// Loggers

		postLogger, postlogfile := PostLogger()
		defer postlogfile.Close()

		// Shutdown

		if shutdown {
			ctx.StatusCode(iris.StatusInternalServerError)
			return
		}

		// Variables

		var body []PostTask
		var resp []GetTask
		var p GetTask

		// IP Client

		ip := ctx.RemoteAddr()
		cip := net.ParseIP(ip)
		vhost := strings.Split(ctx.Host(), ":")[0]

		params := ctx.URLParams()
		length := ctx.GetHeader("Content-Length")

		badhost := true
		badip := true

		shell := "/bin/bash"

		vtimeout := uint32(28800)

		for _, Server := range config.Server {

			if vhost == Server.HOST {

				badhost = false

				for _, Vhost := range ipsallow {

					if vhost == Vhost.Vhost {

						for _, CIDR := range Vhost.CIDR {
							_, ipnet, _ := net.ParseCIDR(CIDR.Addr)
							if ipnet.Contains(cip) {
								badip = false
								break
							}
						}

						break

					}

				}

				shell = Server.SHELL

				vtimeout = Server.VTIMEOUT

				break

			}

		}

		if badhost {

			ctx.StatusCode(iris.StatusMisdirectedRequest)

			postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 421 | Not found configured virtual host", vhost, ip)

			if debugmode {

				_, err = ctx.Writef("[ERRO] Not found configured virtual host | Virtual Host [%s]\n", vhost)
				if err != nil {
					postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		if badip {

			ctx.StatusCode(iris.StatusForbidden)

			postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 403 | Forbidden", vhost, ip)

			if debugmode {

				_, err = ctx.Writef("[ERRO] Not found allowed ip | Virtual Host [%s]\n", vhost)
				if err != nil {
					postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		if len(params) != 0 {

			ctx.StatusCode(iris.StatusForbidden)

			postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 403 | The query arguments is not allowed during POST request", vhost, ip)

			if debugmode {

				_, err = ctx.WriteString("[ERRO] The query arguments is not allowed during POST request\n")
				if err != nil {
					postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		clength, err := strconv.ParseInt(length, 10, 64)
		if err != nil {

			ctx.StatusCode(iris.StatusBadRequest)

			postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 400 | Content length error during POST request | Content-Length [%s] | %v", vhost, ip, length, err)

			if debugmode {

				_, err = ctx.WriteString("[ERRO] Content length error during POST request\n")
				if err != nil {
					postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		if clength == 0 {

			ctx.StatusCode(iris.StatusBadRequest)

			postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 400 | The body was empty during POST request | Content-Length [%s]", vhost, ip, length)

			if debugmode {

				_, err = ctx.WriteString("[ERRO] The body was empty during POST request\n")
				if err != nil {
					postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		err = ctx.ReadJSON(&body)
		if err != nil {

			ctx.StatusCode(iris.StatusBadRequest)

			postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 400 | The body was bad during POST request | Content-Length [%s]", vhost, ip, length)

			if debugmode {

				//				_, err = ctx.WriteString("[ERRO] The body was bad during POST request\n")
				_, err = ctx.Writef("[ERRO] The body was bad during POST request, %v\n", err)

				if err != nil {
					postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		for _, task := range body {

			stdcode := 0
			errcode := 0

			stdout := ""
			stderr := ""

			if task.Key == "" || task.Type == "" || task.Path == "" || task.Lock == "" || task.Command == "" {

				ctx.StatusCode(iris.StatusBadRequest)

				postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 400 | The body was not contains enough parameters during POST request", vhost, ip)

				if debugmode {

					_, err = ctx.WriteString("[ERRO] The body was not contains enough parameters during POST request\n")
					if err != nil {
						postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
					}

				}

				return

			}

			skey := task.Key
			ftmst := time.Now().Unix()
			fpath := task.Path
			flock := task.Lock
			fcomm := task.Command

			ftout := vtimeout

			if task.Timeout > uint32(0) {
				ftout = task.Timeout
			}

			var cmmout bytes.Buffer
			var cmmerr bytes.Buffer

			scm := "timeout" + " " + strconv.FormatUint(uint64(ftout), 10) + " " + shell + " -c " + "\"cd " + fpath + " " + "&&" + " " +  fcomm + "\""
			cmm := exec.Command(shell, "-c", scm)

			var rtime float64

			stime := time.Now()

			cmm.Stdout = &cmmout
			cmm.Stderr = &cmmerr

			err = cmm.Run()
			if err != nil {
				errcode = 1
				postLogger.Errorf("| Execute command error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", skey, fpath, flock, scm, err)
			}

			rtime = float64(time.Since(stime)) / float64(time.Millisecond)

			stdout = cmmout.String()
			stderr = cmmerr.String()

			p.Key = task.Key
			p.Time = ftmst
			p.Type = task.Type
			p.Path = task.Path
			p.Lock = task.Lock
			p.Command = task.Command
			p.Timeout = ftout
			p.Stdcode = stdcode
			p.Stdout = stdout
			p.Errcode = errcode
			p.Stderr = stderr
			p.Runtime = rtime

			resp = append(resp, p)

		}

		jkeys, _ := JSONMarshal(resp, true)
		allkeys := string(jkeys)
		rbytes := []byte(allkeys)
		hsize := fmt.Sprintf("%d", len(rbytes))

		ctx.Header("Content-Type", "application/json")
		ctx.Header("Content-Length", hsize)

		_, err = ctx.Write(rbytes)
		if err != nil {
			postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
		}

	}

}

// CtrlTask : Start scheduled task command or commands method
func CtrlTask(cldb *nutsdb.DB, wg *sync.WaitGroup) iris.Handler {
	return func(ctx iris.Context) {
		defer wg.Done()

		var err error

		// Wait Group

		wg.Add(1)

		// Loggers

		postLogger, postlogfile := PostLogger()
		defer postlogfile.Close()

		// Shutdown

		if shutdown {
			ctx.StatusCode(iris.StatusInternalServerError)
			return
		}

		// Variables

		var body []PostTask

		// IP Client

		ip := ctx.RemoteAddr()
		cip := net.ParseIP(ip)
		vhost := strings.Split(ctx.Host(), ":")[0]

		params := ctx.URLParams()
		length := ctx.GetHeader("Content-Length")

		badhost := true
		badip := true

		vtimeout := uint32(28800)

		rvbucket := ""

		for _, Server := range config.Server {

			if vhost == Server.HOST {

				badhost = false

				for _, Vhost := range ipsallow {

					if vhost == Vhost.Vhost {

						for _, CIDR := range Vhost.CIDR {
							_, ipnet, _ := net.ParseCIDR(CIDR.Addr)
							if ipnet.Contains(cip) {
								badip = false
								break
							}
						}

						break

					}

				}

				vtimeout = Server.VTIMEOUT

				rvbucket = "recv" + "_" + vhost + ":"

				break

			}

		}

		if badhost {

			ctx.StatusCode(iris.StatusMisdirectedRequest)

			postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 421 | Not found configured virtual host", vhost, ip)

			if debugmode {

				_, err = ctx.Writef("[ERRO] Not found configured virtual host | Virtual Host [%s]\n", vhost)
				if err != nil {
					postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		if badip {

			ctx.StatusCode(iris.StatusForbidden)

			postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 403 | Forbidden", vhost, ip)

			if debugmode {

				_, err = ctx.Writef("[ERRO] Not found allowed ip | Virtual Host [%s]\n", vhost)
				if err != nil {
					postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		if len(params) != 0 {

			ctx.StatusCode(iris.StatusForbidden)

			postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 403 | The query arguments is not allowed during POST request", vhost, ip)

			if debugmode {

				_, err = ctx.WriteString("[ERRO] The query arguments is not allowed during POST request\n")
				if err != nil {
					postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		clength, err := strconv.ParseInt(length, 10, 64)
		if err != nil {

			ctx.StatusCode(iris.StatusBadRequest)

			postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 400 | Content length error during POST request | Content-Length [%s] | %v", vhost, ip, length, err)

			if debugmode {

				_, err = ctx.WriteString("[ERRO] Content length error during POST request\n")
				if err != nil {
					postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		if clength == 0 {

			ctx.StatusCode(iris.StatusBadRequest)

			postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 400 | The body was empty during POST request | Content-Length [%s]", vhost, ip, length)

			if debugmode {

				_, err = ctx.WriteString("[ERRO] The body was empty during POST request\n")
				if err != nil {
					postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		err = ctx.ReadJSON(&body)
		if err != nil {

			ctx.StatusCode(iris.StatusBadRequest)

			postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 400 | The body was bad during POST request | Content-Length [%s]", vhost, ip, length)

			if debugmode {

				//				_, err = ctx.WriteString("[ERRO] The body was bad during POST request\n")
				_, err = ctx.Writef("[ERRO] The body was bad during POST request, %v\n", err)

				if err != nil {
					postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		for _, task := range body {

			stdcode := 0
			errcode := 0

			stdout := ""
			stderr := ""

			if task.Key == "" || task.Type == "" || task.Path == "" || task.Lock == "" || task.Command == "" {

				ctx.StatusCode(iris.StatusBadRequest)

				postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 400 | The body was not contains enough parameters during POST request", vhost, ip)

				if debugmode {

					_, err = ctx.WriteString("[ERRO] The body was not contains enough parameters during POST request\n")
					if err != nil {
						postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
					}

				}

				return

			}

			pbuffer := new(bytes.Buffer)
			enc := gob.NewEncoder(pbuffer)

			internal := uuid.Must(uuid.NewV4())

			ftmst := time.Now().Unix()

			bkey := []byte("t:" + fmt.Sprintf("%d", ftmst) + ":" + fmt.Sprintf("%x", internal) + ":" + task.Key)
			skey := task.Key

			ftype := task.Type
			fpath := task.Path
			flock := task.Lock
			fcomm := task.Command

			ftout := vtimeout

			if task.Timeout > uint32(0) {
				ftout = task.Timeout
			}

			etsk := &RawTask{
				Time:    ftmst,
				Type:    ftype,
				Path:    fpath,
				Lock:    flock,
				Command: fcomm,
				Timeout: ftout,
				Stdcode: stdcode,
				Stdout:  stdout,
				Errcode: errcode,
				Stderr:  stderr,
				Runtime: float64(0),
			}

			err = enc.Encode(etsk)
			if err != nil {

				ctx.StatusCode(iris.StatusInternalServerError)

				postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 500 | Gob task encode error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, ip, skey, fpath, flock, fcomm, err)

				if debugmode {

					_, err = ctx.WriteString("[ERRO] Gob task encode error\n")
					if err != nil {
						postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
					}

				}

				return

			}

			err = NDBInsert(cldb, rvbucket, bkey, pbuffer.Bytes(), 0)
			if err != nil {

				ctx.StatusCode(iris.StatusInternalServerError)

				postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 500 | Insert received task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, ip, skey, fpath, flock, fcomm, err)

				if debugmode {

					_, err = ctx.WriteString("[ERRO] Insert received task db error\n")
					if err != nil {
						postLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
					}

				}

				return

			}

		}

	}

}
