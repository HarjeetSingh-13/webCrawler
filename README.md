# Web Crawler

A concurrent web crawler written in Go that can efficiently crawl websites and collect data about their pages.

## Features

- Concurrent crawling with configurable concurrency level
- Respects website's domain boundaries (doesn't crawl external links)
- Collects page data including:
  - URLs
  - H1 headings
  - First Paragraph
  - Links found on each page
- Outputs results to a CSV file
- Configurable maximum page limit

## Requirements

- Go 1.23.0 or higher

## Dependencies

- github.com/PuerkitoBio/goquery - HTML parsing and manipulation
- golang.org/x/net - Network utilities

## Usage

```bash
go run . [baseURL] [maxConcurrency] [maxPages]
```

### Parameters

- `baseURL`: The starting URL to begin crawling from
- `maxConcurrency`: Maximum number of concurrent crawling operations
- `maxPages`: Maximum number of pages to crawl

### Example

```bash
go run . https://example.com 10 100
```

This will:

1. Start crawling from https://example.com
2. Use 10 concurrent crawlers
3. Stop after crawling 100 pages
4. Generate a report.csv file with the results

## Output

The crawler generates a `report.csv` file containing information about each crawled page, including:

- Page URL
- H1 heading content
- First Paragraph
- Links found on the page
