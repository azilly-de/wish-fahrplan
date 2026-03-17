package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

type allDeparturesMsg struct {
	results map[string][]Departure
	errors  map[string]error
}

type model struct {
	connections []Connection
	results     map[string][]Departure
	errors      map[string]error
	loading     bool
	quitting    bool
	width       int
	lastUpdate  time.Time
}

func initialModel() model {
	return model{
		connections: connections,
		results:     make(map[string][]Departure),
		errors:      make(map[string]error),
		loading:     true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		fetchAllCmd(m.connections),
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(60*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchAllCmd(conns []Connection) tea.Cmd {
	return func() tea.Msg {
		type result struct {
			name string
			deps []Departure
			err  error
		}

		ch := make(chan result, len(conns))
		for _, c := range conns {
			go func(conn Connection) {
				deps, err := fetchDepartures(conn)
				ch <- result{name: conn.Name, deps: deps, err: err}
			}(c)
		}

		msg := allDeparturesMsg{
			results: make(map[string][]Departure),
			errors:  make(map[string]error),
		}
		for range conns {
			r := <-ch
			if r.err != nil {
				msg.errors[r.name] = r.err
			} else {
				msg.results[r.name] = r.deps
			}
		}
		return msg
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "r":
			m.loading = true
			return m, fetchAllCmd(m.connections)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tickMsg:
		m.loading = true
		return m, tea.Batch(
			fetchAllCmd(m.connections),
			tickCmd(),
		)

	case allDeparturesMsg:
		m.results = msg.results
		m.errors = msg.errors
		m.loading = false
		m.lastUpdate = time.Now()
	}

	return m, nil
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	connNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginTop(1)

	timeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	lineStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF8700"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4672"))

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginTop(1)

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4"))
)

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Zugplan - Fahrplanauskunft"))
	b.WriteString("\n")

	if m.loading && len(m.results) == 0 {
		b.WriteString(loadingStyle.Render("Lade Verbindungen..."))
		b.WriteString("\n")
		return b.String()
	}

	for _, conn := range m.connections {
		b.WriteString(connNameStyle.Render(fmt.Sprintf("  %s", conn.Name)))

		err, hasErr := m.errors[conn.Name]
		deps, hasDeps := m.results[conn.Name]

		if hasErr {
			b.WriteString("\n")
			b.WriteString(errorStyle.Render(fmt.Sprintf("    Fehler: %v", err)))
			b.WriteString("\n")
			continue
		}

		if !hasDeps {
			b.WriteString("\n")
			b.WriteString(loadingStyle.Render("    Lade..."))
			b.WriteString("\n")
			continue
		}

		if len(deps) == 0 {
			b.WriteString("\n")
			b.WriteString("    Keine Verbindungen gefunden.")
			b.WriteString("\n")
			continue
		}

		b.WriteString("\n")
		header := fmt.Sprintf("    %-7s %-7s %-10s %-25s %-5s %-6s %-8s",
			"Abfahrt", "Ankunft", "Linie", "Richtung", "Stop", "Umst.", "Dauer")
		b.WriteString(headerStyle.Render(header))
		b.WriteString("\n")

		for _, dep := range deps {
			transfer := "n"
			if dep.Transfer {
				transfer = "j"
			}
			line := fmt.Sprintf("    %-7s %-7s %-10s %-25s %-5d %-6s %-8s",
				timeStyle.Render(dep.Departure),
				timeStyle.Render(dep.Arrival),
				lineStyle.Render(truncate(dep.Line, 9)),
				truncate(dep.Direction, 24),
				dep.Stops,
				transfer,
				dep.Duration)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	if m.loading {
		b.WriteString(loadingStyle.Render("\n  Aktualisiere..."))
	}

	updateInfo := ""
	if !m.lastUpdate.IsZero() {
		updateInfo = fmt.Sprintf(" | Letzte Aktualisierung: %s", m.lastUpdate.Format("15:04:05"))
	}
	b.WriteString(footerStyle.Render(fmt.Sprintf("\n  [q] Beenden | [r] Aktualisieren%s", updateInfo)))

	return b.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
