package scraper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// YokohamaScraper scrapes the Yokohama city facility reservation system.
type YokohamaScraper struct {
	client  *http.Client
	baseURL string
}

// NewYokohamaScraper creates a new scraper for Yokohama city.
func NewYokohamaScraper() *YokohamaScraper {
	jar, _ := cookiejar.New(nil)
	return &YokohamaScraper{
		client: &http.Client{
			Jar:     jar,
			Timeout: 60 * time.Second,
		},
		baseURL: "https://www.shisetsu.city.yokohama.lg.jp",
	}
}

func (s *YokohamaScraper) Name() string {
	return "yokohama"
}

func (s *YokohamaScraper) Scrape(ctx context.Context) (*Result, error) {
	result := &Result{
		ScrapedAt:   time.Now(),
		Diagnostics: make(map[string]interface{}),
	}

	// Step 1: Access home page to get token
	body, err := s.get(ctx, s.baseURL+"/user/Home")
	if err != nil {
		result.Status = StatusNetworkError
		result.Error = fmt.Sprintf("failed to access home page: %v", err)
		return result, nil
	}

	token := s.extractToken(body)
	if token == "" {
		result.Status = StatusParseError
		result.Error = "failed to extract request verification token"
		return result, nil
	}

	result.Diagnostics["token"] = token[:20] + "..."

	// Step 2: Search for baseball facilities
	// Search for the current month and next month
	var allSlots []Slot

	for monthOffset := 0; monthOffset < 2; monthOffset++ {
		now := time.Now()
		if monthOffset == 1 {
			now = now.AddDate(0, 1, 0)
			now = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		}

		endOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())

		slots, err := s.searchMonth(ctx, token, now, endOfMonth)
		if err != nil {
			result.Diagnostics[fmt.Sprintf("month_%d_error", monthOffset)] = err.Error()
			continue
		}

		allSlots = append(allSlots, slots...)
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

func (s *YokohamaScraper) searchMonth(ctx context.Context, token string, startDate, endDate time.Time) ([]Slot, error) {
	// Create multipart form data
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add form fields
	fields := map[string][]string{
		"HomeModel.SearchByDateTimeModel.SelectedPlaceClass":         {"3", "9"}, // 3=野球場, 9=スポーツ広場
		"HomeModel.SearchByDateTimeModel.SelectedPlaceClassCategory": {"1"},
		"HomeModel.SearchByDateTimeModel.SelectedPurpose":            {"36"}, // 36=野球
		"HomeModel.SearchByDateTimeModel.SelectedPurposeCategory":    {"1"},
		"HomeModel.DateFrom":             {startDate.Format("2006-01-02")},
		"HomeModel.DateTo":               {endDate.Format("2006-01-02")},
		"HomeModel.TimeFrom":             {"0600"},
		"HomeModel.TimeTo":               {"2100"},
		"HomeModel.SelectedWeekDays":     {"月曜日,火曜日,水曜日,木曜日,金曜日,土曜日,日曜日"},
		"HomeModel.SelectedSearchTarget": {"1"},
		"SelectedLanguageCode":           {"0"},
		"__RequestVerificationToken":     {token},
	}

	for key, values := range fields {
		for _, value := range values {
			w.WriteField(key, value)
		}
	}
	w.Close()

	// Send search request
	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/user/Home/SearchByDateTime", &b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	// Get results page
	body, err := s.get(ctx, s.baseURL+"/user/VacantFrameFacilityStatus")
	if err != nil {
		return nil, err
	}

	return s.parseResults(body)
}

func (s *YokohamaScraper) parseResults(body string) ([]Slot, error) {
	var slots []Slot

	// Check for "no results" message
	if strings.Contains(body, "条件に該当する施設はありません") {
		return slots, nil
	}

	// Parse HTML table
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Find facilities table
	var parseTable func(*html.Node)
	parseTable = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			for _, a := range n.Attr {
				if a.Key == "class" && strings.Contains(a.Val, "facilities") {
					// Found the facilities table, parse rows
					for tr := n.FirstChild; tr != nil; tr = tr.NextSibling {
						if tr.Type == html.ElementNode && (tr.Data == "tr" || tr.Data == "tbody") {
							slot := s.parseTableRow(tr)
							if slot != nil {
								slots = append(slots, *slot)
							}
							// Also check children (for tbody)
							for child := tr.FirstChild; child != nil; child = child.NextSibling {
								if child.Type == html.ElementNode && child.Data == "tr" {
									slot := s.parseTableRow(child)
									if slot != nil {
										slots = append(slots, *slot)
									}
								}
							}
						}
					}
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseTable(c)
		}
	}
	parseTable(doc)

	return slots, nil
}

func (s *YokohamaScraper) parseTableRow(tr *html.Node) *Slot {
	var cells []string
	for td := tr.FirstChild; td != nil; td = td.NextSibling {
		if td.Type == html.ElementNode && td.Data == "td" {
			cells = append(cells, strings.TrimSpace(extractText(td)))
		}
	}

	// Expected columns: [checkbox, facility_name, location, date, time_slot, ...]
	if len(cells) < 5 {
		return nil
	}

	facilityName := cells[1]
	dateStr := cells[3]
	timeSlot := cells[4]

	// Parse date (e.g., "2026/01/25(土)" -> "2026-01-25")
	dateRe := regexp.MustCompile(`(\d{4})/(\d{2})/(\d{2})`)
	dateMatch := dateRe.FindStringSubmatch(dateStr)
	if len(dateMatch) < 4 {
		return nil
	}
	parsedDate := fmt.Sprintf("%s-%s-%s", dateMatch[1], dateMatch[2], dateMatch[3])

	// Parse time slot (e.g., "09:00～12:00" -> timeFrom, timeTo)
	timeRe := regexp.MustCompile(`(\d{2}:\d{2})[\-～~](\d{2}:\d{2})`)
	timeMatch := timeRe.FindStringSubmatch(timeSlot)
	if len(timeMatch) < 3 {
		return nil
	}

	return &Slot{
		Date:      parsedDate,
		TimeFrom:  timeMatch[1],
		TimeTo:    timeMatch[2],
		CourtName: facilityName,
		RawText:   fmt.Sprintf("%s %s %s", dateStr, timeSlot, facilityName),
	}
}

func (s *YokohamaScraper) get(ctx context.Context, urlStr string) (string, error) {
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

func (s *YokohamaScraper) extractToken(body string) string {
	re := regexp.MustCompile(`name="__RequestVerificationToken"\s+value="([^"]+)"`)
	matches := re.FindStringSubmatch(body)
	if len(matches) >= 2 {
		return matches[1]
	}

	// Alternative pattern
	re = regexp.MustCompile(`__RequestVerificationToken"\s*value="([^"]+)"`)
	matches = re.FindStringSubmatch(body)
	if len(matches) >= 2 {
		return matches[1]
	}

	return ""
}
