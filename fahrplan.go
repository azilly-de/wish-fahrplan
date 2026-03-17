package main

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type Departure struct {
	Departure string
	Arrival   string
	Line      string
	Direction string
	Stop      string
	Stops     int
	Transfer  bool
	Duration  string
}

var (
	timeRegex     = regexp.MustCompile(`\d{2}:\d{2}`)
	stopIDRegex   = regexp.MustCompile(`/haltestelle/(de:\d+:\d+)`)
	lineLinkRegex = regexp.MustCompile(`/linien-details/linie/([^"]+)`)
)

func buildURL(conn Connection, t time.Time) string {
	base := "https://www.vrs.de/partner/vrm/fahrplanauskunft"
	params := url.Values{}
	params.Set("tx_vrsinfo_pi_connection[originId]", conn.OriginID)
	params.Set("tx_vrsinfo_pi_connection[originType]", conn.OriginType)
	params.Set("tx_vrsinfo_pi_connection[destinationId]", conn.DestID)
	params.Set("tx_vrsinfo_pi_connection[destinationType]", conn.DestType)
	params.Set("tx_vrsinfo_pi_connection[date]", t.Format("02.01.2006"))
	params.Set("tx_vrsinfo_pi_connection[time]", t.Format("15:04"))
	params.Set("tx_vrsinfo_pi_connection[request]", "result")
	return base + "?" + params.Encode()
}

func fetchDepartures(conn Connection) ([]Departure, error) {
	now := time.Now()
	pageURL := buildURL(conn, now)

	resp, err := http.Get(pageURL)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", conn.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: status %d", conn.Name, resp.StatusCode)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML for %s: %w", conn.Name, err)
	}

	departures := parseDepartures(doc, conn.OriginID)
	return departures, nil
}

func parseDepartures(doc *html.Node, originID string) []Departure {
	var departures []Departure

	var findDetails func(*html.Node)
	findDetails = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			id := getAttr(n, "id")
			if strings.HasPrefix(id, "route-details-") {
				dep := parseRouteDetail(n, originID)
				if dep.Departure != "" {
					departures = append(departures, dep)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findDetails(c)
		}
	}
	findDetails(doc)
	return departures
}

func parseRouteDetail(n *html.Node, originID string) Departure {
	var dep Departure

	dep.Departure = findTextByClass(n, "departure")
	dep.Arrival = findTextByClass(n, "arrival")
	dep.Duration = findDivText(n, "duration")

	// Parse changes count for transfer flag
	changesStr := findDivText(n, "changes")
	dep.Transfer = changesStr != "" && changesStr != "0"

	// Find the first transit segment (not walk) whose origin stop matches originID
	seg := findFirstTransitSegment(n, originID)
	if seg.line == "" {
		return Departure{}
	}

	dep.Line = seg.line
	dep.Direction = seg.direction
	dep.Stop = seg.originStop
	dep.Stops = seg.stops

	return dep
}

type transitSegment struct {
	line         string
	direction    string
	originStop   string
	originStopID string
	stops        int
}

func findFirstTransitSegment(n *html.Node, originID string) transitSegment {
	var result transitSegment

	// Find all route-segments groups
	var groups []*html.Node
	var findGroups func(*html.Node)
	findGroups = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			cls := getAttr(n, "class")
			if strings.Contains(cls, "route-segments") {
				groups = append(groups, n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findGroups(c)
		}
	}
	findGroups(n)

	// Each route-segments group is a trip segment (origin -> destination within one leg)
	// Find the first group where the origin stop ID matches
	for _, g := range groups {
		seg := parseSegmentGroup(g)
		if seg.line == "" {
			continue // walk segment or no line info
		}
		if originID == "" || seg.originStopID == originID {
			result = seg
			break
		}
	}

	return result
}

func parseSegmentGroup(n *html.Node) transitSegment {
	var seg transitSegment

	// Find the origin segment and extract stop ID from its link
	var findOrigin func(*html.Node)
	findOrigin = func(n *html.Node) {
		if seg.originStopID != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" {
			cls := getAttr(n, "class")
			if strings.Contains(cls, "orig-dest-name") && strings.Contains(cls, "origin") {
				// Look for a link with the stop ID
				seg.originStopID = findStopIDInSubtree(n)
				seg.originStop = extractText(n)
				seg.originStop = strings.TrimSpace(seg.originStop)
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findOrigin(c)
		}
	}
	findOrigin(n)

	// If origin not found with origin class, try first orig-dest-name div
	if seg.originStopID == "" {
		var findFirstOrig func(*html.Node)
		findFirstOrig = func(n *html.Node) {
			if seg.originStopID != "" {
				return
			}
			if n.Type == html.ElementNode && n.Data == "div" {
				cls := getAttr(n, "class")
				if strings.Contains(cls, "orig-dest-name") {
					seg.originStopID = findStopIDInSubtree(n)
					seg.originStop = strings.TrimSpace(extractText(n))
					return
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				findFirstOrig(c)
			}
		}
		findFirstOrig(n)
	}

	// Find the line info (skip walk segments)
	var findLine func(*html.Node)
	findLine = func(n *html.Node) {
		if seg.line != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" {
			cls := getAttr(n, "class")
			if hasClass(cls, "line") && !hasClass(cls, "line-image") {
				text := extractText(n)
				text = strings.TrimSpace(text)
				if text != "" && !strings.Contains(text, "Fußweg") {
					parts := strings.Split(text, "Richtung:")
					seg.line = strings.TrimSpace(parts[0])
					if len(parts) > 1 {
						seg.direction = strings.TrimSpace(strings.SplitN(parts[1], "\n", 2)[0])
					}
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findLine(c)
		}
	}
	findLine(n)

	// Count intermediate stops in this segment group
	seg.stops = countIntermediates(n)

	return seg
}

func countIntermediates(n *html.Node) int {
	count := 0
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			cls := getAttr(n, "class")
			if hasClass(cls, "intermediates") && !hasClass(cls, "show-hide-intermediates") {
				count++
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(n)
	return count
}

func findStopIDInSubtree(n *html.Node) string {
	var result string
	var find func(*html.Node)
	find = func(n *html.Node) {
		if result != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			href := getAttr(n, "href")
			if strings.Contains(href, "/haltestelle/") {
				m := stopIDRegex.FindStringSubmatch(href)
				if len(m) > 1 {
					result = m[1]
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(n)
	return result
}

func findTextByClass(n *html.Node, className string) string {
	var result string
	var find func(*html.Node)
	find = func(n *html.Node) {
		if result != "" {
			return
		}
		if n.Type == html.ElementNode {
			cls := getAttr(n, "class")
			if hasClass(cls, className) {
				text := extractText(n)
				text = strings.TrimSpace(text)
				matches := timeRegex.FindString(text)
				if matches != "" {
					result = matches
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(n)
	return result
}

func hasClass(attr, className string) bool {
	for _, c := range strings.Fields(attr) {
		if c == className {
			return true
		}
	}
	return false
}

func findDivText(n *html.Node, className string) string {
	var result string
	var find func(*html.Node)
	find = func(n *html.Node) {
		if result != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" {
			cls := getAttr(n, "class")
			if hasClass(cls, className) {
				text := extractText(n)
				text = strings.TrimSpace(text)
				if text != "" {
					result = text
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(n)
	return result
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var buf strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		buf.WriteString(extractText(c))
	}
	return buf.String()
}
