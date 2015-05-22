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
	AtCmdFormat        = "01021504.05"
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
	"nhkbs1": "BS15_1",
	"nhkbs2": "BS15_1",
	"bsntv":  "BS13_0",
	"bsex":   "BS01_0",
	"bstbs":  "BS01_1",
	"bsj":    "BS03_1",
	"bsfuji": "BS13_1",
}

var (
	now              time.Time
	defaultStartTime string
	defaultPrefix    string
)

func init() {
	now = time.Now().Add(10 * time.Second)
	defaultStartTime = now.Format(AtCmdFormat)
	defaultPrefix = now.Format(FilePrefixFormat)
}

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
		year = now.Year()
		month = s.Month()
		day = s.Day()
	}
	t := time.Date(year, month, day, s.Hour(), s.Minute(), s.Second(), 0, loc)
	startTime := t.Format(AtCmdFormat)
	prefix := t.Format(FilePrefixFormat)
	return startTime, prefix, nil
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
	log.Printf("start time: %v", startTime)

	duration := strconv.Itoa(min * 60)
	filename := prefix + "-" + title + ".ts"
	recpt1Str := []string{"recpt1", "--b25", "--strip", v, duration, filename}
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

// EPGDump inserts egpdata into existing SQLite3 database table.
func EPGDump(epgjson string) {
	file, err := os.Open(epgjson)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	data, err := epg.New(file)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite3", "epg.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	for _, d := range data {
		tx, err := db.Begin()
		if err != nil {
			log.Println(err)
			continue
		}
		statement, err := tx.Prepare(EPGInsertStatement)
		if err != nil {
			log.Println(err)
			continue
		}
		defer statement.Close()
		for _, p := range d.Programs {
			_, err := statement.Exec(p.EventID, p.Channel, p.Title, p.Detail, p.Start, p.End, p.Duration)
			if err != nil {
				log.Println(err)
				continue
			}
		}
		tx.Commit()
	}
}

func main() {
	bookFlags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	var (
		tv    = bookFlags.String("tv", "", "TV channel to record in remote control ID.")
		min   = bookFlags.Int("min", 60, "minites to record")
		start = bookFlags.String("start", "", "recording start time in format like HHMM or mmddHHMM")
		title = bookFlags.String("title", "test", "tv program title")
	)

	epgdumpFlags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	var (
		epgjson = epgdumpFlags.String("json", "", "")
	)

	sub := os.Args[1]
	switch sub {
	case "book":
		bookFlags.Parse(os.Args[2:])
		log.Printf("book: %v %v %v %v", *tv, *start, *title, *min)
		Book(*tv, *start, *title, *min)
	case "epgdump":
		epgdumpFlags.Parse(os.Args[2:])
		log.Printf("epgdump: %v", *epgjson)
		EPGDump(*epgjson)
	}
}
