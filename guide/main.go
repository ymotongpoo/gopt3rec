package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"time"

	"github.com/gorilla/feeds"
	xmlpath "gopkg.in/xmlpath.v2"
)

const (
	baseURL = "http://tv.so-net.ne.jp"
	tvURL   = baseURL + "/chart/23.action"
	bsURL   = baseURL + "/chart/bs1.action"
)

var (
	// in chart page
	schedulePath = xmlpath.MustCompile(`//a[@class="schedule-link"]`)
	linkPath     = xmlpath.MustCompile(`./@href`)

	// in schedule page
	infoPath = xmlpath.MustCompile(`//dl[@class="basicTxt"]/dd`)
	textPath = xmlpath.MustCompile(`./text()`)

	// schedule handling
	dateLayout      = "200601021504"
	durationPattern = regexp.MustCompile(`\d+`)
	interval        = 3 * time.Second
	maxOffsetIndex  = 33
	idSet           = make(map[string]bool)
)

// Program stores information of one tv program extracted from tv prgram detail page.
// eg. http://tv.so-net.ne.jp/schedule/101048201511291800.action
type Program struct {
	ChannelID int           `json:"channel_id"`
	Title     string        `json:"title"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Link      string        `json:"link"`
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

	date, err := time.Parse(dateLayout, startd)
	if err != nil {
		return nil, err
	}
	return &Program{
		ChannelID: cid,
		Title:     title,
		StartTime: date,
		EndTime:   date.Add(time.Duration(dur) * time.Minute),
		Duration:  time.Duration(dur) * time.Minute,
		Link:      uri,
	}, nil
}

func offsetParam(hours int) string {
	now := time.Now()
	offset := now.Add(time.Duration(hours) * time.Hour)
	return "head=" + offset.Format(dateLayout)
}

func main() {
	chartURLs := make(chan string)
	go func() {
		for i := 0; i < maxOffsetIndex; i++ {
			param := offsetParam(i * 5)
			chartURLs <- tvURL + "?" + param
			chartURLs <- bsURL + "?" + param
		}
	}()
	programURLs := RenderChart(chartURLs)
	programs := ProgramInfo(programURLs)
	feed := generateFeed(programs)
	file, err := os.Create("feed.atom")
	if err != nil {
		log.Fatal(err)
	}
	err = feed.WriteAtom(file)
	if err != nil {
		log.Fatal(err)
	}
}

func RenderChart(links <-chan string) chan string {
	programURLs := make(chan string, 100)
	go func() {
		for l := range links {
			time.Sleep(interval)
			resp, err := http.Get(l)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()
			root, err := xmlpath.ParseHTML(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			schedules := schedulePath.Iter(root)
			for schedules.Next() {
				n := schedules.Node()
				link := linkPath.Iter(n)
				link.Next()
				programURLs <- link.Node().String()
			}
		}
		close(programURLs)
	}()
	return programURLs
}

func ProgramInfo(links <-chan string) chan *Program {
	programs := make(chan *Program, 100)
	go func() {
		for l := range links {
			uri := baseURL + l
			id := path.Base(uri)
			if idSet[id] {
				continue
			} else {
				idSet[id] = true
			}
			time.Sleep(interval)
			p, err := NewProgram(uri)
			if err != nil {
				log.Println(err)
				continue
			}
			programs <- p
			time.Sleep(2 * time.Second)
		}
		close(programs)
	}()
	return programs
}

func generateFeed(programs <-chan *Program) *feeds.Feed {
	feed := &feeds.Feed{
		Title:       "Gガイド",
		Link:        &feeds.Link{Href: "https://tv.so-net.ne.jp/"},
		Description: "GガイドのRSS",
		Author:      &feeds.Author{"Anonymous", "john.doe@example.com"},
	}
	i := 0
	items := make([]*feeds.Item, 10000)
	for p := range programs {
		items[i] = &feeds.Item{
			Title:       p.Title,
			Link:        &feeds.Link{Href: p.Link},
			Description: fmt.Sprintf("%v - %v (%v)", p.StartTime, p.EndTime, p.Duration),
			Author:      &feeds.Author{strconv.Itoa(p.ChannelID), strconv.Itoa(p.ChannelID) + "@example.com"},
			Created:     p.StartTime,
		}
		i++
		if i > 10000 {
			break
		}
	}
	return feed
}
