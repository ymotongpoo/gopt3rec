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
package epg

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type EPGData struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	OritinalNetworkID int       `json:"original_network_id"`
	ServiceID         int       `json:"service_id"`
	TransportStreamID int       `json:"transport_stream_id"`
	Programs          []Program `json:"programs"`
}

type Program struct {
	EventID    int           `json:"event_id"`
	Channel    string        `json:"channel"`
	Title      string        `json:"title"`
	Detail     string        `json:"detail"`
	StartUnix  int64         `json:"start,omitempty"`
	EndUnix    int64         `json:"end,omitempty"`
	Start      time.Time     `json:"starttime"`
	End        time.Time     `json:"endtime"`
	Category   []Category    `json:"category,omitempty"`
	AttachInfo []interface{} `json:"attachinfo,omitempty"` // TODO(ymotongpoo): confirm contents
	FreeCA     bool          `json:"freeCA,omitempty"`
	Video      Video         `json:"video,omitempty"`
	Duration   time.Duration `json:"duration,omitempty"`
	Audio      []Audio       `json:"audio,omitempty"`
	ExtDetails []ExtDetail   `json:"extdetail,omitempty"`
}

type Category struct {
	Large  CategoryLabels `json:"large"`
	Middle CategoryLabels `json:"middle"`
}

type CategoryLabels struct {
	Japanese string `json:"ja_JP"`
	English  string `json:"en"`
}

type Video struct {
	Resolution string `json:"resolution"`
	Aspect     string `json:"aspect"`
}

type Audio struct {
	LangCode string `json:"langcode"`
	Type     string `json:"type"`
	ExtDesc  string `json:"extdesc"`
}

type ExtDetail struct {
	ItemDescription string `json:"item_description"`
	Item            string `json:"item"`
}

func New(r io.Reader) ([]EPGData, error) {
	decoder := json.NewDecoder(r)
	data := []EPGData{}
	err := decoder.Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("epg.New: %v", err)
	}

	for i := range data {
		for j := range data[i].Programs {
			startUnix := data[i].Programs[j].StartUnix
			endUnix := data[i].Programs[j].EndUnix
			loc, _ := time.LoadLocation("Asia/Tokyo")
			data[i].Programs[j].Start = time.Unix(startUnix/10000, 0).In(loc)
			data[i].Programs[j].End = time.Unix(endUnix/10000, 0).In(loc)
		}
	}
	return data, nil
}
