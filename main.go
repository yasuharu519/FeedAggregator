package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gilliek/go-opml/opml"
	"github.com/oleiade/lane"
)

const (
	opml_url = "https://raw.githubusercontent.com/yasuharu519/opml/master/main.opml"
)

type Feed struct {
	Title string
	URL   string
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

func crawl(url string) {
	fmt.Println("start fetching: ", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(resp)
	}
}

func main() {
	doc, err := opml.NewOPMLFromURL(opml_url)

	if err != nil {
		log.Fatal(err)
	}

	queue := make(chan string)

	list := extract_feeds(doc)
	for _, feed := range list {
		url := feed.URL
		go func() {
			queue <- url
		}()
	}

	for uri := range queue {
		go crawl(uri)
	}
}
