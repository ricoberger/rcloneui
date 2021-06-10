package view

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

type Status struct {
	*tview.TextView

	currentRemote string
	currentPath   []string

	selectedRemote string
	selectedPath   []string

	action string
}

func (s *Status) render() {
	if s.currentRemote != "" && len(s.selectedPath) > 0 {
		s.SetText(fmt.Sprintf("[black:blue] %s:%s [black:black] [black:blue] %s %s:%s ", s.currentRemote, strings.Join(s.currentPath, "/"), s.action, s.selectedRemote, strings.Join(s.selectedPath, "/")))
	} else if s.currentRemote != "" {
		s.SetText(fmt.Sprintf("[black:blue] %s:%s [black:black] [black:blue] - ", s.currentRemote, strings.Join(s.currentPath, "/")))
	} else if len(s.selectedPath) > 0 {
		s.SetText(fmt.Sprintf("[black:blue] - [black:black] [black:blue] %s %s:%s ", s.action, s.selectedRemote, strings.Join(s.selectedPath, "/")))
	} else {
		s.SetText("")
	}
}

func (s *Status) SetLocation(currentRemote string, currentPath []string) {
	s.currentRemote = currentRemote
	s.currentPath = currentPath

	s.render()
}

func (s *Status) SetSelect(selectedRemote string, selectedPath []string, action string) {
	s.selectedRemote = selectedRemote
	s.selectedPath = selectedPath
	s.action = action

	s.render()
}

func (s *Status) GetSelectedRemote() string {
	return s.selectedRemote
}

func (s *Status) GetSelectedPath() []string {
	return s.selectedPath
}

func (s *Status) GetAction() string {
	return s.action
}

// NewStatus returns the status bar component. We display the current remote, path and action in the status bar.
func NewStatus(app *tview.Application) *Status {
	text := tview.NewTextView().SetDynamicColors(true).SetRegions(true).SetChangedFunc(func() {
		app.Draw()
	}).SetTextAlign(tview.AlignLeft)

	return &Status{
		text,
		"",
		nil,
		"",
		nil,
		"",
	}
}
