package main

import (
	_ "fmt"
	_ "os"
	"os/exec"
	"flag"
	"time"
	"strconv"
)

const (
	FilePrefixFormat = "20060102T1504"
	AtCmdFormat = "15:04"
)

var (
	tv := flag.String("tv", "", "TV channel to record in remote control ID.")
	min := flag.Int("min", 60, "minites to record")
	start := flag.String("start", "", "recording start time in format like XX:YY")
	title := flag.String("title", "test", "tv program title")
)

func init() {
	now := time.Now()
	start = now.Format(AtCmdFormat)
}

func main() {
	flag.Parse()
	duration := strconv.Itoa(min*60)
	filename := time.Now().Format(FilePrefixFormat) + "-" + title + ".ts"
	cmd := exec.Command("at", start, "<", "recpt1", "--b25", tv, duration, filename)
}

