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
	"crypto/sha512"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/eltaline/mmutex"
	"github.com/eltaline/nutsdb"
	"github.com/kataras/iris/v12"
	"net"
	"strings"
	"sync"
)

// Get

// CtrlShow : Show task command or commands method
func CtrlShow(cldb *nutsdb.DB, wg *sync.WaitGroup) iris.Handler {
	return func(ctx iris.Context) {
		defer wg.Done()

		var err error

		// Wait Group

		wg.Add(1)

		// Loggers

		getLogger, getlogfile := GetLogger()
		defer getlogfile.Close()

		// Shutdown

		if shutdown {
			ctx.StatusCode(iris.StatusInternalServerError)
			return
		}

		// Variables

		errempty := errors.New("bucket is empty")
		errscans := errors.New("prefix scans not found")
		errsearchscans := errors.New("prefix and search scans not found")

		bprefix := []byte("t:")

		var ftsk []FullTask
		var f FullTask
		var resp []GetTask
		var p GetTask

		// IP Client

		ip := ctx.RemoteAddr()
		cip := net.ParseIP(ip)
		ush := ctx.GetHeader("Auth")
		vhost := strings.Split(ctx.Host(), ":")[0]

		key := ctx.URLParam("key")
		queue := ctx.URLParam("queue")

		params := ctx.URLParams()

		badhost := true
		baduser := true
		badip := true

		user := ""
		pass := ""
		phsh := ""

		rvbucket := ""
		wvbucket := ""
		fvbucket := ""

		if ush != "" {

			mchpair := rgxpair.MatchString(ush)

			if !mchpair {

				ctx.StatusCode(iris.StatusBadRequest)

				getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 400 | Bad authorization format", vhost, ip)

				if debugmode {

					_, err = ctx.Writef("[ERRO] Bad authorization format | Virtual Host [%s]\n", vhost)
					if err != nil {
						getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
					}

				}

				return

			}

			sha_512 := sha512.New()

			user = strings.Split(ush, ":")[0]
			pass = strings.Split(ush, ":")[1]

			sha_512.Write([]byte(pass))
			phsh = fmt.Sprintf("%x", sha_512.Sum(nil))

		}

		for _, Server := range config.Server {

			if vhost == Server.HOST {

				badhost = false

				if user != "" && pass != "" {

					for _, Vhost := range ussallow {

						if vhost == Vhost.Vhost {

							for _, PAIR := range Vhost.PAIR {

								if user == PAIR.User && phsh == PAIR.Hash {
									baduser = false
									break
								}
							}

							break

						}

					}

				}

				if baduser {

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

				}

				rvbucket = "recv" + "_" + vhost + ":"
				wvbucket = "work" + "_" + vhost + ":"
				fvbucket = "comp" + "_" + vhost + ":"

				break

			}

		}

		if badhost {

			ctx.StatusCode(iris.StatusMisdirectedRequest)

			getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 421 | Not found configured virtual host", vhost, ip)

			if debugmode {

				_, err = ctx.Writef("[ERRO] Not found configured virtual host | Virtual Host [%s]\n", vhost)
				if err != nil {
					getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		if baduser && badip {

			ctx.StatusCode(iris.StatusForbidden)

			getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 403 | Forbidden", vhost, ip)

			if debugmode {

				_, err = ctx.Writef("[ERRO] Not found allowed user or not found allowed ip | Virtual Host [%s]\n", vhost)
				if err != nil {
					getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		if len(params) == 0 {

			ctx.StatusCode(iris.StatusForbidden)

			getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 403 | The query arguments is not given during GET request", vhost, ip)

			if debugmode {

				_, err = ctx.WriteString("[ERRO] The query arguments is not given during GET request\n")
				if err != nil {
					getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		if queue != "" || (queue != "" && key != "") {

			cerr := cldb.View(func(tx *nutsdb.Tx) error {

				var err error

				var tasks nutsdb.Entries

				rgxkey := "(.+" + key + ")$"

				switch {
				case key != "" && queue == "received":

					tasks, _, err = tx.PrefixSearchScan(rvbucket, bprefix, rgxkey, -1, -1)
					if tasks == nil {
						return nil
					}

					if err != nil && (err.Error() == errempty.Error() || err.Error() == errscans.Error() || err.Error() == errsearchscans.Error()) {
						return nil
					}

					if err != nil {
						return err
					}

				case key != "" && queue == "working":
					tasks, _, err = tx.PrefixSearchScan(wvbucket, bprefix, rgxkey, -1, -1)

					if tasks == nil {
						return nil
					}

					if err != nil && (err.Error() == errempty.Error() || err.Error() == errscans.Error() || err.Error() == errsearchscans.Error()) {
						return nil
					}

					if err != nil {
						return err
					}

				case key != "" && queue == "completed":
					tasks, _, err = tx.PrefixSearchScan(fvbucket, bprefix, rgxkey, -1, -1)

					if tasks == nil {
						return nil
					}

					if err != nil && (err.Error() == errempty.Error() || err.Error() == errscans.Error() || err.Error() == errsearchscans.Error()) {
						return nil
					}

					if err != nil {
						return err
					}

				case queue == "received":
					tasks, err = tx.GetAll(rvbucket)

					if tasks == nil {
						return nil
					}

					if err != nil && (err.Error() == errempty.Error() || err.Error() == errscans.Error() || err.Error() == errsearchscans.Error()) {
						return nil
					}

					if err != nil {
						return err
					}

				case queue == "working":
					tasks, err = tx.GetAll(wvbucket)

					if tasks == nil {
						return nil
					}

					if err != nil && (err.Error() == errempty.Error() || err.Error() == errscans.Error() || err.Error() == errsearchscans.Error()) {
						return nil
					}

					if err != nil {
						return err
					}

				case queue == "completed":
					tasks, err = tx.GetAll(fvbucket)

					if tasks == nil {
						return nil
					}

					if err != nil && (err.Error() == errempty.Error() || err.Error() == errscans.Error() || err.Error() == errsearchscans.Error()) {
						return nil
					}

					if err != nil {
						return err
					}

				default:

					return nil

				}

				for _, rtask := range tasks {

					var rt RawTask

					rtdec := gob.NewDecoder(bytes.NewReader(rtask.Value))
					err := rtdec.Decode(&rt)
					if err != nil {

						ctx.StatusCode(iris.StatusInternalServerError)

						getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 500 | Gob decode from db error | %v", vhost, ip, err)

						if debugmode {

							_, err = ctx.WriteString("[ERRO] Gob decode from db error\n")
							if err != nil {
								getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
							}

						}

					}

					f.Key = rtask.Key
					f.Time = rt.Time
					f.Type = rt.Type
					f.Path = rt.Path
					f.Lock = rt.Lock
					f.Command = rt.Command
					f.Threads = rt.Threads
					f.Timeout = rt.Timeout
					f.Ttltime = rt.Ttltime
					f.Interval = rt.Interval
					f.Repeaterr = rt.Repeaterr
					f.Repeatcnt = rt.Repeatcnt
					f.Stdcode = rt.Stdcode
					f.Stdout = rt.Stdout
					f.Errcode = rt.Errcode
					f.Stderr = rt.Stderr
					f.Runtime = rt.Runtime

					ftsk = append(ftsk, f)

				}

				return nil

			})

			if len(ftsk) == 0 {

				_, err = ctx.WriteString("[]")
				if err != nil {
					getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

				return

			}

			if cerr != nil {

				ctx.StatusCode(iris.StatusInternalServerError)

				getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 500 |  Get task from db error | Key [%s] | %v", vhost, ip, key, err)

				if debugmode {

					_, err = ctx.WriteString("[ERRO] Get task from  db error\n")
					if err != nil {
						getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
					}

				}

				return

			}

			for _, task := range ftsk {

				spk := strings.Split(string(task.Key), ":")[3]

				p.Key = spk
				p.Time = task.Time
				p.Type = task.Type
				p.Path = task.Path
				p.Lock = task.Lock
				p.Command = task.Command
				p.Threads = task.Threads
				p.Timeout = task.Timeout
				p.Ttltime = task.Ttltime
				p.Interval = task.Interval
				p.Repeaterr = task.Repeaterr
				p.Repeatcnt = task.Repeatcnt
				p.Stdcode = task.Stdcode
				p.Stdout = task.Stdout
				p.Errcode = task.Errcode
				p.Stderr = task.Stderr
				p.Runtime = task.Runtime

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
				getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
			}

			return

		}

		ctx.StatusCode(iris.StatusBadRequest)

		getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 400 | The query is not supported during GET request", vhost, ip)

		if debugmode {

			_, err = ctx.WriteString("[ERRO] The query is not supported during GET request\n")
			if err != nil {
				getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
			}

		}

	}

}

// CtrlDel : Delete task command or commands method
func CtrlDel(cldb *nutsdb.DB, keymutex *mmutex.Mutex, wg *sync.WaitGroup) iris.Handler {
	return func(ctx iris.Context) {
		defer wg.Done()

		var err error

		// Wait Group

		wg.Add(1)

		// Loggers

		getLogger, getlogfile := GetLogger()
		defer getlogfile.Close()

		// Shutdown

		if shutdown {
			ctx.StatusCode(iris.StatusInternalServerError)
			return
		}

		// Variables

		errempty := errors.New("bucket is empty")
		errscans := errors.New("prefix scans not found")
		errsearchscans := errors.New("prefix and search scans not found")

		bprefix := []byte("t:")

		var ftsk []FullTask
		var f FullTask
		var resp []DelTask
		var d DelTask

		bucket := ""

		// IP Client

		ip := ctx.RemoteAddr()
		cip := net.ParseIP(ip)
		ush := ctx.GetHeader("Auth")
		vhost := strings.Split(ctx.Host(), ":")[0]

		key := ctx.URLParam("key")
		queue := ctx.URLParam("queue")

		params := ctx.URLParams()

		badhost := true
		baduser := true
		badip := true

		user := ""
		pass := ""
		phsh := ""

		rvbucket := ""
		wvbucket := ""
		fvbucket := ""

		if ush != "" {

			mchpair := rgxpair.MatchString(ush)

			if !mchpair {

				ctx.StatusCode(iris.StatusBadRequest)

				getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 400 | Bad authorization format", vhost, ip)

				if debugmode {

					_, err = ctx.Writef("[ERRO] Bad authorization format | Virtual Host [%s]\n", vhost)
					if err != nil {
						getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
					}

				}

				return

			}

			sha_512 := sha512.New()

			user = strings.Split(ush, ":")[0]
			pass = strings.Split(ush, ":")[1]

			sha_512.Write([]byte(pass))
			phsh = fmt.Sprintf("%x", sha_512.Sum(nil))

		}

		for _, Server := range config.Server {

			if vhost == Server.HOST {

				badhost = false

				if user != "" && pass != "" {

					for _, Vhost := range ussallow {

						if vhost == Vhost.Vhost {

							for _, PAIR := range Vhost.PAIR {

								if user == PAIR.User && phsh == PAIR.Hash {
									baduser = false
									break
								}
							}

							break

						}

					}

				}

				if baduser {

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

				}

				rvbucket = "recv" + "_" + vhost + ":"
				wvbucket = "work" + "_" + vhost + ":"
				fvbucket = "comp" + "_" + vhost + ":"

				break

			}

		}

		if badhost {

			ctx.StatusCode(iris.StatusMisdirectedRequest)

			getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 421 | Not found configured virtual host", vhost, ip)

			if debugmode {

				_, err = ctx.Writef("[ERRO] Not found configured virtual host | Virtual Host [%s]\n", vhost)
				if err != nil {
					getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		if baduser && badip {

			ctx.StatusCode(iris.StatusForbidden)

			getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 403 | Forbidden", vhost, ip)

			if debugmode {

				_, err = ctx.Writef("[ERRO] Not found allowed user or not found allowed ip | Virtual Host [%s]\n", vhost)
				if err != nil {
					getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		if len(params) == 0 {

			ctx.StatusCode(iris.StatusForbidden)

			getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 403 | The query arguments is not given during GET request", vhost, ip)

			if debugmode {

				_, err = ctx.WriteString("[ERRO] The query arguments is not given during GET request\n")
				if err != nil {
					getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

			}

			return

		}

		if queue != "" || (queue != "" && key != "") {

			cerr := cldb.View(func(tx *nutsdb.Tx) error {

				var err error

				var tasks nutsdb.Entries

				rgxkey := "(.+" + key + ")$"

				switch {
				case key != "" && queue == "received":

					bucket = rvbucket

					tasks, _, err = tx.PrefixSearchScan(rvbucket, bprefix, rgxkey, -1, -1)
					if tasks == nil {
						return nil
					}

					if err != nil && (err.Error() == errempty.Error() || err.Error() == errscans.Error() || err.Error() == errsearchscans.Error()) {
						return nil
					}

					if err != nil {
						return err
					}

				case key != "" && queue == "working":

					bucket = wvbucket

					tasks, _, err = tx.PrefixSearchScan(wvbucket, bprefix, rgxkey, -1, -1)

					if tasks == nil {
						return nil
					}

					if err != nil && (err.Error() == errempty.Error() || err.Error() == errscans.Error() || err.Error() == errsearchscans.Error()) {
						return nil
					}

					if err != nil {
						return err
					}

				case key != "" && queue == "completed":

					bucket = fvbucket

					tasks, _, err = tx.PrefixSearchScan(fvbucket, bprefix, rgxkey, -1, -1)

					if tasks == nil {
						return nil
					}

					if err != nil && (err.Error() == errempty.Error() || err.Error() == errscans.Error() || err.Error() == errsearchscans.Error()) {
						return nil
					}

					if err != nil {
						return err
					}

				case queue == "received":

					bucket = rvbucket

					tasks, err = tx.GetAll(rvbucket)

					if tasks == nil {
						return nil
					}

					if err != nil && (err.Error() == errempty.Error() || err.Error() == errscans.Error() || err.Error() == errsearchscans.Error()) {
						return nil
					}

					if err != nil {
						return err
					}

				case queue == "working":

					bucket = wvbucket

					tasks, err = tx.GetAll(wvbucket)

					if tasks == nil {
						return nil
					}

					if err != nil && (err.Error() == errempty.Error() || err.Error() == errscans.Error() || err.Error() == errsearchscans.Error()) {
						return nil
					}

					if err != nil {
						return err
					}

				case queue == "completed":

					bucket = fvbucket

					tasks, err = tx.GetAll(fvbucket)

					if tasks == nil {
						return nil
					}

					if err != nil && (err.Error() == errempty.Error() || err.Error() == errscans.Error() || err.Error() == errsearchscans.Error()) {
						return nil
					}

					if err != nil {
						return err
					}

				default:

					return nil

				}

				for _, rtask := range tasks {

					var rt RawTask

					rtdec := gob.NewDecoder(bytes.NewReader(rtask.Value))
					err := rtdec.Decode(&rt)
					if err != nil {

						ctx.StatusCode(iris.StatusInternalServerError)

						getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 500 | Gob decode from db error | %v", vhost, ip, err)

						if debugmode {

							_, err = ctx.WriteString("[ERRO] Gob decode from db error\n")
							if err != nil {
								getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
							}

						}

					}

					f.Key = rtask.Key
					f.Time = rt.Time
					f.Type = rt.Type
					f.Path = rt.Path
					f.Lock = rt.Lock
					f.Command = rt.Command
					f.Threads = rt.Threads
					f.Timeout = rt.Timeout
					f.Ttltime = rt.Ttltime
					f.Interval = rt.Interval
					f.Repeaterr = rt.Repeaterr
					f.Repeatcnt = rt.Repeatcnt
					f.Stdcode = rt.Stdcode
					f.Stdout = rt.Stdout
					f.Errcode = rt.Errcode
					f.Stderr = rt.Stderr
					f.Runtime = rt.Runtime

					ftsk = append(ftsk, f)

				}

				return nil

			})

			if len(ftsk) == 0 {

				_, err = ctx.WriteString("[]")
				if err != nil {
					getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
				}

				return

			}

			if cerr != nil {

				ctx.StatusCode(iris.StatusInternalServerError)

				getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 500 | Del task/tasks from db error | Key [%s] | %v", vhost, ip, key, err)

				if debugmode {

					_, err = ctx.WriteString("[ERRO] Del task/tasks from db error\n")
					if err != nil {
						getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
					}

				}

				return

			}

			for _, task := range ftsk {

				delcode := 0

				delerr := ""

				spk := strings.Split(string(task.Key), ":")[3]

				err = NDBDelete(cldb, bucket, task.Key)
				if err != nil {
					delcode = 1
					delerr = err.Error()
					getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 500 | Delete task from db error | Key [%s] | %v", vhost, ip, key, err)
				}

				if bucket == wvbucket {

					skey := string(task.Key)

					cmkey := vhost + ":" + skey

					if keymutex.IsLock(cmkey) {
						keymutex.UnLock(cmkey)
					}

				}

				d.Key = spk
				d.Time = task.Time
				d.Type = task.Type
				d.Path = task.Path
				d.Lock = task.Lock
				d.Command = task.Command
				d.Threads = task.Threads
				d.Timeout = task.Timeout
				d.Ttltime = task.Ttltime
				d.Interval = task.Interval
				d.Repeaterr = task.Repeaterr
				d.Repeatcnt = task.Repeatcnt
				d.Stdcode = task.Stdcode
				d.Stdout = task.Stdout
				d.Errcode = task.Errcode
				d.Stderr = task.Stderr
				d.Runtime = task.Runtime
				d.Delcode = delcode
				d.Delerr = delerr

				resp = append(resp, d)

			}

			jkeys, _ := JSONMarshal(resp, true)
			allkeys := string(jkeys)
			rbytes := []byte(allkeys)
			hsize := fmt.Sprintf("%d", len(rbytes))

			ctx.Header("Content-Type", "application/json")
			ctx.Header("Content-Length", hsize)

			_, err = ctx.Write(rbytes)
			if err != nil {
				getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
			}

			return

		}

		ctx.StatusCode(iris.StatusBadRequest)

		getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 400 | The query is not supported during GET request", vhost, ip)

		if debugmode {

			_, err = ctx.WriteString("[ERRO] The query is not supported during GET request\n")
			if err != nil {
				getLogger.Errorf("| Virtual Host [%s] | Client IP [%s] | 499 | Can`t complete response to client | %v", vhost, ip, err)
			}

		}

	}

}
