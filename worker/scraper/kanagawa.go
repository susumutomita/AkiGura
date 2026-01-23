package scraper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// KanagawaScraper scrapes the e-kanagawa facility reservation system.
// This covers facilities like Hodogaya Park (Kanagawa Prefecture).
type KanagawaScraper struct {
	client     *http.Client
	baseURL    string
	facilities []kanagawaFacility
}

type kanagawaFacility struct {
	Code string // e.g., "01" for サーティーフォー保土ケ谷球場
	Name string
}

// NewKanagawaScraper creates a new scraper for Kanagawa Prefecture facilities.
func NewKanagawaScraper() *KanagawaScraper {
	jar, _ := cookiejar.New(nil)
	return &KanagawaScraper{
		client: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		baseURL: "https://yoyaku.e-kanagawa.lg.jp",
		facilities: []kanagawaFacility{
			{Code: "01", Name: "サーティーフォー保土ケ谷球場"},
			{Code: "08", Name: "軟式野球場全面"},
			{Code: "09", Name: "軟式野球場半面Ａ"},
			{Code: "10", Name: "軟式野球場半面Ｂ"},
		},
	}
}

func (s *KanagawaScraper) Name() string {
	return "kanagawa"
}

func (s *KanagawaScraper) Scrape(ctx context.Context) (*Result, error) {
	result := &Result{
		ScrapedAt:   time.Now(),
		Diagnostics: make(map[string]interface{}),
	}

	// Step 1: Access top page to establish session
	if _, err := s.get(ctx, s.baseURL+"/Portal/Web/Wgp_Map.aspx"); err != nil {
		result.Status = StatusNetworkError
		result.Error = fmt.Sprintf("failed to access top page: %v", err)
		return result, nil
	}

	// Step 2: Access smartphone page
	if _, err := s.get(ctx, s.baseURL+"/Kanagawa/SmartPhone"); err != nil {
		result.Status = StatusNetworkError
		result.Error = fmt.Sprintf("failed to access smartphone page: %v", err)
		return result, nil
	}

	// Step 3: Access facility selection page
	body, err := s.get(ctx, s.baseURL+"/Kanagawa/SmartPhone/Wsp_ShisetsuSentaku.aspx")
	if err != nil {
		result.Status = StatusNetworkError
		result.Error = fmt.Sprintf("failed to access facility selection: %v", err)
		return result, nil
	}

	viewState := extractViewState(body)
	if viewState == "" {
		result.Status = StatusParseError
		result.Error = "failed to extract ViewState"
		return result, nil
	}

	// Step 4: Select Hodogaya Park (000001) and get UFPS
	formData := url.Values{
		"__EVENTTARGET":     {"cmdNext"},
		"__EVENTARGUMENT":   {""},
		"__VIEWSTATE":       {viewState},
		"slShisetsu$rbList": {"000001"},
		"slNen":             {"0"},
		"slTsuki":           {"0"},
		"slHi":              {"0"},
		"cmdNext":           {"次へ"},
	}

	body, err = s.post(ctx, s.baseURL+"/Kanagawa/SmartPhone/Wsp_ShisetsuSentaku.aspx", formData)
	if err != nil {
		result.Status = StatusNetworkError
		result.Error = fmt.Sprintf("failed to select facility: %v", err)
		return result, nil
	}

	// Extract UFPS from form action
	ufps := extractUFPS(body)
	if ufps == "" {
		result.Status = StatusParseError
		result.Error = "failed to extract UFPS"
		return result, nil
	}

	result.Diagnostics["ufps"] = ufps

	// Step 5: Scrape each facility for each date
	var allSlots []Slot
	now := time.Now()

	// Check dates for the next 60 days
	for dayOffset := 0; dayOffset < 60; dayOffset++ {
		targetDate := now.AddDate(0, 0, dayOffset)
		dateStr := targetDate.Format("20060102")

		for _, fac := range s.facilities {
			slots, err := s.scrapeDate(ctx, ufps, fac, dateStr, targetDate)
			if err != nil {
				continue
			}
			allSlots = append(allSlots, slots...)
		}
	}

	// Filter out excluded facilities
	var filteredSlots []Slot
	for _, slot := range allSlots {
		if !ShouldExclude(slot.CourtName) {
			filteredSlots = append(filteredSlots, slot)
		}
	}

	result.Slots = filteredSlots
	result.Success = true
	if len(filteredSlots) > 0 {
		result.Status = StatusSuccess
	} else {
		result.Status = StatusSuccessEmpty
	}

	return result, nil
}

func (s *KanagawaScraper) scrapeDate(ctx context.Context, ufps string, fac kanagawaFacility, dateStr string, targetDate time.Time) ([]Slot, error) {
	timeURL := fmt.Sprintf("%s/Kanagawa/SmartPhone/Wsp_JikanSentaku.aspx?__ufps=%s&SJCode=%s&UseDate=%s",
		s.baseURL, ufps, fac.Code, dateStr)

	body, err := s.get(ctx, timeURL)
	if err != nil {
		return nil, err
	}

	// Check for error page
	if strings.Contains(body, "エラー") {
		return nil, fmt.Errorf("error page returned")
	}

	// Parse available time slots
	return s.parseTimeSlots(body, fac.Name, targetDate)
}

func (s *KanagawaScraper) parseTimeSlots(body string, facilityName string, date time.Time) ([]Slot, error) {
	var slots []Slot

	// Look for available time slots indicated by "空" (available) markers
	// The page shows a table with time slots
	// If there's "申込できる空きがありません" it means no availability
	if strings.Contains(body, "申込できる空きがありません") {
		return slots, nil
	}

	// Parse time slots from the HTML
	// Time slots are typically in format like "06:00" to "08:00"
	timePattern := regexp.MustCompile(`(\d{2}:\d{2})\s*[\-～~]\s*(\d{2}:\d{2})`)

	// Look for links/buttons that indicate available slots
	if strings.Contains(body, "空") || strings.Contains(body, "○") {
		// Extract time ranges
		matches := timePattern.FindAllStringSubmatch(body, -1)
		dateStr := date.Format("2006-01-02")

		for _, match := range matches {
			if len(match) >= 3 {
				slots = append(slots, Slot{
					Date:      dateStr,
					TimeFrom:  match[1],
					TimeTo:    match[2],
					CourtName: facilityName,
					RawText:   fmt.Sprintf("%s %s-%s %s", dateStr, match[1], match[2], facilityName),
				})
			}
		}
	}

	return slots, nil
}

func (s *KanagawaScraper) get(ctx context.Context, urlStr string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (s *KanagawaScraper) post(ctx context.Context, urlStr string, data url.Values) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func extractViewState(body string) string {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return ""
	}

	var viewState string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			var name, value string
			for _, a := range n.Attr {
				if a.Key == "name" {
					name = a.Val
				}
				if a.Key == "value" {
					value = a.Val
				}
			}
			if name == "__VIEWSTATE" {
				viewState = value
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return viewState
}

func extractUFPS(body string) string {
	// Extract __ufps from form action URL
	re := regexp.MustCompile(`__ufps=(\d+)`)
	matches := re.FindStringSubmatch(body)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// Helper to check if body contains a string (avoiding import issues)
func bodyContains(body []byte, s string) bool {
	return bytes.Contains(body, []byte(s))
}
