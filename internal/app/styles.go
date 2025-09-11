package app

import "github.com/charmbracelet/lipgloss"

// Lucy palette
var (
	colBg     = lipgloss.Color("#180f6e")
	colFg     = lipgloss.Color("#dfc0e9")
	colPurple = lipgloss.Color("#7a258d")
	colMuted  = lipgloss.Color("#8771a6")
	colAccent = lipgloss.Color("#fada16")
)

type Styles struct {
	AppTitle      lipgloss.Style
	Header        lipgloss.Style
	HeaderNow     lipgloss.Style
	HeaderBadge   lipgloss.Style
	TabActive     lipgloss.Style
	TabInactive   lipgloss.Style
	Body          lipgloss.Style
	ListRow       lipgloss.Style
	ListRowDim    lipgloss.Style
	Cursor        lipgloss.Style
	Breadcrumb    lipgloss.Style
	Footer        lipgloss.Style
	Error         lipgloss.Style
	Panel         lipgloss.Style
	ProgressOuter lipgloss.Style
	ProgressFill  lipgloss.Style
}

func newStyles() Styles {
	base := lipgloss.NewStyle().Foreground(colFg)

	return Styles{
		AppTitle:    base.Bold(true),
		Header:      base.Background(colBg).Padding(0, 1),
		HeaderNow:   base.Faint(true),
		HeaderBadge: base.Foreground(colAccent).Bold(true),

		TabActive:   base.Bold(true).Background(colPurple).Padding(0, 1).MarginRight(1),
		TabInactive: base.Foreground(colMuted).Padding(0, 1).MarginRight(1),

		Body:       base.Padding(1, 2),
		ListRow:    base,
		ListRowDim: base.Foreground(colMuted),
		Cursor:     base.Foreground(colAccent),

		Breadcrumb: base.Foreground(colMuted),
		Footer:     base.Foreground(colMuted).Padding(0, 1),
		Error:      base.Foreground(lipgloss.Color("#ff6b6b")),

		Panel: base.
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colPurple).
			Padding(0, 1).
			Margin(0, 1),

		ProgressOuter: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colMuted).
			Padding(0, 1),

		ProgressFill: lipgloss.NewStyle().
			Foreground(colBg).
			Background(colAccent),
	}
}

