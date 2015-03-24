package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/SlyMarbo/rss"
	"github.com/gilliek/go-opml/opml"
	"github.com/gorilla/feeds"
	"github.com/oleiade/lane"
)

const (
	opml_url = "https://raw.githubusercontent.com/yasuharu519/opml/master/main.opml"
)

type Feed struct {
	Title string
	URL   string
}

var wg sync.WaitGroup
var feed_list []Feed
var fetched_count int = 0
var transport = &http.Transport{
	TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true,
	},
}
var client = &http.Client{Transport: transport,
	Timeout: time.Duration(10) * time.Second,
}

type ItemHeap []*feeds.Item
var heap = &ItemHeap{}

func (self ItemHeap) Len() int {
	return len(self)
}

func (self ItemHeap) Less(i, j int) bool {
	return self[i].Updated.After(self[j].Updated)
}

func (self ItemHeap) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self *ItemHeap) Push(x interface{}) {
	item := x.(*feeds.Item)
	*self = append(*self, item)
}

func (self *ItemHeap) Pop() interface{} {
	old := *self
	n := len(old)
	item := old[n-1]
	*self = old[0:n-1]
	return item
}


func extract_feeds(doc *opml.OPML) []Feed {
	var res []Feed

	stack := lane.NewStack()

	for _, outline := range doc.Outlines() {
		stack.Push(outline)
	}

	for !stack.Empty() {
		outline := stack.Pop().(opml.Outline)

		if len(outline.Outlines) > 0 {
			for _, child := range outline.Outlines {
				stack.Push(child)
			}
		} else {
			feed_type := outline.Type
			if feed_type == "rss" {
				title := outline.Title
				xml := outline.XMLURL
				res = append(res, Feed{title, xml})
			}
		}
	}

	return res
}

func crawl(url string, ch chan<- *feeds.Item) {
	defer wg.Done()

	feed, err := rss.FetchByClient(url, client)
	fetched_count = fetched_count + 1
	fmt.Println("start fetching: [", fetched_count, "/", len(feed_list), "]", url)
	if err != nil {
		fmt.Println("error: ", url)
		fmt.Println(err.Error())
		return
	}

	fmt.Println("Done :", feed.Title)

	for _, item := range feed.Items {
		fmt.Println("Date: ", item.Date, "item: ", item.Title)
		i := &feeds.Item{
			Title:       item.Title,
			Link:        &feeds.Link{Href: item.Link},
			Description: item.Summary,
			Id:          item.ID,
			Updated:     item.Date,
		}
		heap.Push(i)
	}

	fmt.Println("Done")
	return
}

func main() {
	doc, err := opml.NewOPMLFromURL(opml_url)

	if err != nil {
		log.Fatal(err)
	}

	feed_ch := make(chan *feeds.Item)
	feed_list = extract_feeds(doc)

	for _, feed := range feed_list {
		url := feed.URL
		wg.Add(1)
		go func(uri string, ch chan<- *feeds.Item) {
			crawl(uri, ch)
		}(url, feed_ch)
	}

	wg.Wait()

	fmt.Println("LEN: ", heap.Len())

	for heap.Len() > 0 {
		i := heap.Pop()
		f := i.(*feeds.Item)
		fmt.Println(f.Updated, ";", f.Title)
	}
}
