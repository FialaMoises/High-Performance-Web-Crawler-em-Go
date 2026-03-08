package parser

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// HTMLParser handles parsing HTML documents to extract links
type HTMLParser struct {
	baseURL *url.URL
}

// NewHTMLParser creates a new HTMLParser with the given base URL
func NewHTMLParser(baseURL string) (*HTMLParser, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &HTMLParser{
		baseURL: parsedURL,
	}, nil
}

// ParseLinks extracts all valid links from an HTML document
func (p *HTMLParser) ParseLinks(doc *goquery.Document, currentURL string) ([]string, error) {
	var links []string
	seen := make(map[string]bool)

	currentParsed, err := url.Parse(currentURL)
	if err != nil {
		return nil, err
	}

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// Parse and resolve the URL
		linkURL, err := url.Parse(href)
		if err != nil {
			return
		}

		// Resolve relative URLs
		absoluteURL := currentParsed.ResolveReference(linkURL)

		// Normalize the URL
		normalizedURL := p.normalizeURL(absoluteURL)

		// Skip invalid URLs
		if normalizedURL == "" {
			return
		}

		// Deduplicate
		if !seen[normalizedURL] {
			seen[normalizedURL] = true
			links = append(links, normalizedURL)
		}
	})

	return links, nil
}

// normalizeURL cleans and normalizes a URL
func (p *HTMLParser) normalizeURL(u *url.URL) string {
	// Only accept http and https
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}

	// Remove fragment
	u.Fragment = ""

	// Lowercase scheme and host
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)

	return u.String()
}

// IsSameDomain checks if a URL belongs to the same domain as the base URL
func (p *HTMLParser) IsSameDomain(targetURL string) bool {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	return strings.ToLower(parsed.Host) == strings.ToLower(p.baseURL.Host)
}

// GetDomain extracts the domain from a URL
func GetDomain(urlStr string) (string, error) {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	return parsed.Host, nil
}