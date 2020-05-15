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
	"errors"
	"github.com/eltaline/go-waitgroup"
	"github.com/eltaline/nutsdb"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// CtrlScheduler : Control threads scheduler
func CtrlScheduler(cldb *nutsdb.DB) {

	// Variables

	serr := errors.New("shutdown operation")

	bprefix := []byte("t:")

	// Throttling

	vhc := 0

	for range config.Server {
		vhc++
	}

	vwg := waitgroup.NewWaitGroup(vhc)

	for _, Server := range config.Server {

		prevhost := Server.HOST
		prevshell := Server.SHELL
		prevthreads := Server.VTHREADS
		prevttltime := Server.VTTLTIME

		vc.RLock()
		curthreads := vc.vcounter[prevhost]
		vc.RUnlock()

		if curthreads >= prevthreads {
			continue
		}

		prelimit := prevthreads - curthreads

		if prelimit <= 0 {
			continue
		}

		vwg.Add(func() {

			vhost := prevhost
			shell := prevshell
			vttltime := prevttltime

			limit := prelimit

			rvbucket := "recv" + "_" + vhost + ":"
			wvbucket := "work" + "_" + vhost + ":"
			fvbucket := "comp" + "_" + vhost + ":"

			mcompare := make(map[string]bool)

			errempty := errors.New("bucket is empty")
			errscans := errors.New("prefix scans not found")

			var ftsk []FullTask
			var f FullTask
			etsk := &RawTask{}

			// Loggers

			appLogger, applogfile := AppLogger()
			defer applogfile.Close()

			// Shutdown

			if shutdown {
				return
			}

			cerr := cldb.View(func(tx *nutsdb.Tx) error {

				var err error

				var received nutsdb.Entries
				var working nutsdb.Entries
				var completed nutsdb.Entries

				received, _, err = tx.PrefixScan(rvbucket, bprefix, -1, -1)

				if received == nil {
					return nil
				}

				if err != nil {
					return err
				}

				working, err = tx.GetAll(wvbucket)

				if err != nil && err.Error() != errempty.Error() {
					return err
				}

			Main:

				for _, recv := range received {

					if shutdown {
						return serr
					}

					var rv RawTask

					rkey := recv.Key

					rvdec := gob.NewDecoder(bytes.NewReader(recv.Value))
					err := rvdec.Decode(&rv)
					if err != nil {
						appLogger.Errorf("| Gob decode from db error | %v", err)
						continue
					}

					pair := rv.Type + "_" + rv.Lock
					_, found := mcompare[pair]

					if found {
						continue
					}

					if !found {
						mcompare[pair] = true
					}

					completed, _, err = tx.PrefixScan(fvbucket, rkey, -1, 1)

					if completed != nil {
						continue
					}

					if err != nil && err.Error() != errscans.Error() {
						return err
					}

					if working != nil {

						for _, work := range working {

							var wv RawTask

							wvdec := gob.NewDecoder(bytes.NewReader(work.Value))
							err := wvdec.Decode(&wv)
							if err != nil {
								appLogger.Errorf("| Gob decode from db error | %v", err)
								continue
							}

							if rv.Type == wv.Type && rv.Lock == wv.Lock {
								continue Main
							}

						}

					}

					f.Key = recv.Key
					f.Time = rv.Time
					f.Type = rv.Type
					f.Path = rv.Path
					f.Lock = rv.Lock
					f.Command = rv.Command
					f.Timeout = rv.Timeout
					f.Stdcode = rv.Stdcode
					f.Stdout = rv.Stdout
					f.Errcode = rv.Errcode
					f.Stderr = rv.Stderr
					f.Runtime = rv.Runtime

					ftsk = append(ftsk, f)

				}

				return nil

			})

			if cerr != nil {
				appLogger.Errorf("| Work with db error | %v", cerr)
				return
			}

			qwg := waitgroup.NewWaitGroup(limit)

			tscounter := 0

			for _, task := range ftsk {

				if shutdown {
					break
				}

				tscounter++

				if tscounter > limit {
					break
				}

				prefbkey := task.Key
				prefskey := string(task.Key)
				preftmst := task.Time
				preftype := task.Type
				prefpath := task.Path
				preflock := task.Lock
				prefcomm := task.Command
				preftout := task.Timeout

				vc.Lock()
				vc.vcounter[vhost]++
				vc.Unlock()

				qwg.Add(func() {

					defer func() {
						vc.Lock()
						vc.vcounter[vhost]--
						vc.Unlock()
					}()

					var err error

					bkey := prefbkey
					skey := prefskey
					ftmst := preftmst
					ftype := preftype
					fpath := prefpath
					flock := preflock
					fcomm := prefcomm
					ftout := preftout

					stdcode := 0
					errcode := 0

					stdout := ""
					stderr := ""

					pbuffer := new(bytes.Buffer)
					penc := gob.NewEncoder(pbuffer)

					if !DirExists(fpath) {

						appLogger.Errorf("| Can`t find directory error | Path [%s] | %v", fpath, err)

						etsk = &RawTask{
							Time:    ftmst,
							Type:    ftype,
							Path:    fpath,
							Lock:    flock,
							Command: fcomm,
							Timeout: ftout,
							Stdcode: stdcode,
							Stdout:  stdout,
							Errcode: errcode,
							Stderr:  "Can`t open file error",
							Runtime: float64(0),
						}

						err = penc.Encode(etsk)
						if err != nil {
							appLogger.Errorf("| Gob task encode error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", skey, fpath, flock, fcomm, err)
							return
						}

						err = NDBInsert(cldb, fvbucket, bkey, pbuffer.Bytes(), 0)
						if err != nil {
							appLogger.Errorf("| Insert completed task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", skey, fpath, flock, fcomm, err)
							return
						}

						err = NDBDelete(cldb, rvbucket, bkey)
						if err != nil {
							appLogger.Errorf("| Delete received task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", skey, fpath, flock, fcomm, err)
							return
						}

						return

					}

					_, err = os.Stat(fpath)
					if err != nil {

						appLogger.Errorf("| Can`t stat directory error | Path [%s] | %v", fpath, err)

						etsk = &RawTask{
							Time:    ftmst,
							Type:    ftype,
							Path:    fpath,
							Lock:    flock,
							Command: fcomm,
							Timeout: ftout,
							Stdcode: stdcode,
							Stdout:  stdout,
							Errcode: errcode,
							Stderr:  "Can`t stat file error",
							Runtime: float64(0),
						}

						err = penc.Encode(etsk)
						if err != nil {
							appLogger.Errorf("| Gob task encode error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", skey, fpath, flock, fcomm, err)
							return
						}

						err = NDBInsert(cldb, fvbucket, bkey, pbuffer.Bytes(), 0)
						if err != nil {
							appLogger.Errorf("| Insert completed task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", skey, fpath, flock, fcomm, err)
							return
						}

						err = NDBDelete(cldb, rvbucket, bkey)
						if err != nil {
							appLogger.Errorf("| Delete received task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", skey, fpath, flock, fcomm, err)
							return
						}

						return

					}

					var cmmout bytes.Buffer
					var cmmerr bytes.Buffer

					scm := "timeout" + " " + strconv.FormatUint(uint64(ftout), 10) + " " + shell + " -c " + "\"cd " + fpath + " " + "&&" + " " +  fcomm + "\""
					cmm := exec.Command(shell, "-c", scm)

					etsk = &RawTask{
						Time:    ftmst,
						Type:    ftype,
						Path:    fpath,
						Lock:    flock,
						Command: fcomm,
						Timeout: ftout,
						Stdcode: stdcode,
						Stdout:  stdout,
						Errcode: errcode,
						Stderr:  "",
						Runtime: float64(0),
					}

					err = penc.Encode(etsk)
					if err != nil {
						appLogger.Errorf("| Gob task encode error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", skey, fpath, flock, fcomm, err)
						return
					}

					err = NDBInsert(cldb, wvbucket, bkey, pbuffer.Bytes(), 0)
					if err != nil {
						appLogger.Errorf("| Insert working task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", skey, fpath, flock, fcomm, err)
						return
					}

					var rtime float64

					stime := time.Now()

					cmm.Stdout = &cmmout
					cmm.Stderr = &cmmerr

					interrupt := false

					err = cmm.Run()
					if err != nil {
						interrupt = true
						errcode = 1
						stderr = err.Error()
						appLogger.Errorf("| Execute command error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", skey, fpath, flock, scm, err)
					}

					rtime = float64(time.Since(stime)) / float64(time.Millisecond)

					stdout = cmmout.String()

					if !interrupt {
						stderr = cmmerr.String()
					}

					ebuffer := new(bytes.Buffer)
					eenc := gob.NewEncoder(ebuffer)

					etsk = &RawTask{
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
						Runtime: rtime,
					}

					err = eenc.Encode(etsk)
					if err != nil {
						appLogger.Errorf("| Gob task encode error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", skey, fpath, flock, fcomm, err)
						return
					}

					err = NDBInsert(cldb, fvbucket, bkey, ebuffer.Bytes(), vttltime)
					if err != nil {
						appLogger.Errorf("| Insert completed task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", skey, fpath, flock, fcomm, err)
						return
					}

					err = NDBDelete(cldb, wvbucket, bkey)
					if err != nil {
						appLogger.Errorf("| Delete working task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", skey, fpath, flock, fcomm, err)
						return
					}

					err = NDBDelete(cldb, rvbucket, bkey)
					if err != nil {
						appLogger.Errorf("| Delete received task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", skey, fpath, flock, fcomm, err)
						return
					}

				})

			}

			qwg.Wait()

		})

	}

	vwg.Wait()

}
