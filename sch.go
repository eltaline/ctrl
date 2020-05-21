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
	"github.com/eltaline/mmutex"
	"github.com/eltaline/nutsdb"
	"github.com/pieterclaerhout/go-waitgroup"
	"os"
	"os/exec"
	"regexp"
	"time"
)

// CtrlScheduler : Control threads scheduler
func CtrlScheduler(cldb *nutsdb.DB, keymutex *mmutex.Mutex) {

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

		vwait := make(chan bool)

		vwg.Add(func() {

			vhost := prevhost
			shell := prevshell

			rvbucket := "recv" + "_" + vhost + ":"
			wvbucket := "work" + "_" + vhost + ":"
			fvbucket := "comp" + "_" + vhost + ":"

			vwait <- true

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
						appLogger.Errorf("| Virtual Host [%s] | Gob decode from db error | %v", vhost, err)
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
								appLogger.Errorf("| Virtual Host [%s] | Gob decode from db error | %v", vhost, err)
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
					f.Threads = rv.Threads
					f.Timeout = rv.Timeout
					f.Ttltime = rv.Ttltime
					f.Interval = rv.Interval
					f.Repeaterr = rv.Repeaterr
					f.Repeatcnt = rv.Repeatcnt
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
				appLogger.Errorf("| Virtual Host [%s] | Work with db error | %v", vhost, cerr)
				return
			}

			qwg := waitgroup.NewWaitGroup(-1)

			for _, task := range ftsk {

				if shutdown {
					break
				}

				prefbkey := task.Key
				prefskey := string(task.Key)
				preftmst := task.Time
				preftype := task.Type
				prefpath := task.Path
				preflock := task.Lock
				prefcomm := task.Command
				prefthreads := task.Threads
				preftout := task.Timeout
				prefttl := task.Ttltime
				prefint := task.Interval
				prefrerr := task.Repeaterr
				prefrcnt := task.Repeatcnt

				pretthr := vhost + ":" + preftype

				vc.RLock()
				curthreads := vc.vcounter[pretthr]
				vc.RUnlock()

				if curthreads >= int(prefthreads) {
					continue
				}

				vc.Lock()
				vc.vcounter[pretthr]++
				vc.Unlock()

				qwait := make(chan bool)

				qwg.Add(func() {

					var err error

					tthr := pretthr

					bkey := prefbkey
					skey := prefskey
					ftmst := preftmst
					ftype := preftype
					fpath := prefpath
					flock := preflock
					fcomm := prefcomm
					fthreads := prefthreads
					ftout := preftout
					fttl := prefttl
					fint := prefint
					frerr := prefrerr
					frcnt := prefrcnt

					vc.RLock()
					vtscnt := vc.vcounter[tthr]
					vc.RUnlock()

					qwait <- true

					defer func() {
						vc.Lock()
						vc.vcounter[tthr]--
						vc.Unlock()
					}()

					stdcode := 0
					errcode := 0

					stdout := ""
					stderr := ""

					pbuffer := new(bytes.Buffer)
					penc := gob.NewEncoder(pbuffer)

					if !DirExists(fpath) {

						appLogger.Errorf("| Virtual Host [%s] | Can`t find directory error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)

						etsk = &RawTask{
							Time:      ftmst,
							Type:      ftype,
							Path:      fpath,
							Lock:      flock,
							Command:   fcomm,
							Threads:   fthreads,
							Timeout:   ftout,
							Ttltime:   fttl,
							Interval:  fint,
							Repeaterr: frerr,
							Repeatcnt: frcnt,
							Stdcode:   stdcode,
							Stdout:    stdout,
							Errcode:   errcode,
							Stderr:    "Can`t open file error",
							Runtime:   float64(0),
						}

						err = penc.Encode(etsk)
						if err != nil {
							appLogger.Errorf("| Virtual Host [%s] | Gob task encode error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							return
						}

						err = NDBDelete(cldb, rvbucket, bkey)
						if err != nil {
							appLogger.Errorf("| Virtual Host [%s] | Delete received task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							return
						}

						err = NDBInsert(cldb, fvbucket, bkey, pbuffer.Bytes(), 0)
						if err != nil {
							appLogger.Errorf("| Virtual Host [%s] | Insert completed task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							return
						}

						return

					}

					_, err = os.Stat(fpath)
					if err != nil {

						appLogger.Errorf("| Virtual Host [%s] | Can`t stat directory error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)

						etsk = &RawTask{
							Time:      ftmst,
							Type:      ftype,
							Path:      fpath,
							Lock:      flock,
							Command:   fcomm,
							Threads:   fthreads,
							Timeout:   ftout,
							Ttltime:   fttl,
							Interval:  fint,
							Repeaterr: frerr,
							Repeatcnt: frcnt,
							Stdcode:   stdcode,
							Stdout:    stdout,
							Errcode:   errcode,
							Stderr:    "Can`t stat file error",
							Runtime:   float64(0),
						}

						err = penc.Encode(etsk)
						if err != nil {
							appLogger.Errorf("| Virtual Host [%s] | Gob task encode error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							return
						}

						err = NDBDelete(cldb, rvbucket, bkey)
						if err != nil {
							appLogger.Errorf("| Virtual Host [%s] | Delete received task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							return
						}

						err = NDBInsert(cldb, fvbucket, bkey, pbuffer.Bytes(), 0)
						if err != nil {
							appLogger.Errorf("| Virtual Host [%s] | Insert completed task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							return
						}

						return

					}

					var cmmout bytes.Buffer
					var cmmerr bytes.Buffer

					scm := shell + " -c " + "\"cd " + fpath + " && " + fcomm + "\""
					cmm := exec.Command(shell, "-c", scm)

					etsk = &RawTask{
						Time:      ftmst,
						Type:      ftype,
						Path:      fpath,
						Lock:      flock,
						Command:   fcomm,
						Threads:   fthreads,
						Timeout:   ftout,
						Ttltime:   fttl,
						Interval:  fint,
						Repeaterr: frerr,
						Repeatcnt: frcnt,
						Stdcode:   stdcode,
						Stdout:    stdout,
						Errcode:   errcode,
						Stderr:    "",
						Runtime:   float64(0),
					}

					err = penc.Encode(etsk)
					if err != nil {
						appLogger.Errorf("| Virtual Host [%s] | Gob task encode error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
						return
					}

					err = NDBInsert(cldb, wvbucket, bkey, pbuffer.Bytes(), 0)
					if err != nil {
						appLogger.Errorf("| Virtual Host [%s] | Insert working task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
						return
					}

					var rtime float64

					cmm.Stdout = &cmmout
					cmm.Stderr = &cmmerr

					cwg := waitgroup.NewWaitGroup(1)

					crun := make(chan bool)
					kill := make(chan bool)
					kwait := make(chan bool)

					cmkey := vhost + ":" + skey

					stime := time.Now()

					time.Sleep(time.Duration(int(fint)*(vtscnt-1)) * time.Second)

					if keymutex.TryLock(cmkey) {

						cwg.Add(func() {

							err = cmm.Start()
							if err != nil {
								errcode = 255
								stderr = err.Error()
								appLogger.Errorf("| Virtual Host [%s] | Start command error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, scm, err)
							}

							err = cmm.Wait()
							if err != nil {
								errcode = 1
								stderr = err.Error()
								appLogger.Errorf("| Virtual Host [%s] | Execute command error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, scm, err)
							}

							crun <- true

						})

					} else {

						errcode = 255
						stderr = "Timeout mmutex lock error"
						appLogger.Errorf("| Virtual Host [%s] | Timeout mmutex lock error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, scm, err)

					}

					kwg := waitgroup.NewWaitGroup(1)

					kwg.Add(func() {

						kwait <- true

						for {

							if !keymutex.IsLock(cmkey) {
								kill <- true
								break
							}

							time.Sleep(time.Duration(5) * time.Millisecond)

						}

					})

					<-kwait
					close(kwait)

					cmmt := time.After(time.Duration(int(ftout)) * time.Second)

				Kill:

					for {

						select {

						case <-crun:

							keymutex.UnLock(cmkey)
							<-kill
							break Kill

						case <-kill:

							_ = cmm.Process.Kill()
							<-crun
							break Kill

						case <-cmmt:

							_ = cmm.Process.Kill()
							<-crun
							keymutex.UnLock(cmkey)
							<-kill
							errcode = 124
							break Kill

						default:

							time.Sleep(time.Duration(5) * time.Millisecond)

						}

					}

					cwg.Wait()
					close(crun)

					if keymutex.IsLock(cmkey) {
						keymutex.UnLock(cmkey)
					}

					kwg.Wait()
					close(kill)

					rtime = float64(time.Since(stime)) / float64(time.Millisecond)

					stdout = cmmout.String()
					stderr = cmmerr.String()

					vrrcnt := 0
					vrrbool := false

					for _, rgxrpt := range frerr {

						rreg, err := regexp.Compile(rgxrpt)
						if err != nil {
							continue
						}

						mchvrr := rreg.MatchString(stderr)

						if mchvrr {

							if vrrcnt == 0 {

								vrrbool = true

								rcnt.Lock()
								rcnt.trycounter[skey]++
								rcnt.Unlock()

								vrrcnt++

							}

						}

					}

					ebuffer := new(bytes.Buffer)
					eenc := gob.NewEncoder(ebuffer)

					etsk = &RawTask{
						Time:      ftmst,
						Type:      ftype,
						Path:      fpath,
						Lock:      flock,
						Command:   fcomm,
						Threads:   fthreads,
						Timeout:   ftout,
						Ttltime:   fttl,
						Interval:  fint,
						Repeaterr: frerr,
						Repeatcnt: frcnt,
						Stdcode:   stdcode,
						Stdout:    stdout,
						Errcode:   errcode,
						Stderr:    stderr,
						Runtime:   rtime,
					}

					err = eenc.Encode(etsk)
					if err != nil {
						appLogger.Errorf("| Virtual Host [%s] | Gob task encode error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
						return
					}

					rcnt.RLock()
					vrrc := rcnt.trycounter[skey]
					rcnt.RUnlock()

					if vrrbool && vrrc > 0 && vrrc <= int(frcnt) {

						err = NDBDelete(cldb, wvbucket, bkey)
						if err != nil {
							appLogger.Errorf("| Virtual Host [%s] | Delete working task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							return
						}

						return

					}

					rcnt.Lock()
					_, ok := rcnt.trycounter[skey]
					if ok {
						delete(rcnt.trycounter, skey)
					}
					rcnt.Unlock()

					err = NDBDelete(cldb, rvbucket, bkey)
					if err != nil {
						appLogger.Errorf("| Virtual Host [%s] | Delete received task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
						return
					}

					err = NDBDelete(cldb, wvbucket, bkey)
					if err != nil {
						appLogger.Errorf("| Virtual Host [%s] | Delete working task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
						return
					}

					err = NDBInsert(cldb, fvbucket, bkey, ebuffer.Bytes(), fttl)
					if err != nil {
						appLogger.Errorf("| Virtual Host [%s] | Insert completed task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
						return
					}

				})

				<-qwait
				close(qwait)

			}

			qwg.Wait()

		})

		<-vwait
		close(vwait)

	}

	vwg.Wait()

}
