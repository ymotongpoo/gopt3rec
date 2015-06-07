// Copyright 2015 Yoshi Yamaguchi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// 	You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// 	See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"database/sql"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

//var BatchDumpDuration = strconv.Itoa(1 * 60)
var BatchDumpDuration = strconv.Itoa(200)

func BatchDump(path string) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		log.Fatalf("Path doesn't exist: %v", path)
	}

	db, err := sql.Open("sqlite3", filepath.Join(path, "epg.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tsCh := make(chan string, len(TVChannelMap))
	jsonCh := make(chan string, len(TVChannelMap))
	go batchRec(path, tsCh)
	go batchDump(tsCh, jsonCh)

	for jsonfile := range jsonCh {
		err := epgStore(db, jsonfile)
		if err != nil {
			log.Println(jsonfile, err)
		}
	}
}

func batchRec(path string, tsCh chan<- string) {
	for _, v := range TVChannelMap {
		tsfile := filepath.Join(path, v+".ts")
		cmdStr := []string{"recpt1", "--b25", "--strip", v, BatchDumpDuration, tsfile}
		cmd := exec.Command(cmdStr[0], cmdStr[1:]...)
		err := cmd.Run()
		if err != nil {
			log.Println("failed to record: %v", v)
			continue
		}
		tsCh <- tsfile
	}
	close(tsCh)
}

func batchDump(tsCh <-chan string, jsonCh chan<- string) {
	for tsfile := range tsCh {
		jsonfile := tsfile + ".json"
		cmdStr := []string{"epgdump", "json", tsfile, jsonfile}
		cmd := exec.Command(cmdStr[0], cmdStr[1:]...)
		err := cmd.Run()
		if err != nil {
			continue
		}
		jsonCh <- jsonfile
	}
	close(jsonCh)
}
