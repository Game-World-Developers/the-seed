package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"Game-Developers-World/seed/internal/generators"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenDashboard screen = iota
	screenList
	screenDetail
	screenProblems
)

type model struct {
	width  int
	height int

	tab      int
	selected int
	screen   screen

	models []generators.ModelInfo

	detail       *generators.ModelDetail
	detailType   string
	detailName   string
	detailDomain string
	detailErr    error
	loading      bool

	problems    []generators.Problem
	problemsErr error

	err error
}

type detailLoadedMsg struct {
	detail *generators.ModelDetail
	err    error
}

type problemsLoadedMsg struct {
	problems []generators.Problem
	err      error
}

var tabNames = []string{
	"Dashboard",
	"Components",
	"Traits",
	"Entities",
	"Archetypes",
	"State Machines",
	"Events",
	"Assets",
	"Systems",
}

var tabTypes = []string{
	"",
	"component",
	"trait",
	"entity",
	"archetype",
	"state_machine",
	"event",
	"asset",
	"system",
}

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7B59C4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	warnCol   = lipgloss.AdaptiveColor{Light: "#DC9656", Dark: "#E0AF68"}
	errorCol  = lipgloss.AdaptiveColor{Light: "#CC241D", Dark: "#FB4934"}

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight).
			Padding(0, 1)

	tabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(subtle)

	activeTabStyle = tabStyle.Copy().
			Foreground(lipgloss.Color("#FFF")).
			Background(highlight).
			Bold(true)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(1, 2).
			Width(25)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(highlight).
				Bold(true)

	detailLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(subtle)

	syncOKStyle = lipgloss.NewStyle().
			Foreground(special).
			Bold(true)

	syncStaleStyle = lipgloss.NewStyle().
			Foreground(warnCol).
			Bold(true)

	syncMissingStyle = lipgloss.NewStyle().
				Foreground(errorCol).
				Bold(true)

	problemErrorStyle = lipgloss.NewStyle().
				Foreground(errorCol)

	problemWarnStyle = lipgloss.NewStyle().
				Foreground(warnCol)

	backStyle = lipgloss.NewStyle().
			Foreground(highlight).
			Underline(true)
)

func Run() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func initialModel() model {
	all, err := generators.ListModels()
	if err != nil {
		return model{err: err}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Type != all[j].Type {
			return all[i].Type < all[j].Type
		}
		return all[i].Name < all[j].Name
	})
	return model{
		models: all,
		tab:    int(screenDashboard),
		screen: screenDashboard,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case detailLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.detailErr = msg.err
		} else {
			m.detail = msg.detail
			m.detailErr = nil
		}
		return m, nil

	case problemsLoadedMsg:
		if msg.err != nil {
			m.problemsErr = msg.err
		} else {
			m.problems = msg.problems
			m.problemsErr = nil
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			return m.handleEsc()

		case "tab", "right", "l":
			return m.handleNextTab()

		case "shift+tab", "left", "h":
			return m.handlePrevTab()

		case "down", "j":
			return m.handleDown()

		case "up", "k":
			return m.handleUp()

		case "enter":
			return m.handleEnter()

		case "p":
			return m.handleToggleProblems()
		}
	}

	return m, nil
}

func (m model) handleEsc() (model, tea.Cmd) {
	switch m.screen {
	case screenDetail:
		m.screen = screenList
		m.detail = nil
		m.detailErr = nil
	case screenProblems:
		m.screen = screenDashboard
	}
	return m, nil
}

func (m model) handleNextTab() (model, tea.Cmd) {
	switch m.screen {
	case screenDashboard, screenList:
		if m.tab < len(tabNames)-1 {
			m.tab++
			m.selected = 0
			m.screen = screenList
		}
	}
	return m, nil
}

func (m model) handlePrevTab() (model, tea.Cmd) {
	switch m.screen {
	case screenDashboard, screenList:
		if m.tab > 0 {
			m.tab--
			m.selected = 0
		}
		if m.tab == 0 {
			m.screen = screenDashboard
		} else {
			m.screen = screenList
		}
	}
	return m, nil
}

func (m model) handleDown() (model, tea.Cmd) {
	switch m.screen {
	case screenList:
		models := m.filteredModels()
		if m.selected < len(models)-1 {
			m.selected++
		}
	case screenProblems:
		if m.selected < len(m.problems)-1 {
			m.selected++
		}
	}
	return m, nil
}

func (m model) handleUp() (model, tea.Cmd) {
	switch m.screen {
	case screenList, screenProblems:
		if m.selected > 0 {
			m.selected--
		}
	}
	return m, nil
}

func (m model) handleEnter() (model, tea.Cmd) {
	switch m.screen {
	case screenDashboard:
		m.screen = screenList
		if m.tab == 0 {
			m.tab = 1
		}
		m.selected = 0
		return m, nil

	case screenList:
		models := m.filteredModels()
		if m.selected >= 0 && m.selected < len(models) {
			mo := models[m.selected]
			m.detailType = mo.Type
			m.detailName = mo.Name
			m.detailDomain = mo.Namespace
			m.screen = screenDetail
			m.loading = true
			m.detail = nil
			m.detailErr = nil
			return m, loadDetail(mo.Type, mo.Namespace, mo.Name)
		}
	}
	return m, nil
}

func (m model) handleToggleProblems() (model, tea.Cmd) {
	if m.screen == screenProblems {
		m.screen = screenDashboard
		return m, nil
	}
	m.screen = screenProblems
	m.selected = 0
	if m.problems == nil && m.problemsErr == nil {
		return m, loadProblems()
	}
	return m, nil
}

func loadDetail(compType, domain, name string) tea.Cmd {
	return func() tea.Msg {
		d, err := generators.GetModelDetail(compType, domain, name)
		return detailLoadedMsg{detail: d, err: err}
	}
}

func loadProblems() tea.Cmd {
	return func() tea.Msg {
		p, err := generators.GetProblems()
		return problemsLoadedMsg{problems: p, err: err}
	}
}

func (m model) filteredModels() []generators.ModelInfo {
	if m.tab <= 0 || m.tab >= len(tabTypes) {
		return nil
	}
	t := tabTypes[m.tab]
	var filtered []generators.ModelInfo
	for _, mo := range m.models {
		if mo.Type == t {
			filtered = append(filtered, mo)
		}
	}
	return filtered
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	projName := projectName()
	b.WriteString(titleStyle.Render(projName))
	b.WriteString("\n\n")

	for i := 0; i < len(tabNames); i++ {
		if i == m.tab {
			b.WriteString(activeTabStyle.Render(tabNames[i]))
		} else {
			b.WriteString(tabStyle.Render(tabNames[i]))
		}
		b.WriteString(" ")
	}
	b.WriteString("\n\n")

	switch m.screen {
	case screenDashboard:
		b.WriteString(m.dashboardView())
	case screenList:
		b.WriteString(m.listView())
	case screenDetail:
		b.WriteString(m.detailView())
	case screenProblems:
		b.WriteString(m.problemsView())
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func projectName() string {
	wd, err := os.Getwd()
	if err != nil {
		return "seed"
	}
	return filepath.Base(wd)
}

func (m model) dashboardView() string {
	typeCount := map[string]int{}
	for _, mo := range m.models {
		typeCount[mo.Type]++
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Model Summary"))
	b.WriteString("\n\n")

	types := []string{"component", "trait", "entity", "archetype", "state_machine", "event", "asset", "system"}
	var total int
	for _, t := range types {
		c := typeCount[t]
		total += c
		label := capitalize(t)
		rendered := fmt.Sprintf("%s\n\n%d", label, c)
		style := cardStyle.Copy()
		if c == 0 {
			style = style.Copy().BorderForeground(errorCol)
		}
		b.WriteString(style.Render(rendered))
		b.WriteString(" ")
	}

	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(special).
		Render(fmt.Sprintf("Total: %d models", total)))

	b.WriteString("\n\n")

	if m.problemsErr != nil {
		b.WriteString(problemErrorStyle.Render(fmt.Sprintf("Error loading problems: %v", m.problemsErr)))
	} else if len(m.problems) > 0 {
		errCount := 0
		warnCount := 0
		for _, p := range m.problems {
			switch p.Level {
			case generators.ProblemError:
				errCount++
			case generators.ProblemWarning:
				warnCount++
			}
		}
		if errCount > 0 {
			b.WriteString(problemErrorStyle.Render(fmt.Sprintf("✘ %d errors", errCount)))
			b.WriteString("  ")
		}
		if warnCount > 0 {
			b.WriteString(problemWarnStyle.Render(fmt.Sprintf("⚠ %d warnings", warnCount)))
		}
		b.WriteString("  ")
		b.WriteString(backStyle.Render("[p] view details"))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(special).Render("✓ No problems detected"))
	}

	b.WriteString("\n\n")
	b.WriteString(subtleStyle().Render("Enter focus · Tab navigate · p problems · q quit"))

	return b.String()
}

func (m model) listView() string {
	models := m.filteredModels()
	tabType := tabTypes[m.tab]

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(tabNames[m.tab]))
	b.WriteString("\n")

	if len(models) == 0 {
		b.WriteString(fmt.Sprintf("\n  No %s models found.", tabType))
		b.WriteString("\n\n")
		b.WriteString(subtleStyle().Render("Tab switch · q quit"))
		return b.String()
	}

	b.WriteString("\n")

	problemMap := map[string][]generators.Problem{}
	if m.problems != nil {
		for _, p := range m.problems {
			if p.ModelType == tabType {
				problemMap[p.ModelName] = append(problemMap[p.ModelName], p)
			}
		}
	}

	for i, mo := range models {
		line := fmt.Sprintf("  %s", mo.Name)
		if probs, ok := problemMap[mo.Name]; ok {
			hasError := false
			for _, p := range probs {
				if p.Level == generators.ProblemError {
					hasError = true
					break
				}
			}
			if hasError {
				line = problemErrorStyle.Render(fmt.Sprintf("  ✘ %s", mo.Name))
			} else {
				line = problemWarnStyle.Render(fmt.Sprintf("  ⚠ %s", mo.Name))
			}
		}

		if i == m.selected {
			b.WriteString("> ")
			b.WriteString(selectedItemStyle.Render(strings.TrimLeft(line, " ")))
		} else {
			b.WriteString("  ")
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(subtleStyle().Render(
		fmt.Sprintf("↑ ↓ navigate · Enter detail · Tab switch · p problems · q quit (showing %d)", len(models)),
	))

	return b.String()
}

func (m model) detailView() string {
	if m.loading {
		return "\n  Loading..."
	}

	if m.detailErr != nil {
		return fmt.Sprintf("\n  Error: %v", m.detailErr)
	}

	if m.detail == nil {
		return "\n  No data."
	}

	d := m.detail
	var b strings.Builder

	b.WriteString(backStyle.Render("← Back"))
	b.WriteString("\n\n")

	syncIcon := ""
	syncStyle := syncOKStyle
	switch d.SyncStatus {
	case "ok":
		syncIcon = "✓ Synced"
		syncStyle = syncOKStyle
	case "stale":
		syncIcon = "⚠ Stale (needs sync)"
		syncStyle = syncStaleStyle
	case "missing":
		syncIcon = "✘ Missing header"
		syncStyle = syncMissingStyle
	default:
		syncIcon = "? Unknown"
	}

	header := fmt.Sprintf("%s (%s)", d.Name, capitalize(d.Type))
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(header))
	b.WriteString("  ")
	b.WriteString(syncStyle.Render(syncIcon))
	b.WriteString("\n\n")

	b.WriteString(detailLabelStyle.Render("Namespace"))
	b.WriteString(fmt.Sprintf("  %s\n", d.Namespace))

	if len(d.Dependencies) > 0 {
		b.WriteString(detailLabelStyle.Render("Depends on"))
		b.WriteString("\n")
		for _, dep := range d.Dependencies {
			status := "✓"
			b.WriteString(fmt.Sprintf("    %s  %s (%s)\n", status, dep.Name, dep.Namespace))
		}
		b.WriteString("\n")
	}

	if len(d.Dependents) > 0 {
		b.WriteString(detailLabelStyle.Render("Used by"))
		b.WriteString("\n")
		for _, dep := range d.Dependents {
			b.WriteString(fmt.Sprintf("    • %s (%s)\n", dep.Name, dep.Namespace))
		}
		b.WriteString("\n")
	}

	b.WriteString(detailLabelStyle.Render("YAML"))
	b.WriteString("\n")
	yamlLines := strings.Split(d.YAML, "\n")
	for _, line := range yamlLines {
		b.WriteString(fmt.Sprintf("  %s\n", line))
	}

	b.WriteString("\n")
	b.WriteString(subtleStyle().Render("Esc back · Tab next · q quit"))

	return b.String()
}

func (m model) modelExists(t, n string) bool {
	for _, mo := range m.models {
		if mo.Type == t && mo.Name == n {
			return true
		}
	}
	return false
}

func (m model) problemsView() string {
	var b strings.Builder

	header := fmt.Sprintf("Problems (%d)", len(m.problems))
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(header))
	b.WriteString("\n\n")

	if m.problemsErr != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n", m.problemsErr))
		b.WriteString("\n")
		b.WriteString(subtleStyle().Render("p toggle · q quit"))
		return b.String()
	}

	if len(m.problems) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(special).Render("  ✓ No problems found"))
		b.WriteString("\n\n")
		b.WriteString(subtleStyle().Render("p toggle · q quit"))
		return b.String()
	}

	for i, p := range m.problems {
		icon := "⚠"
		style := problemWarnStyle
		if p.Level == generators.ProblemError {
			icon = "✘"
			style = problemErrorStyle
		}

		line := fmt.Sprintf("%s  %s %s — %s", icon, capitalize(p.ModelType), p.ModelName, p.Message)

		if i == m.selected {
			b.WriteString("> ")
			b.WriteString(selectedItemStyle.Render(strings.TrimLeft(line, " ")))
		} else {
			b.WriteString("  ")
			b.WriteString(style.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(subtleStyle().Render("↑ ↓ scroll · p toggle · Esc back · q quit"))

	return b.String()
}

func subtleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(subtle).Italic(true)
}

func capitalize(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
