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
	"errors"
	"github.com/eltaline/nutsdb"
	"sync"
)

// ResetWorking : Reset working queue
func ResetWorking(cldb *nutsdb.DB, wg *sync.WaitGroup) {
	defer wg.Done()

	var err error

	// Wait Group

	wg.Add(1)

	// Loggers

	appLogger, applogfile := AppLogger()
	defer applogfile.Close()

	// Shutdown

	if shutdown {
		return
	}

	// Variables

	errempty := errors.New("bucket is empty")

	var rstt []ResetTask
	var r ResetTask

	wvbucket := "work"

	cerr := cldb.View(func(tx *nutsdb.Tx) error {

		var err error

		var tasks nutsdb.Entries

		tasks, err = tx.GetAll(wvbucket)

		if tasks == nil {
			return nil
		}

		if err != nil && err.Error() == errempty.Error() {
			return nil
		}

		if err != nil {
			return err
		}

		for _, rtask := range tasks {

			r.Key = rtask.Key
			rstt = append(rstt, r)

		}

		return nil

	})

	if len(rstt) == 0 {
		return
	}

	if cerr != nil {
		appLogger.Errorf("| Work with db error | %v", cerr)
		return
	}

	for _, task := range rstt {

		skey := string(task.Key)

		err = NDBDelete(cldb, wvbucket, task.Key)
		if err != nil {
			appLogger.Errorf("| Initial reset task from db error | Key [%s] | %v", skey, err)
		}

	}

}
