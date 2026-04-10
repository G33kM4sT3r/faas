package ui

import (
	"errors"
	"fmt"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// SpinnerResult is sent when the work function completes.
type SpinnerResult struct {
	Output string
	Err    error
}

type spinnerModel struct {
	spinner  spinner.Model
	message  string
	result   *SpinnerResult
	quitting bool
}

// RunWithSpinner displays a spinner with the given message while workFn runs.
// If colors are disabled, runs workFn directly without a spinner.
func RunWithSpinner(message string, workFn func() (string, error)) (string, error) {
	if ColorsDisabled() {
		return workFn()
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

	m := spinnerModel{
		spinner: s,
		message: message,
	}

	p := tea.NewProgram(m)

	go func() {
		out, err := workFn()
		p.Send(SpinnerResult{Output: out, Err: err})
	}()

	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("spinner program failed: %w", err)
	}

	fm, ok := finalModel.(spinnerModel)
	if !ok {
		return "", errors.New("unexpected model type")
	}
	if fm.result != nil {
		return fm.result.Output, fm.result.Err
	}
	return "", errors.New("spinner exited without result")
}

func (m spinnerModel) Init() tea.Cmd { //nolint:gocritic // value receiver required by tea.Model interface
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { //nolint:gocritic // value receiver required by tea.Model interface
	switch msg := msg.(type) {
	case SpinnerResult:
		m.result = &msg
		m.quitting = true
		return m, tea.Quit
	case tea.KeyPressMsg:
		if msg.Code == 'c' && msg.Mod == tea.ModCtrl {
			m.quitting = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() tea.View { //nolint:gocritic // value receiver required by tea.Model interface
	if m.quitting {
		return tea.NewView("")
	}
	return tea.NewView(m.spinner.View() + " " + StyleDim.Render(m.message) + "\n")
}
