package main

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"time"

	xmlpath "gopkg.in/xmlpath.v2"
)

const (
	baseURL  = "http://tv.so-net.ne.jp/"
	chartURL = baseURL + "chart/23.action"
)

var (
	// in chart page
	schedulePath = xmlpath.MustCompile(`//a[@class="schedule-link"]`)
	linkPath     = xmlpath.MustCompile(`./@href`)

	// in schedule page
	infoPath = xmlpath.MustCompile(`//dl[@class="basicTxt"]/dd`)
	textPath = xmlpath.MustCompile(`./text()`)

	// schedule handling
	durationPattern = regexp.MustCompile(`\d+`)
)

// Program stores information of one tv program extracted from tv prgram detail page.
// eg. http://tv.so-net.ne.jp/schedule/101048201511291800.action
type Program struct {
	ChannelID int           `json:"channel_id"`
	Title     string        `json:"title"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
}

// NewProgram is a constructor of Program. uri should be URL of the tv program page.
func NewProgram(uri string) (*Program, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}

	// parse HTML data.
	root, err := xmlpath.ParseHTML(resp.Body)
	if err != nil {
		return nil, err
	}
	infos := infoPath.Iter(root)
	infos.Next()
	titleNode := textPath.Iter(infos.Node())
	titleNode.Next()
	title := titleNode.Node().String()
	infos.Next()
	scheduleNode := textPath.Iter(infos.Node())
	scheduleNode.Next()
	scheduleDesc := durationPattern.FindAllStringSubmatch(scheduleNode.Node().String(), -1)
	durn := scheduleDesc[len(scheduleDesc)-1][0]
	dur, err := strconv.Atoi(durn)
	if err != nil {
		return nil, err
	}

	// parse info in URL
	urlid := path.Base(uri)[0:18]
	cidd := urlid[2:6]
	startd := urlid[6:]

	cid, err := strconv.Atoi(cidd)
	if err != nil {
		return nil, err
	}

	date, err := time.Parse("200601021504", startd)
	if err != nil {
		return nil, err
	}
	return &Program{
		ChannelID: cid,
		Title:     title,
		StartTime: date,
		EndTime:   date.Add(time.Duration(dur) * time.Minute),
		Duration:  time.Duration(dur) * time.Minute,
	}, nil
}

func main() {
	resp, err := http.Get(chartURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	root, err := xmlpath.ParseHTML(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	schedules := schedulePath.Iter(root)

	programURLs := make(chan string, 100)
	go func() {
		for schedules.Next() {
			n := schedules.Node()
			link := linkPath.Iter(n)
			link.Next()
			programURLs <- link.Node().String()
		}
		close(programURLs)
	}()
	time.Sleep(3 * time.Second)
	ProgramInfo(programURLs)
}

func ProgramInfo(links <-chan string) {
	num := 0
	for l := range links {
		p, err := NewProgram(baseURL + l)
		if err != nil {
			log.Println(err)
			continue
		}
		fmt.Println(p.ChannelID, p.Title, p.StartTime, p.Duration)
		time.Sleep(2 * time.Second)
		num++
	}
}
