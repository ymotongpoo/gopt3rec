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
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/ymotongpoo/gopt3rec/epg"

	"github.com/mattn/go-sqlite3"
)

const (
	EPGSelectStatement  = `select id, channel, title, detail, start, end, duration from epg where start >= ? and end < ?`
	SelectDefaultWindow = 2 * 24 * time.Hour // 1 week
	ResultBufSize       = 100
)

var (
	db *sql.DB
)

func init() {
	var err error
	db, err = sql.Open("sqlite3", "./epg.db")
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	defer db.Close()
	http.HandleFunc("/epg/v1/list", epgListHandler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func epgListHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	weekLater := now.Add(SelectDefaultWindow)
	nowStr := now.Format(sqlite3.SQLiteTimestampFormats[2])
	weekLaterStr := weekLater.Format(sqlite3.SQLiteTimestampFormats[2])
	rows, err := db.Query(EPGSelectStatement, nowStr, weekLaterStr)
	if err != nil {
		log.Println(err)
		return
	}
	defer rows.Close()

	programs := []epg.Program{}
	for rows.Next() {
		var p epg.Program
		var duration int
		var start, end string
		err = rows.Scan(&p.EventID, &p.Channel, &p.Title, &p.Detail, &start, &end, &duration)
		if err != nil {
			log.Println(err)
			continue
		}
		if p.Start, err = time.Parse(sqlite3.SQLiteTimestampFormats[2], start); err != nil {
			log.Println(err)
			continue
		}
		if p.End, err = time.Parse(sqlite3.SQLiteTimestampFormats[2], end); err != nil {
			log.Println(err)
			continue
		}
		p.Duration = time.Duration(duration)
		programs = append(programs, p)
	}
	e := json.NewEncoder(w)
	err = e.Encode(programs)
	if err != nil {
		log.Println(err)
	}
}
