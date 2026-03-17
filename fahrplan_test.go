package main

import (
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
)

func TestBuildURL(t *testing.T) {
	conn := Connection{
		Name:       "Test",
		OriginID:   "de:07111:010240",
		OriginType: "stop",
		DestID:     "de:07111:010001",
		DestType:   "stop",
	}
	fixedTime := time.Date(2026, 3, 15, 20, 0, 0, 0, time.UTC)
	url := buildURL(conn, fixedTime)

	tests := []struct {
		name string
		want string
	}{
		{"contains base URL", "https://www.vrs.de/partner/vrm/fahrplanauskunft"},
		{"contains origin ID", "de%3A07111%3A010240"},
		{"contains destination ID", "de%3A07111%3A010001"},
		{"contains date", "15.03.2026"},
		{"contains time", "20%3A00"},
		{"contains request", "result"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(url, tt.want) {
				t.Errorf("buildURL() = %q, want to contain %q", url, tt.want)
			}
		})
	}
}

func TestParseDeparturesMatchingOrigin(t *testing.T) {
	htmlContent := `
<!DOCTYPE html>
<html>
<body>
<div class="route-details" id="route-details-0">
	<div class="route-detail-row">
		<div class="route-number-times d-flex">
			<span class="route-number">1</span>
			<div class="departure-arrival">
				<span class="departure">ab <strong>20:20</strong></span><br/>
				<span class="arrival">an <strong>20:35</strong></span>
			</div>
		</div>
		<div class="changes" title="Umstiege"><br/>1</div>
		<div class="duration" title="Dauer"><br/>15 min</div>
	</div>
	<div>
		<div class="route-segments alternate-background">
			<div class="segment first-segment origin">
				<div class="orig-dest-name"><a href="/haltestelle/de:07111:010240"><strong>Franz-Weis-Straße</strong>, Koblenz-Rauental</a></div>
			</div>
			<div class="segment">
				<div class="line-image"><img src="bus.svg" alt="Bus"/></div>
				<div class="line"><strong>Linie 6</strong> Richtung: Horchheim, Im Baumgarten</div>
			</div>
			<div class="segment  show-hide-intermediates">
				<a class="show-intermediates"><span class="data" data-route="0" data-segment="0">Zwischenhalte anzeigen</span></a>
			</div>
			<div class="segment  intermediates intermediates-0-0" style="display:none;">
				<div class="name"><a href="/haltestelle/de:07111:010029"><strong>Saarplatz</strong>, Koblenz</a></div>
			</div>
			<div class="segment  intermediates intermediates-0-0" style="display:none;">
				<div class="name"><a href="/haltestelle/de:07111:010030"><strong>Moselweiß</strong>, Koblenz</a></div>
			</div>
			<div class="segment last-segment destination">
				<div class="orig-dest-name"><a href="/haltestelle/de:07111:010003"><strong>Bahnhof Stadtmitte</strong>, Koblenz</a></div>
			</div>
		</div>
		<div class="route-segments alternate-background">
			<div class="segment first-segment origin"></div>
			<div class="segment">
				<div class="walk">Fußweg 67 m, 1 min</div>
			</div>
			<div class="segment last-segment destination"></div>
		</div>
		<div class="route-segments alternate-background">
			<div class="segment first-segment origin">
				<div class="orig-dest-name"><a href="/haltestelle/de:07111:010003"><strong>Bahnhof Stadtmitte</strong>, Koblenz</a></div>
			</div>
			<div class="segment">
				<div class="line-image"><img src="bus.svg" alt="Bus"/></div>
				<div class="line"><strong>Linie 620</strong> Richtung: Simmern</div>
			</div>
			<div class="segment last-segment destination">
				<div class="orig-dest-name"><a href="/haltestelle/de:07111:010001"><strong>Hauptbahnhof</strong>, Koblenz</a></div>
			</div>
		</div>
	</div>
</div>
<div class="route-details" id="route-details-1">
	<div class="route-detail-row">
		<div class="route-number-times d-flex">
			<span class="route-number">2</span>
			<div class="departure-arrival">
				<span class="departure">ab <strong>20:22</strong></span><br/>
				<span class="arrival">an <strong>20:40</strong></span>
			</div>
		</div>
		<div class="changes" title="Umstiege"><br/>0</div>
		<div class="duration" title="Dauer"><br/>18 min</div>
	</div>
	<div>
		<div class="route-segments alternate-background">
			<div class="segment first-segment origin">
				<div class="orig-dest-name"><a href="/haltestelle/de:07111:010240"><strong>Franz-Weis-Straße</strong>, Koblenz-Rauental</a></div>
			</div>
			<div class="segment">
				<div class="line-image"><img src="bus.svg" alt="Bus"/></div>
				<div class="line"><strong>Linie 3</strong> Richtung: Oberwerth, Stadion</div>
			</div>
			<div class="segment last-segment destination">
				<div class="orig-dest-name"><a href="/haltestelle/de:07111:010001"><strong>Hauptbahnhof</strong>, Koblenz</a></div>
			</div>
		</div>
	</div>
</div>
</body>
</html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
	}

	// Filter by originID de:07111:010240
	departures := parseDepartures(doc, "de:07111:010240")

	if len(departures) != 2 {
		t.Fatalf("expected 2 departures, got %d", len(departures))
	}

	dep := departures[0]
	if dep.Departure != "20:20" {
		t.Errorf("departure time = %q, want %q", dep.Departure, "20:20")
	}
	if dep.Arrival != "20:35" {
		t.Errorf("arrival time = %q, want %q", dep.Arrival, "20:35")
	}
	if dep.Line != "Linie 6" {
		t.Errorf("line = %q, want %q", dep.Line, "Linie 6")
	}
	if dep.Direction != "Horchheim, Im Baumgarten" {
		t.Errorf("direction = %q, want %q", dep.Direction, "Horchheim, Im Baumgarten")
	}
	if !dep.Transfer {
		t.Errorf("transfer = false, want true (changes=1)")
	}
	if dep.Duration != "15 min" {
		t.Errorf("duration = %q, want %q", dep.Duration, "15 min")
	}
	if dep.Stops != 2 {
		t.Errorf("stops = %d, want 2", dep.Stops)
	}

	dep2 := departures[1]
	if dep2.Departure != "20:22" {
		t.Errorf("departure time = %q, want %q", dep2.Departure, "20:22")
	}
	if dep2.Line != "Linie 3" {
		t.Errorf("line = %q, want %q", dep2.Line, "Linie 3")
	}
	if dep2.Transfer {
		t.Errorf("transfer = true, want false (changes=0)")
	}
}

func TestParseDeparturesFiltersWrongOrigin(t *testing.T) {
	// Route where first transit segment starts from a different stop (transfer)
	htmlContent := `
<!DOCTYPE html>
<html>
<body>
<div class="route-details" id="route-details-0">
	<div class="route-detail-row">
		<div class="route-number-times d-flex">
			<span class="route-number">1</span>
			<div class="departure-arrival">
				<span class="departure">ab <strong>20:20</strong></span><br/>
				<span class="arrival">an <strong>20:35</strong></span>
			</div>
		</div>
		<div class="changes"><br/>1</div>
		<div class="duration"><br/>15 min</div>
	</div>
	<div>
		<div class="route-segments">
			<div class="segment first-segment origin">
				<div class="orig-dest-name"><a href="/haltestelle/de:07111:099999"><strong>Other Stop</strong>, Somewhere</a></div>
			</div>
			<div class="segment">
				<div class="line-image"><img src="bus.svg" alt="Bus"/></div>
				<div class="line"><strong>Linie 99</strong> Richtung: Elsewhere</div>
			</div>
			<div class="segment last-segment destination">
				<div class="orig-dest-name"><a href="/haltestelle/de:07111:010001"><strong>Hauptbahnhof</strong>, Koblenz</a></div>
			</div>
		</div>
	</div>
</div>
</body>
</html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
	}

	// Filter by originID de:07111:010240 - should NOT match
	departures := parseDepartures(doc, "de:07111:010240")

	if len(departures) != 0 {
		t.Errorf("expected 0 departures (wrong origin), got %d", len(departures))
	}

	// But should match when searching for the correct origin
	departures2 := parseDepartures(doc, "de:07111:099999")
	if len(departures2) != 1 {
		t.Errorf("expected 1 departure for correct origin, got %d", len(departures2))
	}
}

func TestParseDeparturesEmpty(t *testing.T) {
	htmlContent := `<html><body><p>No results</p></body></html>`
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
	}

	departures := parseDepartures(doc, "de:07111:010240")
	if len(departures) != 0 {
		t.Errorf("expected 0 departures, got %d", len(departures))
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"very short max", "hello", 3, "hel"},
		{"empty string", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}
