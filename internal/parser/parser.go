// Package parser provides HTML parsing for Apache-style directory listings.
package parser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// FileInfo represents a file in the directory listing
type FileInfo struct {
	Name string
	URL  string
	Size int64
}

// ParseDirectoryListing fetches and parses an Apache-style directory listing
func ParseDirectoryListing(ctx context.Context, directoryURL string) ([]FileInfo, error) {
	// Fetch the directory listing
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, directoryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent for polite web scraping
	req.Header.Set("User-Agent", "myrient-dl/1.0 (https://github.com/nchapman/myrient-dl)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch directory: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return parseHTML(resp.Body, directoryURL)
}

// parseHTML extracts file information from the HTML directory listing
func parseHTML(r io.Reader, baseURL string) ([]FileInfo, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var files []FileInfo

	// Apache directory listings use <a> tags for file links within table#list
	// We constrain to table#list to avoid picking up navigation links
	doc.Find("table#list a").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// Skip parent directory links
		if href == "../" || href == ".." {
			return
		}

		// Skip query parameters (sorting links)
		if strings.Contains(href, "?C=") {
			return
		}

		// Skip directories (end with /)
		if strings.HasSuffix(href, "/") {
			return
		}

		// Get the filename (text content of the link)
		name := strings.TrimSpace(s.Text())
		if name == "" {
			name = href
		}

		// Build absolute URL
		fileURL, err := buildAbsoluteURL(baseURL, href)
		if err != nil {
			return
		}

		// Try to extract size from the HTML
		// Apache listings typically show size in the same row
		size := extractSize(s)

		files = append(files, FileInfo{
			Name: name,
			URL:  fileURL,
			Size: size,
		})
	})

	return files, nil
}

// buildAbsoluteURL constructs an absolute URL from a base and relative path
func buildAbsoluteURL(base, relative string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	relURL, err := url.Parse(relative)
	if err != nil {
		return "", err
	}

	return baseURL.ResolveReference(relURL).String(), nil
}

// extractSize attempts to extract file size from the HTML context
// Apache directory listings show size like "70.5 KiB" or "1.2 MiB"
func extractSize(s *goquery.Selection) int64 {
	// Try multiple strategies to find the size

	// Strategy 1: Look in parent table cell (td)
	td := s.Parent()
	if td.Is("td") {
		// Look at the next sibling(s) for size
		nextTd := td.Next()
		if nextTd.Length() > 0 {
			text := nextTd.Text()
			if size := parseSizeString(text); size > 0 {
				return size
			}
		}
	}

	// Strategy 2: Look at the parent row (tr) or container
	row := s.Closest("tr")
	if row.Length() > 0 {
		text := row.Text()
		if size := parseSizeString(text); size > 0 {
			return size
		}
	}

	// Strategy 3: Look at parent element's text (for non-table layouts)
	text := s.Parent().Text()
	return parseSizeString(text)
}

// parseSizeString extracts size from a string like "70.5 KiB"
func parseSizeString(text string) int64 {
	// Try to find size patterns like "70.5 KiB", "1.2 MiB", "500 B"
	sizeRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(B|KiB|MiB|GiB|TiB|K|M|G|T)(?:\s|$)`)
	matches := sizeRegex.FindStringSubmatch(text)

	if len(matches) < 3 {
		return 0
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	unit := matches[2]

	// Convert to bytes
	multiplier := int64(1)
	switch unit {
	case "K", "KiB":
		multiplier = 1024
	case "M", "MiB":
		multiplier = 1024 * 1024
	case "G", "GiB":
		multiplier = 1024 * 1024 * 1024
	case "T", "TiB":
		multiplier = 1024 * 1024 * 1024 * 1024
	}

	return int64(value * float64(multiplier))
}
