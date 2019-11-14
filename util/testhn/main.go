package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
)

var url = flag.String("url", "https://news.ycombinator.com/item?id=21530860", "URL to scrape")

func main() {
	flag.Parse()
	//scrapeScoreAndComments(*url, func(score, comments int) {
	//	fmt.Printf("Finished scraping %s\nScore: %d, Comments: %d\n",
	//		*url, score, comments)
	//})
	score, comments := scrapeScoreAndComments(*url)
	fmt.Printf("Finished scraping %s\nScore: %d, Comments: %d\n",
		*url, score, comments)
}

func scrapeScoreAndComments(url string) (int, int) {
	c := colly.NewCollector()
	c.Async = true

	finished := make(chan bool)

	score := 0
	comments := 0

	c.OnHTML("td.subtext > span.score", func(r *colly.HTMLElement) {
		score, _ = strconv.Atoi(strings.Fields(r.Text)[0])
	})

	c.OnHTML("td.subtext > a[href*='item?id=']:last-of-type", func(r *colly.HTMLElement) {
		comments, _ = strconv.Atoi(strings.Fields(r.Text)[0])
	})

	c.OnScraped(func(r *colly.Response) {
		finished <- true
	})

	c.Visit(url)
	<-finished
	return score, comments
}
