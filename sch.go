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
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/eltaline/counter"
	"github.com/eltaline/mmutex"
	"github.com/pieterclaerhout/go-waitgroup"
	"github.com/xujiajun/nutsdb"
	"math/rand"
	"os/exec"
	"regexp"
	"sync"
	"syscall"
	"time"
)

// CtrlScheduler : Control threads scheduler
func CtrlScheduler(cldb *nutsdb.DB, keymutex *mmutex.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	// Wait Group

	wg.Add(1)

	// Shutdown

	if shutdown {
		return
	}

	// Variables

	serr := errors.New("shutdown operation")
	kerr := errors.New("signal: killed")

	bprefix := []byte("t:")

	// Throttling

	vhc := 0

	for range config.Server {
		vhc++
	}

	vwg := waitgroup.NewWaitGroup(vhc)

	for _, Server := range config.Server {

		if shutdown {
			continue
		}

		vwait := make(chan bool)

		vwg.Add(func() {

			vhost := Server.HOST
			shell := Server.SHELL

			rvbucket := "recv" + "_" + vhost + ":"
			wvbucket := "work" + "_" + vhost + ":"
			fvbucket := "comp" + "_" + vhost + ":"

			vwait <- true

			if keymutex.IsLock(rvbucket) {
				return
			}

			if keymutex.TryLock(rvbucket) {

				defer func() { keymutex.UnLock(rvbucket) }()

				mcompare := make(map[string]bool)

				errempty := errors.New("bucket is empty")
				errscans := errors.New("prefix scans not found")

				var ftsk []FullTask
				var f FullTask
				etsk := &RawTask{}

				// Loggers

				appLogger, applogfile := AppLogger()
				defer applogfile.Close()

				cerr := cldb.View(func(tx *nutsdb.Tx) error {

					var err error

					var received nutsdb.Entries
					var working nutsdb.Entries
					var inworking nutsdb.Entries
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

						inworking, _, err = tx.PrefixScan(wvbucket, rkey, -1, 1)

						if inworking != nil {
							continue
						}

						if err != nil && err.Error() != errscans.Error() {
							return err
						}

						completed, _, err = tx.PrefixScan(fvbucket, rkey, -1, 1)

						if completed != nil {
							continue
						}

						if err != nil && err.Error() != errscans.Error() {
							return err
						}

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
						f.Interr = rv.Interr
						f.Intcnt = rv.Intcnt
						f.Lookout = rv.Lookout
						f.Replace = rv.Replace
						f.Stdcode = rv.Stdcode
						f.Stdout = rv.Stdout
						f.Errcode = rv.Errcode
						f.Stderr = rv.Stderr
						f.Runtime = rv.Runtime

						ftsk = append(ftsk, f)

					}

					rand.Seed(time.Now().UnixNano())
					rand.Shuffle(len(ftsk), func(i, j int) { ftsk[i], ftsk[j] = ftsk[j], ftsk[i] })

					/*

					for i := 1; i < len(ftsk); i++ {

						r := rand.Intn(i + 1)

						if i != r {
							ftsk[r], ftsk[i] = ftsk[i], ftsk[r]
						}

					}

					*/

					return nil

				})

				if cerr != nil {
					appLogger.Errorf("| Virtual Host [%s] | Work with db error | %v", vhost, cerr)
					return
				}

				mpmap := make(map[string]int)

				qwg := waitgroup.NewWaitGroup(128)

				go func() {

					time.Sleep(30 * time.Second)

					for {

						if shutdown {
							return
						}

						c := 0

						for _, mp := range mpmap {
							c = c + mp
						}

						q := qwg.PendingCount()

						if c > 16 && q < 4 {
							keymutex.UnLock(rvbucket)
							break
						}

						time.Sleep(250 * time.Millisecond)

					}

				}()

				for _, task := range ftsk {

					if shutdown {
						continue
					}

					prefbkey := task.Key
					prefskey := string(task.Key)
					preftmst := time.Now().Unix()
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
					prefierr := task.Interr
					preficnt := task.Intcnt
					prefsout := task.Lookout
					prefrepl := task.Replace

					pretthr := vhost + ":" + preftype

					GlbMap.RLock()
					cntthr, ok := GlbMap.Glb[pretthr]
					GlbMap.RUnlock()

					if !ok {

						GlbMap.Lock()

						cntthr, ok = GlbMap.Glb[pretthr]
						if !ok {

							GlbMap.Glb[pretthr] = new(Cnts)
							cntthr = GlbMap.Glb[pretthr]
							cntthr.Cnt = counter.NewInt64()

						}

						GlbMap.Unlock()

					}

					if cntthr.Cnt.Get() >= int64(prefthreads) {

						mpc, _ := mpmap[pretthr]

						if mpc > 16 {
							continue
						}

						mpmap[pretthr]++

					}

					qwait := make(chan bool)

					qwg.Add(func() {

						var err error

						rand.Seed(time.Now().UnixNano())

						intthr := pretthr

						GlbMap.RLock()
						qcntthr, _ := GlbMap.Glb[intthr]
						GlbMap.RUnlock()

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
						fierr := prefierr
						ficnt := preficnt
						fsout := prefsout
						frepl := prefrepl

						qwait <- true

						if qcntthr.Cnt.Get() >= int64(fthreads) {

							for {

								if qcntthr.Cnt.Get() < int64(fthreads) {
									break
								}

								time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

							}

						}

						qcntthr.Cnt.Incr()

						vtscnt := qcntthr.Cnt.Get()

						stdcode := 0
						errcode := 0

						stdout := ""
						stderr := ""

						lookout := fsout

						var ig Ereg
						var rg Ereg

						var ireg []Ereg
						var rreg []Ereg

						for _, rgxint := range fierr {

							irgx, err := regexp.Compile(rgxint)
							if err != nil {
								appLogger.Errorf("| Virtual Host [%s] | Bad pattern in intercept errors array | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | Regular [%s] | %v", vhost, skey, fpath, flock, fcomm, rgxint, err)
								continue
							}

							ig.Str = rgxint
							ig.Rgx = irgx

							ireg = append(ireg, ig)

						}

						for _, rgxrpt := range frerr {

							rrgx, err := regexp.Compile(rgxrpt)
							if err != nil {
								appLogger.Errorf("| Virtual Host [%s] | Bad pattern in repeat errors array | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | Regular [%s] | %v", vhost, skey, fpath, flock, fcomm, rgxrpt, err)
								continue
							}

							rg.Str = rgxrpt
							rg.Rgx = rrgx

							rreg = append(rreg, rg)

						}

						lenireg := len(ireg)
						lenrreg := len(rreg)

						pbuffer := new(bytes.Buffer)
						penc := gob.NewEncoder(pbuffer)

						if !DirExists(fpath) {

							stderr = fmt.Sprintf("Path [%s] | directory from json field path does not exists", fpath)

							errcode = 1

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
								Interr:    fierr,
								Intcnt:    ficnt,
								Lookout:   fsout,
								Replace:   frepl,
								Stdcode:   stdcode,
								Stdout:    stdout,
								Errcode:   errcode,
								Stderr:    stderr,
								Runtime:   float64(0),
							}

							err = penc.Encode(etsk)
							if err != nil {
								qcntthr.Cnt.Decr()
								appLogger.Errorf("| Virtual Host [%s] | Gob task encode error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
								return
							}

							err = NDBDelete(cldb, rvbucket, bkey)
							if err != nil {
								qcntthr.Cnt.Decr()
								appLogger.Errorf("| Virtual Host [%s] | Delete received task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
								return
							}

							err = NDBInsert(cldb, fvbucket, bkey, pbuffer.Bytes(), 0)
							if err != nil {
								qcntthr.Cnt.Decr()
								appLogger.Errorf("| Virtual Host [%s] | Insert completed task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
								return
							}

							qcntthr.Cnt.Decr()
							return

						}

						scm := shell + " -c " + "\"cd " + fpath + " && " + fcomm + "\""
						cmm := exec.Command(shell, "-c", scm)
						cmm.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

						pstdout, err := cmm.StdoutPipe()
						if err != nil {
							qcntthr.Cnt.Decr()
							appLogger.Errorf("| Virtual Host [%s] | Can`t attach to stdout pipe | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							return
						}

						pstderr, err := cmm.StderrPipe()
						if err != nil {
							qcntthr.Cnt.Decr()
							appLogger.Errorf("| Virtual Host [%s] | Can`t attach to stderr pipe | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							return
						}

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
							Interr:    fierr,
							Intcnt:    ficnt,
							Lookout:   fsout,
							Replace:   frepl,
							Stdcode:   stdcode,
							Stdout:    stdout,
							Errcode:   errcode,
							Stderr:    "",
							Runtime:   float64(0),
						}

						err = penc.Encode(etsk)
						if err != nil {
							qcntthr.Cnt.Decr()
							appLogger.Errorf("| Virtual Host [%s] | Gob task encode error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							return
						}

						err = NDBInsert(cldb, wvbucket, bkey, pbuffer.Bytes(), 0)
						if err != nil {
							qcntthr.Cnt.Decr()
							appLogger.Errorf("| Virtual Host [%s] | Insert working task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							return
						}

						var rtime float64

						cwg := waitgroup.NewWaitGroup(1)

						var intercept bool

						imsgout := ""
						imsgerr := ""

						crun := make(chan bool)
						kill := make(chan bool)
						kwait := make(chan bool)
						owait := make(chan bool)
						ewait := make(chan bool)

						ssync := make(chan bool)

						cmkey := vhost + ":" + skey

						stime := time.Now()

						time.Sleep(time.Duration(int(fint)*(int(vtscnt)-1)) * time.Second)

						killed := false

						if keymutex.TryLock(cmkey) {

							cwg.Add(func() {

								<-ssync
								close(ssync)

								var err error

								err = cmm.Start()
								if err != nil {

									if exitError, ok := err.(*exec.ExitError); ok {
										errcode = exitError.ExitCode()
									} else {
										errcode = 255
									}

									imsgerr = err.Error()
									appLogger.Errorf("| Virtual Host [%s] | Start command error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, scm, err)

								}

								err = cmm.Wait()
								if err != nil {

									if exitError, ok := err.(*exec.ExitError); ok {
										errcode = exitError.ExitCode()
									} else {
										errcode = 1
									}

									if !shutdown {
										appLogger.Errorf("| Virtual Host [%s] | Execute command error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, scm, err)
									}

									if err.Error() == kerr.Error() {

										err = NDBDelete(cldb, wvbucket, bkey)
										if err != nil {
											appLogger.Errorf("| Virtual Host [%s] | Delete working task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
										}

									}

								}

								if !killed {
									crun <- true
								}

							})

						} else {

							errcode = 255
							imsgerr = "Timeout mmutex lock error"
							appLogger.Errorf("| Virtual Host [%s] | Timeout mmutex lock error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | Error [%s] | %v", vhost, skey, fpath, flock, scm, imsgerr, err)

							err = NDBDelete(cldb, wvbucket, bkey)
							if err != nil {
								qcntthr.Cnt.Decr()
								appLogger.Errorf("| Virtual Host [%s] | Delete working task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
								return
							}

							qcntthr.Cnt.Decr()
							return

						}

						kwg := waitgroup.NewWaitGroup(1)

						kwg.Add(func() {

							kwait <- true

							for {

								if !keymutex.IsLock(cmkey) || killed || shutdown {

									if !killed {
										kill <- true
									}

									break

								}

								time.Sleep(time.Duration(5) * time.Millisecond)

							}

						})

						<-kwait
						close(kwait)

						owg := waitgroup.NewWaitGroup(1)

						owg.Add(func() {

							scanner := bufio.NewScanner(pstdout)

							owait <- true

						Loop:

							for scanner.Scan() {

								msg := scanner.Text()

								imsgout = imsgout + msg + "\n"

								if lookout && lenireg > 0 {

									for _, rgx := range ireg {

										mchvir := rgx.Rgx.MatchString(msg)

										if mchvir && keymutex.IsLock(cmkey) && !killed {

											intercept = true
											kill <- true
											break Loop

										}

										if killed || shutdown {
											break Loop
										}

									}

								}

								if !keymutex.IsLock(cmkey) || killed || shutdown {
									break
								}

							}

						})

						<-owait
						close(owait)

						ewg := waitgroup.NewWaitGroup(1)

						ewg.Add(func() {

							scanner := bufio.NewScanner(pstderr)

							ewait <- true

						Loop:

							for scanner.Scan() {

								msg := scanner.Text()

								imsgerr = imsgerr + msg + "\n"

								if lenireg > 0 {

									for _, rgx := range ireg {

										mchvir := rgx.Rgx.MatchString(msg)

										if mchvir && keymutex.IsLock(cmkey) && !killed {

											intercept = true
											kill <- true
											break Loop

										}

										if killed || shutdown {
											break Loop
										}

									}

								}

								if !keymutex.IsLock(cmkey) || killed || shutdown {
									break
								}

							}

						})

						<-ewait
						close(ewait)

						ssync <- true

						cmmt := time.After(time.Duration(int(ftout)) * time.Second)

					Kill:

						for {

							select {

							case <-crun:

								killed = true
								break Kill

							case <-kill:

								if cmm.Process != nil {

									pgid, err := syscall.Getpgid(cmm.Process.Pid)
									if err == nil {

										zerr := syscall.Kill(-pgid, syscall.SIGKILL)
										if zerr != nil {
											appLogger.Errorf("| Virtual Host [%s] | Process kill error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, zerr)
										}

									}

								}

								killed = true
								break Kill

							case <-cmmt:

								if cmm.Process != nil {

									pgid, err := syscall.Getpgid(cmm.Process.Pid)
									if err == nil {

										zerr := syscall.Kill(-pgid, syscall.SIGKILL)
										if zerr != nil {
											appLogger.Errorf("| Virtual Host [%s] | Process kill error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, zerr)
										}

									}

								}

								killed = true
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
						owg.Wait()
						ewg.Wait()
						close(kill)

						if shutdown {
							qcntthr.Cnt.Decr()
							return
						}

						stdout = imsgout
						stderr = imsgerr

						rtime = float64(time.Since(stime)) / float64(time.Millisecond)

						vircnt := 0
						virbool := false

						vrrcnt := 0
						vrrbool := false

						if intercept && lenireg > 0 {

							iskey := "i:" + skey

							GlbMap.RLock()
							cnticnt, ok := GlbMap.Glb[iskey]
							GlbMap.RUnlock()

							if !ok {

								GlbMap.Lock()

								cnticnt, ok = GlbMap.Glb[iskey]
								if !ok {

									GlbMap.Glb[iskey] = new(Cnts)
									cnticnt = GlbMap.Glb[iskey]
									cnticnt.Cnt = counter.NewInt64()

								}

								GlbMap.Unlock()

							}

							for _, rgx := range ireg {

								var mchvio bool

								mchvir := rgx.Rgx.MatchString(stderr)

								if lookout {
									mchvio = rgx.Rgx.MatchString(stdout)
								}

								if mchvir || mchvio {

									if vircnt == 0 {

										virbool = true

										cnticnt.Cnt.Incr()

										vircnt++

									}

								}

							}

							virc := cnticnt.Cnt.Get()

							if virbool && int(virc) > 0 && int(virc) <= int(ficnt) {

								err = NDBDelete(cldb, wvbucket, bkey)
								if err != nil {
									qcntthr.Cnt.Decr()
									appLogger.Errorf("| Virtual Host [%s] | Delete working task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
									return
								}

								qcntthr.Cnt.Decr()
								return

							}

							GlbMap.Lock()
							cnticnt, ok = GlbMap.Glb[iskey]
							if ok {
								delete(GlbMap.Glb, iskey)
							}
							GlbMap.Unlock()

						}

						if lenrreg > 0 {

							rskey := "r:" + skey

							GlbMap.RLock()
							cntrcnt, ok := GlbMap.Glb[rskey]
							GlbMap.RUnlock()

							if !ok {

								GlbMap.Lock()

								cntrcnt, ok = GlbMap.Glb[rskey]
								if !ok {

									GlbMap.Glb[rskey] = new(Cnts)
									cntrcnt = GlbMap.Glb[rskey]
									cntrcnt.Cnt = counter.NewInt64()

								}

								GlbMap.Unlock()

							}

							for _, rgx := range rreg {

								var mchvoo bool

								mchvrr := rgx.Rgx.MatchString(stderr)

								if lookout {
									mchvoo = rgx.Rgx.MatchString(stdout)
								}

								if mchvrr || mchvoo {

									if vrrcnt == 0 {

										vrrbool = true

										cntrcnt.Cnt.Incr()

										vrrcnt++

									}

								}

							}

							vrrc := cntrcnt.Cnt.Incr()

							if vrrbool && int(vrrc) > 0 && int(vrrc) <= int(frcnt) {

								err = NDBDelete(cldb, wvbucket, bkey)
								if err != nil {
									qcntthr.Cnt.Decr()
									appLogger.Errorf("| Virtual Host [%s] | Delete working task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
									return
								}

								qcntthr.Cnt.Decr()
								return

							}

							GlbMap.Lock()
							cntrcnt, ok = GlbMap.Glb[rskey]
							if ok {
								delete(GlbMap.Glb, rskey)
							}
							GlbMap.Unlock()

						}

						ebuffer := new(bytes.Buffer)
						eenc := gob.NewEncoder(ebuffer)

						ftmst = time.Now().Unix()

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
							Interr:    fierr,
							Intcnt:    ficnt,
							Lookout:   fsout,
							Replace:   frepl,
							Stdcode:   stdcode,
							Stdout:    stdout,
							Errcode:   errcode,
							Stderr:    stderr,
							Runtime:   rtime,
						}

						err = eenc.Encode(etsk)
						if err != nil {
							appLogger.Errorf("| Virtual Host [%s] | Gob task encode error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							qcntthr.Cnt.Decr()
							return
						}

						err = NDBDelete(cldb, rvbucket, bkey)
						if err != nil {
							appLogger.Errorf("| Virtual Host [%s] | Delete received task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							qcntthr.Cnt.Decr()
							return
						}

						err = NDBDelete(cldb, wvbucket, bkey)
						if err != nil {
							appLogger.Errorf("| Virtual Host [%s] | Delete working task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							qcntthr.Cnt.Decr()
							return
						}

						err = NDBInsert(cldb, fvbucket, bkey, ebuffer.Bytes(), fttl)
						if err != nil {
							appLogger.Errorf("| Virtual Host [%s] | Insert completed task db error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | %v", vhost, skey, fpath, flock, fcomm, err)
							qcntthr.Cnt.Decr()
							return
						}

						if errcode != 0 {
							appLogger.Errorf("| Virtual Host [%s] | Task completed with error | Key [%s] | Path [%s] | Lock [%s] | Command [%s] | Error [%s] | exit status %d", vhost, skey, fpath, flock, fcomm, stderr, errcode)
						}

						qcntthr.Cnt.Decr()

					})

					<-qwait
					close(qwait)

					time.Sleep(250 * time.Millisecond)

				}

				qwg.Wait()

			}

		})

		<-vwait
		close(vwait)

	}

	vwg.Wait()

}
