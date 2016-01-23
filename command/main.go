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
	"bytes"
	"database/sql"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ymotongpoo/gopt3rec/epg"
)

const (
	FilePrefixFormat   = "20060102T1504"
	HourMinFormat      = "1504"
	DateHourMinFormat  = "01021504"
	AtCmdFormat        = "0601021504.05"
	EPGInsertStatement = `replace into epg(id, channel, title, detail, start, end, duration) values (?, ?, ?, ?, ?, ?, ?)`
)

var TVChannelMap = map[string]string{
	"1":      "27",
	"nhk":    "27",
	"2":      "26",
	"etv":    "26",
	"4":      "25",
	"ntv":    "25",
	"5":      "24",
	"ex":     "24",
	"6":      "22",
	"tbs":    "22",
	"7":      "23",
	"tx":     "23",
	"8":      "21",
	"cx":     "21",
	"9":      "16",
	"mx":     "16",
	"12":     "28",
	"univ":   "28",
	"nhkbs1": "BS15_0",
	"nhkbs2": "BS15_1",
	"bsntv":  "BS13_0",
	"bsex":   "BS01_0",
	"bstbs":  "BS01_1",
	"bsj":    "BS03_1",
	"bsfuji": "BS13_1",
}

var replaceChars = map[string]string{
	"!": "！",
	"?": "？",
	"#": "＃",
	"(": "（",
	")": "）",
	" ": "_",
	"*": "＊",
	"/": "／",
}

var (
	now              time.Time
	defaultStartTime string
	defaultPrefix    string
)

// init sets default values of each variables which depends on execution time.
func init() {
	now = time.Now().Add(10 * time.Second)
	defaultStartTime = now.Format(AtCmdFormat)
	defaultPrefix = now.Format(FilePrefixFormat)
}

// parseBookSchedule converts specified time string to time.Time.
func parseBookSchedule(start string) (string, string, error) {
	if start == "" {
		return defaultStartTime, defaultPrefix, nil
	}

	loc, _ := time.LoadLocation("Asia/Tokyo")

	var err error
	var s time.Time
	var year, day int
	var month time.Month
	switch len(start) {
	case 4:
		s, err = time.Parse(HourMinFormat, start)
		if err != nil {
			return "", "", err
		}
		year, month, day = now.Date()
	case 8:
		s, err = time.Parse(DateHourMinFormat, start)
		if err != nil {
			return "", "", err
		}
		if s.Month() < now.Month() {
			year = now.Year() + 1
		} else {
			year = now.Year()
		}
		month = s.Month()
		day = s.Day()
	}
	t := time.Date(year, month, day, s.Hour(), s.Minute(), s.Second(), 0, loc)
	startTime := t.Format(AtCmdFormat)
	prefix := t.Format(FilePrefixFormat)
	return startTime, prefix, nil
}

// normalize replaces special characters in shell script to corresponding multi-byte characters.
func normalize(orig string) string {
	for k, v := range replaceChars {
		orig = strings.Replace(orig, k, v, -1)
	}
	return orig
}

// Book is helper command of recpt1. This accepts user friendly arguments to set recpt1 schedule with at command.
func Book(tv, start, title string, min int) {
	var v string
	var ok bool
	if v, ok = TVChannelMap[tv]; !ok {
		log.Fatalf("specified channel doesn't exist: %v", tv)
	}

	startTime, prefix, err := parseBookSchedule(start)
	if err != nil {
		log.Fatalf("Error on parsing start time: %v", err)
	}

	duration := strconv.Itoa(min * 60)
	title = strings.TrimSpace(title)
	title = normalize(title)
	filename := prefix + "-" + title + ".ts"
	recpt1Str := []string{"recpt1", "--b25", "--sid", "hd", "--strip", v, duration, filename}
	recpt1Cmd := exec.Command("echo", recpt1Str...)
	atCmd := exec.Command("at", "-t", startTime)

	r, w := io.Pipe()
	recpt1Cmd.Stdout = w
	atCmd.Stdin = r

	var stdout, stderr bytes.Buffer
	atCmd.Stdout = &stdout
	atCmd.Stderr = &stderr
	err = recpt1Cmd.Start()
	if err != nil {
		log.Fatalf("%v\n%v\n%v", err, strings.Join(recpt1Str, " "), stderr.String())
	}
	err = atCmd.Start()
	if err != nil {
		log.Fatalf("%v", stderr.String())
	}
	recpt1Cmd.Wait()
	w.Close()
	err = atCmd.Wait()
	if err != nil {
		log.Fatalf("%v", stderr.String())
	}
	log.Printf("booked %v (%v)", startTime, strings.Join(recpt1Str, " "))
	log.Printf("stdout: %v, stderr: %v", stdout.String(), stderr.String())
}

// EPGStore inserts egpdata into existing SQLite3 database table.
func epgStore(db *sql.DB, epgjson string) error {
	file, err := os.Open(epgjson)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := epg.New(file)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
	}
	statement, err := tx.Prepare(EPGInsertStatement)
	if err != nil {
		log.Println(err)
	}
	defer statement.Close()

	log.Printf("start: %v", epgjson)
	for _, d := range data {
		log.Printf("file: %v, %v programs", epgjson, len(d.Programs))
		for _, p := range d.Programs {
			_, err := statement.Exec(p.EventID, p.Channel, p.Title, p.Detail, p.Start, p.End, p.Duration)
			if err != nil {
				log.Println(err)
				continue
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Println(err)
	}
	return nil
}

func EPGStore(epgjson string) {
	db, err := sql.Open("sqlite3", "epg.db")
	if err != nil {
		log.Fatalf("EPGStore: %v", err)
	}
	defer db.Close()

	err = epgStore(db, epgjson)
	if err != nil {
		log.Fatalf("EPGStore: %v", err)
	}
}

func main() {
	// options for wrapper command of recpt1
	bookFlags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	var (
		tv    = bookFlags.String("tv", "", "TV channel to record in remote control ID.")
		min   = bookFlags.Int("min", 60, "minites to record")
		start = bookFlags.String("start", "", "recording start time in format like HHMM or mmddHHMM")
		title = bookFlags.String("title", "test", "tv program title")
	)

	// options for wrapper command for epgdump
	epgdumpFlags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	var (
		epgjson = epgdumpFlags.String("json", "", "")
	)

	sub := os.Args[1]
	switch sub {
	case "book":
		bookFlags.Parse(os.Args[2:])
		log.Printf("book: %v %v %v (%v min.)", *tv, *start, *title, *min)
		Book(*tv, *start, *title, *min)
	case "epgstore":
		epgdumpFlags.Parse(os.Args[2:])
		log.Printf("epgstore: %v", *epgjson)
		EPGStore(*epgjson)
	case "batchdump":
		log.Println("batchrec")
		var path string
		switch {
		case len(os.Args) > 3:
			path = os.Args[2]
		case os.Getenv("EPGDUMP_HOME") != "":
			path = os.Getenv("EPGDUMP_HOME")
		default:
			path = "epgdump"
		}
		BatchDump(path)
	}
}
