package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// HiratsukaScraper scrapes the Hiratsuka city facility reservation system.
type HiratsukaScraper struct {
	client  *http.Client
	baseURL string
}

// NewHiratsukaScraper creates a new scraper for Hiratsuka city.
func NewHiratsukaScraper() *HiratsukaScraper {
	jar, _ := cookiejar.New(nil)
	return &HiratsukaScraper{
		client: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		baseURL: "https://shisetsu.city.hiratsuka.kanagawa.jp",
	}
}

func (s *HiratsukaScraper) Name() string {
	return "hiratsuka"
}

func (s *HiratsukaScraper) Scrape(ctx context.Context) (*Result, error) {
	result := &Result{
		ScrapedAt:   time.Now(),
		Diagnostics: make(map[string]interface{}),
	}

	// Step 1: Access top page to get session
	_, err := s.get(ctx, s.baseURL+"/cultos/reserve/gin_menu")
	if err != nil {
		result.Status = StatusNetworkError
		result.Error = fmt.Sprintf("failed to access top page: %v", err)
		return result, nil
	}

	// Step 2: Configure group selection for baseball fields
	// g_bunruicd_1=4 is for スポーツ施設 (Sports facilities)
	body, err := s.get(ctx, s.baseURL+"/cultos/reserve/gml_z_group_dest_sel")
	if err != nil {
		result.Status = StatusNetworkError
		result.Error = fmt.Sprintf("failed to access group selection: %v", err)
		return result, nil
	}

	token := extractSessionToken(body)
	if token == "" {
		result.Status = StatusParseError
		result.Error = "failed to extract session token"
		return result, nil
	}

	result.Diagnostics["token"] = token[:20] + "..."

	// Step 3: Select sports facilities category
	formData := url.Values{
		"g_bunruicd_1":      {"4"}, // スポーツ施設
		"g_bunruicd_1_show": {"4"},
		"g_kinonaiyo":       {"8"},
		"g_sessionid":       {token},
		"u_genzai_idx":      {"0"},
	}

	_, err = s.post(ctx, s.baseURL+"/cultos/reserve/gml_z_group_dest_sel", formData)
	if err != nil {
		result.Status = StatusNetworkError
		result.Error = fmt.Sprintf("failed to select group: %v", err)
		return result, nil
	}

	// Step 4: Select amenity (baseball)
	formData = url.Values{
		"g_kinonaiyo":  {"8"},
		"g_sessionid":  {token},
		"riyosmk":      {"2"},
		"u_genzai_idx": {"0"},
	}

	_, err = s.post(ctx, s.baseURL+"/cultos/reserve/gml_z_amenity_sel", formData)
	if err != nil {
		result.Status = StatusNetworkError
		result.Error = fmt.Sprintf("failed to select amenity: %v", err)
		return result, nil
	}

	// Step 5: Select room (facility)
	formData = url.Values{
		"heyacd": {"1"},
		"g_sessionid": {token},
		"u_genzai_idx": {"0"},
	}

	_, err = s.post(ctx, s.baseURL+"/cultos/reserve/gml_z_room_sel", formData)
	if err != nil {
		result.Status = StatusNetworkError
		result.Error = fmt.Sprintf("failed to select room: %v", err)
		return result, nil
	}

	// Step 6: Scrape available dates
	var allSlots []Slot
	now := time.Now()
	endDate := now.AddDate(0, 2, 0) // 2 months ahead

	for date := now; date.Before(endDate); date = date.AddDate(0, 0, 1) {
		dateStr := date.Format("2006-01-02")

		// Configure date selection
		formData = url.Values{
			"g_sessionid":  {token},
			"u_genzai_idx": {"0"},
			"tyumonbi":     {dateStr},
		}

		_, err = s.post(ctx, s.baseURL+"/cultos/reserve/gml_z_date_sel", formData)
		if err != nil {
			continue
		}

		// Get available time slots
		body, err := s.get(ctx, s.baseURL+"/cultos/reserve/gml_z_datetime_display")
		if err != nil {
			continue
		}

		slots := s.parseAvailability(body, date)
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

func (s *HiratsukaScraper) parseAvailability(body string, date time.Time) []Slot {
	var slots []Slot

	// Parse the HTML table for available slots
	// Look for cells with "空" (available) or "O" image
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return slots
	}

	dateStr := date.Format("2006-01-02")

	// Find all table rows with facility info
	var parseNode func(*html.Node)
	var currentFacility string

	parseNode = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Look for facility name in th elements
			if n.Data == "th" {
				for _, a := range n.Attr {
					if a.Key == "align" && a.Val == "left" {
						currentFacility = extractText(n)
					}
				}
			}

			// Look for available time slots (img alt="O")
			if n.Data == "img" {
				for _, a := range n.Attr {
					if a.Key == "alt" && a.Val == "O" {
						// Found available slot, extract time
						timeSlot := extractTimeFromSibling(n)
						if timeSlot != "" && currentFacility != "" {
							parts := strings.Split(timeSlot, "-")
							if len(parts) == 2 {
								slots = append(slots, Slot{
									Date:      dateStr,
									TimeFrom:  parts[0],
									TimeTo:    parts[1],
									CourtName: currentFacility,
									RawText:   fmt.Sprintf("%s %s %s", dateStr, timeSlot, currentFacility),
								})
							}
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseNode(c)
		}
	}

	parseNode(doc)

	return slots
}

func (s *HiratsukaScraper) get(ctx context.Context, urlStr string) (string, error) {
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

func (s *HiratsukaScraper) post(ctx context.Context, urlStr string, data url.Values) (string, error) {
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

func extractSessionToken(body string) string {
	// Look for g_sessionid input field
	re := regexp.MustCompile(`name="g_sessionid"\s+value="([^"]+)"`)
	matches := re.FindStringSubmatch(body)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func extractText(n *html.Node) string {
	var text string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			text += strings.TrimSpace(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return text
}

func extractTimeFromSibling(n *html.Node) string {
	// Look for time input in sibling elements
	for sib := n.NextSibling; sib != nil; sib = sib.NextSibling {
		if sib.Type == html.ElementNode && sib.Data == "input" {
			for _, a := range sib.Attr {
				if a.Key == "id" && strings.HasPrefix(a.Val, "kaisitime") {
					// Found end time, calculate start time (2 hour slots)
					for _, aa := range sib.Attr {
						if aa.Key == "value" {
							endTime := aa.Val
							startTime := subtractHours(endTime, 2)
							return startTime + "-" + endTime
						}
					}
				}
			}
		}
	}
	return ""
}

func subtractHours(timeStr string, hours int) string {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return timeStr
	}
	h, _ := strconv.Atoi(parts[0])
	h -= hours
	if h < 0 {
		h += 24
	}
	return fmt.Sprintf("%02d:%s", h, parts[1])
}
