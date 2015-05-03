package main

import (
	"bytes"
	"flag"
	_ "fmt"
	"io"
	"log"
	_ "os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	FilePrefixFormat = "20060102T1504"
	AtCmdFormat      = "01021504.05"
)

var (
	tv    = flag.String("tv", "", "TV channel to record in remote control ID.")
	min   = flag.Int("min", 60, "minites to record")
	start = flag.String("start", "", "recording start time in format like XX:YY")
	title = flag.String("title", "test", "tv program title")
)

var TVChannelMap = map[string]string{
	"1":    "27",
	"nhk":  "27",
	"2":    "26",
	"etv":  "26",
	"4":    "25",
	"ntv":  "25",
	"5":    "24",
	"ex":   "24",
	"6":    "22",
	"tbs":  "22",
	"7":    "23",
	"tx":   "23",
	"8":    "21",
	"cx":   "21",
	"9":    "16",
	"mx":   "16",
	"12":   "28",
	"univ": "28",
}

func init() {
	now := time.Now().Add(10 * time.Second)
	*start = now.Format(AtCmdFormat)
}

func main() {
	flag.Parse()
	var v string
	var ok bool
	if v, ok = TVChannelMap[*tv]; !ok {
		log.Fatalf("specified channel doesn't exist: %v", v)
	}
	duration := strconv.Itoa(*min * 60)
	filename := time.Now().Format(FilePrefixFormat) + "-" + *title + ".ts"
	recpt1Str := []string{"recpt1", "--b25", "--strip", v, duration, filename}
	recpt1Cmd := exec.Command("echo", recpt1Str...)
	atCmd := exec.Command("at", "-t", *start)

	r, w := io.Pipe()
	recpt1Cmd.Stdout = w
	atCmd.Stdin = r

	var stdout, stderr bytes.Buffer
	atCmd.Stdout = &stdout
	atCmd.Stderr = &stderr
	err := recpt1Cmd.Start()
	if err != nil {
		log.Fatalf("%v\n%v\n%v", err, strings.Join(recpt1Str, " "), stderr.String())
	}
	err = atCmd.Start()
	if err != nil {
		log.Fatalf("%v\n%v\n%v", stderr.String())
	}
	recpt1Cmd.Wait()
	w.Close()
	atCmd.Wait()
	log.Printf("booked %v: %v", *start, stdout.String())
}
