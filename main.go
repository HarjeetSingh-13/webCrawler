package main

import (
	"log"
	"net/url"
	"os"
	"strconv"
	"sync"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatalln("no website provided")
		os.Exit(1)
	}
	if len(args) > 3 {
		log.Fatalln("too many arguments provided")
		os.Exit(1)
	}
	baseURL := args[0]
	maxConcurrency, _ := strconv.Atoi(args[1])
	maxPages, _ := strconv.Atoi(args[2])
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalln("error parsing the base url!")
		return
	}
	cfg := Config{
		baseURLhost: parsedBaseURL.Host,
		pages: make(map[string]PageData),
		maxPages: maxPages,
		mu: &sync.Mutex{},
		concurrencyControl: make(chan struct{}, maxConcurrency),
		wg: &sync.WaitGroup{},
	}
	log.Printf("starting crawl of: %s", baseURL)
	cfg.wg.Add(1)
	go func (l string)  {
		cfg.concurrencyControl <- struct{}{}
		defer func() { <-cfg.concurrencyControl }()
		cfg.CrawlPage(l)
	}(baseURL)
	cfg.wg.Wait()
	log.Println("writting result into file")
	err = writeCSVRport(cfg.pages, "report.csv")
	if err != nil {
		log.Fatalf("error writting to file: %s", err)
		return
	}
	log.Println("Data written successfully.")
}