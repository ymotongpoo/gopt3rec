package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
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
)

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
	}()
	time.Sleep(3 * time.Second)
	ProgramInfo(programURLs)
}

func ProgramInfo(links chan string) {
	num := 0
	for l := range links {
		resp, err := http.Get(baseURL + l)
		if err != nil {
			log.Println(err)
			continue
		}
		root, err := xmlpath.ParseHTML(resp.Body)
		if err != nil {
			log.Println(err)
			continue
		}
		infos := infoPath.Iter(root)
		infos.Next()
		titleNode := textPath.Iter(infos.Node())
		titleNode.Next()
		title := titleNode.Node().String()
		infos.Next()
		scheduleNode := textPath.Iter(infos.Node())
		scheduleNode.Next()
		schedule := strings.TrimSpace(scheduleNode.Node().String())
		fmt.Println(num, title, schedule)
		time.Sleep(2 * time.Second)
		num++
	}
}
