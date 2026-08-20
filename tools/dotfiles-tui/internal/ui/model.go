package ui

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/DevooKim/my-dotfiles/internal/dotfiles"
)

// Action is one top-level workflow.
type Action int

const (
	ActionInstall Action = iota
	ActionReapply
	ActionUpdate
	ActionRemove
	ActionDoctor
)

type stage int

const (
	stageActions stage = iota
	stagePackages
	stageConflicts
	stagePreview
	stageConfirm
	stageUpdateConfirm
	stageRunning
	stageResult
)

type actionItem struct {
	name        string
	description string
}

var actions = []actionItem{
	{"Install", "Create missing links for selected settings"},
	{"Reapply", "Repair selected settings and preserve correct links"},
	{"Update", "Fast-forward this repository, then reapply settings"},
	{"Remove", "Remove only links managed by selected settings"},
	{"Doctor", "Inspect sources, links, commands, references, and Git"},
}

// Dependencies contains side-effect boundaries used by the Bubble Tea model.
type Dependencies struct {
	Context context.Context
	Runner  dotfiles.CommandRunner
	Lookup  dotfiles.LookPath
	Now     func() time.Time
}

type operationDoneMsg struct {
	result dotfiles.ApplyResult
	err    error
}

type updateDoneMsg struct {
	err error
}

// InterruptMsg requests graceful cancellation without letting Bubble Tea stop
// before an in-flight filesystem transaction has finished rolling back.
type InterruptMsg struct{}

// Model is the complete Bubble Tea state machine.
type Model struct {
	repo     string
	home     string
	packages []dotfiles.Package
	deps     Dependencies

	stage            stage
	action           Action
	actionCursor     int
	packageCursor    int
	selected         map[string]bool
	inspections      map[string]dotfiles.Inspection
	inspectionErrors map[string]error

	conflicts       []string
	conflictCursor  int
	conflictChoices map[string]dotfiles.ConflictChoice
	plan            dotfiles.Plan

	findings           []dotfiles.Finding
	operationResult    dotfiles.ApplyResult
	operationErr       error
	resultTitle        string
	resultDetail       string
	cancel             context.CancelFunc
	quitting           bool
	quitAfterOperation bool
	restartRequested   bool
	afterUpdate        bool
	width              int
	help               help.Model
	keys               keyMap
}

// NewModel inspects current state before opening the action or post-update screen.
func NewModel(repo, home string, packages []dotfiles.Package, deps Dependencies, afterUpdate bool) (*Model, error) {
	if deps.Context == nil {
		deps.Context = context.Background()
	}
	if deps.Runner == nil {
		deps.Runner = dotfiles.OSCommandRunner{}
	}
	if deps.Lookup == nil {
		deps.Lookup = exec.LookPath
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	model := &Model{
		repo: repo, home: home, packages: append([]dotfiles.Package(nil), packages...), deps: deps,
		stage: stageActions, selected: make(map[string]bool), inspections: make(map[string]dotfiles.Inspection),
		inspectionErrors: make(map[string]error), afterUpdate: afterUpdate,
		conflictChoices: make(map[string]dotfiles.ConflictChoice), help: help.New(), keys: defaultKeyMap(),
	}
	if err := model.refreshInspections(); err != nil {
		return nil, err
	}
	if afterUpdate {
		model.action = ActionReapply
		model.stage = stagePackages
	}
	return model, nil
}

func (m *Model) Init() tea.Cmd { return nil }

// RestartRequested reports that Update succeeded and the clean terminal may exec the new binary.
func (m *Model) RestartRequested() bool { return m.restartRequested }

func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.help.SetWidth(msg.Width)
		return m, nil
	case operationDoneMsg:
		m.cancel = nil
		m.operationResult = msg.result
		m.operationErr = msg.err
		if m.afterUpdate {
			m.resultTitle = "Repository updated; settings reapplied"
			if msg.err != nil {
				m.resultTitle = "Repository updated; settings apply failed"
			}
		} else {
			m.resultTitle = "Operation complete"
			if msg.err != nil {
				m.resultTitle = "Operation failed"
			}
		}
		m.stage = stageResult
		if m.quitAfterOperation {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	case updateDoneMsg:
		m.cancel = nil
		if m.quitAfterOperation {
			m.operationErr = msg.err
			m.resultTitle = "Update canceled"
			m.stage = stageResult
			m.quitting = true
			return m, tea.Quit
		}
		if msg.err != nil {
			m.operationErr = msg.err
			m.resultTitle = "Update failed"
			m.stage = stageResult
			return m, nil
		}
		m.restartRequested = true
		m.quitting = true
		return m, tea.Quit
	case InterruptMsg:
		if m.stage == stageRunning && m.cancel != nil {
			m.quitAfterOperation = true
			m.cancel()
			return m, nil
		}
		return m.cancelOrQuit()
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	pressed := msg.String()
	if pressed == "q" || pressed == "esc" || pressed == "ctrl+c" {
		return m.cancelOrQuit()
	}

	switch m.stage {
	case stageActions:
		switch pressed {
		case "up", "k":
			m.actionCursor = wrap(m.actionCursor-1, len(actions))
		case "down", "j":
			m.actionCursor = wrap(m.actionCursor+1, len(actions))
		case "enter":
			m.action = Action(m.actionCursor)
			return m, m.enterAction()
		}

	case stagePackages:
		switch pressed {
		case "up", "k":
			m.packageCursor = wrap(m.packageCursor-1, len(m.packages))
		case "down", "j":
			m.packageCursor = wrap(m.packageCursor+1, len(m.packages))
		case " ", "space":
			name := m.packages[m.packageCursor].Name
			m.selected[name] = !m.selected[name]
		case "a":
			for _, pkg := range m.packages {
				m.selected[pkg.Name] = true
			}
		case "n":
			for _, pkg := range m.packages {
				m.selected[pkg.Name] = false
			}
		case "enter":
			if len(m.selectedPackages()) == 0 {
				m.resultDetail = "Select at least one setting"
				return m, nil
			}
			if err := m.preparePlan(); err != nil {
				m.operationErr = err
				m.resultTitle = m.previewFailureTitle()
				m.stage = stageResult
			}
		}

	case stageConflicts:
		current := m.conflicts[m.conflictCursor]
		switch pressed {
		case "b", "left", "right", " ", "space":
			if m.conflictChoices[current] == dotfiles.BackUpConflict {
				m.conflictChoices[current] = dotfiles.SkipConflict
			} else {
				m.conflictChoices[current] = dotfiles.BackUpConflict
			}
		case "s":
			m.conflictChoices[current] = dotfiles.SkipConflict
		case "enter":
			m.conflictCursor++
			if m.conflictCursor >= len(m.conflicts) {
				if err := m.buildApplyPlan(); err != nil {
					m.operationErr = err
					m.resultTitle = m.previewFailureTitle()
					m.stage = stageResult
				} else {
					m.stage = stagePreview
				}
			}
		}

	case stagePreview:
		if pressed == "enter" {
			m.stage = stageConfirm
		}

	case stageConfirm:
		switch pressed {
		case "y":
			ctx, cancel := context.WithCancel(m.deps.Context)
			m.cancel = cancel
			m.stage = stageRunning
			plan := m.plan
			return m, func() tea.Msg {
				result, err := dotfiles.Apply(ctx, plan, dotfiles.ApplyOptions{})
				return operationDoneMsg{result: result, err: err}
			}
		case "n":
			m.stage = stagePreview
		}

	case stageUpdateConfirm:
		switch pressed {
		case "y":
			ctx, cancel := context.WithCancel(m.deps.Context)
			m.cancel = cancel
			m.stage = stageRunning
			return m, func() tea.Msg {
				return updateDoneMsg{err: dotfiles.GitUpdate(ctx, m.repo, m.deps.Runner)}
			}
		case "n":
			m.stage = stageActions
		}

	case stageResult:
		if pressed == "enter" {
			m.resetToActions()
		}
	}
	return m, nil
}

func (m *Model) cancelOrQuit() (tea.Model, tea.Cmd) {
	if m.stage == stageRunning && m.cancel != nil {
		m.cancel()
		return m, nil
	}
	m.quitting = true
	return m, tea.Quit
}

func (m *Model) enterAction() tea.Cmd {
	m.plan = dotfiles.Plan{}
	m.operationResult = dotfiles.ApplyResult{}
	m.findings = nil
	m.resultDetail = ""
	m.operationErr = nil
	if m.action == ActionDoctor {
		m.findings = dotfiles.Doctor(m.repo, m.home, m.packages, m.deps.Lookup)
		m.resultTitle = "Doctor"
		m.stage = stageResult
		return nil
	}
	if m.action == ActionUpdate {
		if err := dotfiles.GitPreflight(m.deps.Context, m.repo, m.deps.Runner); err != nil {
			m.operationErr = err
			m.resultTitle = "Update unavailable"
			m.stage = stageResult
			return nil
		}
		m.stage = stageUpdateConfirm
		return nil
	}
	m.packageCursor = 0
	m.stage = stagePackages
	return nil
}

func (m *Model) preparePlan() error {
	if m.action == ActionRemove {
		plan, err := dotfiles.BuildRemovePlan(m.repo, m.home, m.selectedPackages())
		if err != nil {
			return err
		}
		m.plan = plan
		m.stage = stagePreview
		return nil
	}

	unique := make(map[string]bool)
	m.conflicts = nil
	m.conflictChoices = make(map[string]dotfiles.ConflictChoice)
	for _, pkg := range m.selectedPackages() {
		inspection := m.inspections[pkg.Name]
		for _, mapping := range inspection.Mappings {
			if mapping.State == dotfiles.TargetConflict && !unique[mapping.ConflictPath] {
				unique[mapping.ConflictPath] = true
				m.conflicts = append(m.conflicts, mapping.ConflictPath)
				m.conflictChoices[mapping.ConflictPath] = dotfiles.SkipConflict
			}
		}
	}
	sort.Strings(m.conflicts)
	if len(m.conflicts) > 0 {
		m.conflictCursor = 0
		m.stage = stageConflicts
		return nil
	}
	if err := m.buildApplyPlan(); err != nil {
		return err
	}
	m.stage = stagePreview
	return nil
}

func (m *Model) buildApplyPlan() error {
	stamp := m.deps.Now().Format("20060102-150405")
	backupRoot := filepath.Join(m.home, ".dotfiles-backups", stamp)
	plan, err := dotfiles.BuildApplyPlan(m.repo, m.home, backupRoot, m.selectedPackages(), m.conflictChoices)
	if err != nil {
		return err
	}
	m.plan = plan
	return nil
}

func (m *Model) selectedPackages() []dotfiles.Package {
	selected := make([]dotfiles.Package, 0, len(m.packages))
	for _, pkg := range m.packages {
		if m.selected[pkg.Name] {
			selected = append(selected, pkg)
		}
	}
	return selected
}

func (m *Model) refreshInspections() error {
	for _, pkg := range m.packages {
		inspection, err := dotfiles.InspectPackage(m.repo, m.home, pkg)
		if err != nil {
			m.inspectionErrors[pkg.Name] = err
			m.inspections[pkg.Name] = dotfiles.Inspection{Package: pkg, Status: dotfiles.Conflict}
			m.selected[pkg.Name] = false
			continue
		}
		delete(m.inspectionErrors, pkg.Name)
		m.inspections[pkg.Name] = inspection
		m.selected[pkg.Name] = inspection.Status == dotfiles.Installed
	}
	return nil
}

func (m *Model) resetToActions() {
	_ = m.refreshInspections()
	m.stage = stageActions
	m.actionCursor = 0
	m.plan = dotfiles.Plan{}
	m.operationResult = dotfiles.ApplyResult{}
	m.operationErr = nil
	m.quitAfterOperation = false
	m.resultTitle = ""
	m.resultDetail = ""
	m.findings = nil
	m.afterUpdate = false
}

func (m *Model) previewFailureTitle() string {
	if m.afterUpdate {
		return "Repository updated; settings preview failed"
	}
	return "Could not build preview"
}

func (m *Model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}
	var body strings.Builder
	body.WriteString(titleStyle.Render("Dotfiles Setup"))
	body.WriteString("\n\n")

	switch m.stage {
	case stageActions:
		body.WriteString(m.viewActions())
	case stagePackages:
		body.WriteString(m.viewPackages())
	case stageConflicts:
		body.WriteString(m.viewConflict())
	case stagePreview:
		body.WriteString(m.viewPreview())
	case stageConfirm:
		body.WriteString(warningStyle.Render("Apply the operations shown in the preview?"))
		body.WriteString("\n\nPress y to apply or n to return.")
	case stageUpdateConfirm:
		body.WriteString(warningStyle.Render("Update the dotfiles repository?"))
		body.WriteString("\n\nThis runs git pull --ff-only, then restarts the latest TUI.")
	case stageRunning:
		body.WriteString(accentStyle.Render("Working…"))
		body.WriteString("\n\nPress q or Esc to cancel safely and roll back filesystem changes.")
	case stageResult:
		body.WriteString(m.viewResult())
	}

	body.WriteString("\n\n")
	body.WriteString(m.help.View(m.keys))
	view := tea.NewView(panelStyle.Render(body.String()))
	view.AltScreen = true
	return view
}

func (m *Model) viewActions() string {
	var body strings.Builder
	body.WriteString("Choose an action\n\n")
	for index, item := range actions {
		cursor := "  "
		style := normalStyle
		if index == m.actionCursor {
			cursor = "› "
			style = selectedStyle
		}
		body.WriteString(cursor + style.Render(item.name) + "\n")
		body.WriteString("    " + mutedStyle.Render(item.description) + "\n")
	}
	return body.String()
}

func (m *Model) viewPackages() string {
	var body strings.Builder
	body.WriteString(fmt.Sprintf("%s settings\n\n", actions[m.action].name))
	for index, pkg := range m.packages {
		cursor := "  "
		if index == m.packageCursor {
			cursor = "› "
		}
		box := "[ ]"
		if m.selected[pkg.Name] {
			box = "[✓]"
		}
		status := m.inspections[pkg.Name].Status
		statusText := string(status)
		if m.inspectionErrors[pkg.Name] != nil {
			statusText = "source error"
		}
		body.WriteString(fmt.Sprintf("%s%s %-12s %-13s %s\n", cursor, box, pkg.Name, statusText, mutedStyle.Render(pkg.Description)))
	}
	body.WriteString("\n" + mutedStyle.Render("a select all • n clear all"))
	if m.resultDetail != "" {
		body.WriteString("\n" + warningStyle.Render(m.resultDetail))
	}
	return body.String()
}

func (m *Model) viewConflict() string {
	path := m.conflicts[m.conflictCursor]
	choice := m.conflictChoices[path]
	return fmt.Sprintf("Resolve conflict %d of %d\n\n%s\n\nChoice: %s\n\n%s",
		m.conflictCursor+1, len(m.conflicts), path, accentStyle.Render(string(choice)),
		mutedStyle.Render("b/Space toggle backup or skip • Enter continue"))
}

func (m *Model) viewPreview() string {
	var body strings.Builder
	body.WriteString("Preview\n\n")
	for _, operation := range m.plan.Operations {
		switch operation.Kind {
		case dotfiles.MakeDirectory:
			body.WriteString("  create directory  " + operation.Path + "\n")
		case dotfiles.MoveToBackup:
			body.WriteString("  back up          " + operation.Path + " → " + operation.Destination + "\n")
		case dotfiles.CreateLink:
			body.WriteString("  link             " + operation.Path + " → " + operation.Source + "\n")
		case dotfiles.RemoveLink:
			body.WriteString("  remove link      " + operation.Path + "\n")
		}
	}
	if len(m.plan.Operations) == 0 {
		body.WriteString(mutedStyle.Render("  No filesystem changes"))
	}
	if len(m.plan.Skipped) > 0 {
		body.WriteString(fmt.Sprintf("\n  %d conflicting or unmanaged path(s) skipped", len(m.plan.Skipped)))
	}
	body.WriteString("\n\nPress Enter for final confirmation.")
	return body.String()
}

func (m *Model) viewResult() string {
	var body strings.Builder
	body.WriteString(accentStyle.Render(m.resultTitle))
	if m.operationErr != nil {
		body.WriteString("\n\n" + errorStyle.Render(m.operationErr.Error()))
		if m.operationResult.RolledBack && len(m.operationResult.RollbackErrors) == 0 {
			body.WriteString("\n" + successStyle.Render("Filesystem changes rolled back"))
		} else if len(m.operationResult.RollbackErrors) > 0 {
			body.WriteString("\n" + errorStyle.Render("Rollback incomplete"))
		}
		for _, err := range m.operationResult.RollbackErrors {
			body.WriteString("\n" + errorStyle.Render("Rollback: "+err.Error()))
		}
	}
	if len(m.plan.Skipped) > 0 {
		body.WriteString(fmt.Sprintf("\n\n%d path(s) skipped", len(m.plan.Skipped)))
	}
	if m.resultDetail != "" {
		body.WriteString("\n\n" + m.resultDetail)
	}
	for _, finding := range m.findings {
		line := fmt.Sprintf("%-7s %-12s %-18s %s", finding.Severity, finding.Package, finding.Code, finding.Detail)
		switch finding.Severity {
		case dotfiles.SeverityOK:
			body.WriteString("\n" + successStyle.Render(line))
		case dotfiles.SeverityError:
			body.WriteString("\n" + errorStyle.Render(line))
		default:
			body.WriteString("\n" + warningStyle.Render(line))
		}
	}
	body.WriteString("\n\nPress Enter to return to the action menu.")
	return body.String()
}

func wrap(value, length int) int {
	if value < 0 {
		return length - 1
	}
	if value >= length {
		return 0
	}
	return value
}

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Toggle  key.Binding
	Confirm key.Binding
	Quit    key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Toggle:  key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle")),
		Confirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continue")),
		Quit:    key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q/esc", "cancel")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Toggle, k.Confirm, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Toggle}, {k.Confirm, k.Quit}}
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D7AFF"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A6E3A1"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#7F849C"))
	accentStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89B4FA"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))
	warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))
	panelStyle    = lipgloss.NewStyle().Padding(1, 2)
)
