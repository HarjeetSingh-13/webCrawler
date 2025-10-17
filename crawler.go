package main

import (
	"encoding/csv"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Config struct {
	pages              map[string]PageData
	baseURLhost        string
	maxPages		   int
	mu                 *sync.Mutex
	concurrencyControl chan struct{}
	wg                 *sync.WaitGroup
}

type PageData struct{
	URL              string
	H1               string
	FirstParagraph	 string
	OutgoingLinks   []string
	ImageURLs       []string
}

func NormalizeURL(inputURL string) (string, error) {
	u, err := url.Parse(inputURL)
	if err != nil {
		return "", err
	}
	path := strings.Trim(u.Path, "/")
	return u.Host+"/"+path, nil
}

func getH1FromHTML(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}
	res := doc.Find("h1").First()
	return res.Text()
}

func getFirstParagraphFromHTML(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}
	res := doc.Find("p").First()
	main := doc.Find("main")
	if main.Length() != 0 {
		res = main.Find("p").First()
	}
	return res.Text()
}

func getURLsFromHTML(htmlBody string, baseURL *url.URL) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	var res []string
	if err != nil {
		return res, err
	}
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if val, ok := s.Attr("href"); ok {
			u, err := url.Parse(val)
			if err != nil {
				return 
			}
			if u.Scheme == "" {
				u.Scheme = baseURL.Scheme
			}
			if u.Host == "" {
				u.Host = baseURL.Host
			}
			res = append(res, u.Scheme+"://"+u.Host+u.Path)
		}
	})
	return res, nil
}

func getImagesFromHTML(htmlBody string, baseURL *url.URL) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	var res []string
	if err != nil {
		return res, err
	}
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		if val, ok := s.Attr("src"); ok {
						u, err := url.Parse(val)
			if err != nil {
				return 
			}
			if u.Scheme == "" {
				u.Scheme = baseURL.Scheme
			}
			if u.Host == "" {
				u.Host = baseURL.Host
			}
			res = append(res, u.Scheme+"://"+u.Host+u.Path)
		}
	})
	return res, nil
}

func ExtractPageData(html, pageURL string) PageData {
	var res PageData
	u, err := url.Parse(pageURL)
	if err!= nil {
		return res
	}
	outgoingLinks, err := getURLsFromHTML(html, u)
	if err != nil {
		return res
	}
	imageURLs, err := getImagesFromHTML(html, u)
	if err != nil {
		return res
	}
	res.URL = pageURL
	res.H1 = getH1FromHTML(html)
	res.FirstParagraph = getFirstParagraphFromHTML(html)
	res.OutgoingLinks = outgoingLinks
	res.ImageURLs = imageURLs
	return res
}

func getHTML(rawURL string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("User-Agent", "crawler/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", errors.New("error response code")
	}

	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		return "", errors.New("invalid content type")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("error reading the html data")
	}
	return string(body), nil
}

func (cfg *Config) addPageVisit(normalizedURL string, data PageData)  bool {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	if len(cfg.pages) >= cfg.maxPages {
		return true
	}
	if _, ok := cfg.pages[normalizedURL]; ok {
		return true
	}
	cfg.pages[normalizedURL] = data
	return false
}

func (cfg *Config) CrawlPage(rawCurrentURL string) {
	defer cfg.wg.Done()
	currentDomain, err := url.Parse(rawCurrentURL)
	if err != nil {
		return
	}
	if cfg.baseURLhost != currentDomain.Host {
		return
	}
	normalizeURL, err := NormalizeURL(rawCurrentURL)
	if err != nil {
		return
	}
	html, err := getHTML(rawCurrentURL)
	if err != nil {
		log.Printf("error fetching %s: %s", rawCurrentURL, err)
		return
	}
	data := ExtractPageData(html, rawCurrentURL)
	if cfg.addPageVisit(normalizeURL, data) {
		return
	}
	for _, link := range data.OutgoingLinks {
		cfg.wg.Add(1)
		go func (l string)  {
			cfg.concurrencyControl <- struct{}{}
			defer func() { <-cfg.concurrencyControl }()
			cfg.CrawlPage(l)
		}(link)
	}
}

func writeCSVRport(pages map[string]PageData, filename string) error{
	log.Println(pages)
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	reportPath := filepath.Join(currentDir, "report.csv")
	file, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	csvWriter := csv.NewWriter(file)
	defer csvWriter.Flush()

	csvWriter.Write([]string{"page_url", "h1", "first_paragraph", "outgoing_link_urls", "image_urls"})
	for _, val := range pages {
		outgoingLinks := strings.Join(val.OutgoingLinks, ";")
		imageLinks := strings.Join(val.ImageURLs, ";")
		csvWriter.Write([]string{val.URL, val.H1, val.FirstParagraph, outgoingLinks, imageLinks})
	}
	return nil
}