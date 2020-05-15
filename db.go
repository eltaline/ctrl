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
	"github.com/eltaline/nutsdb"
	"io/ioutil"
)

// DB Helpers

// NDBInsert : NutsDB insert key function
func NDBInsert(db *nutsdb.DB, bucket string, key []byte, value []byte, ttl uint32) error {

	nerr := db.Update(func(tx *nutsdb.Tx) error {

		err := tx.Put(bucket, key, value, ttl)
		if err != nil {
			return err
		}

		return nil

	})

	if nerr != nil {
		return nerr
	}

	return nil

}

// NDBDelete : NutsDB delete key function
func NDBDelete(db *nutsdb.DB, bucket string, key []byte) error {

	nerr := db.Update(func(tx *nutsdb.Tx) error {

		err := tx.Delete(bucket, key)
		if err != nil {
			return err
		}

		return nil

	})

	if nerr != nil {
		return nerr
	}

	return nil

}

// NDBMerge : NutsDB merge compaction function
func NDBMerge(db *nutsdb.DB, dir string) error {

	var err error

	err = db.Merge()
	if err != nil {
		return err
	}

	segs, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, seg := range segs {

		sn := seg.Name()
		ss := seg.Size()

		fn := dir + "/" + sn

		emptyBuffer := make([]byte, ss)

		segmentBuffer, err := ioutil.ReadFile(fn)
		if err != nil {
			return err
		}

		if segmentBuffer != nil {

			if bytes.Equal(emptyBuffer, segmentBuffer) {
				err = RemoveSegment(fn)
				if err != nil {
					return err
				}
			}

		}

	}

	return nil

}
