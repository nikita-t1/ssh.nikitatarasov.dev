package main

import (
	_ "embed"
	"fmt"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/muesli/termenv"
	"time"
)

//go:embed websiteContent.md
var websiteContent string

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render

// You can write your own custom bubbletea middleware that wraps tea.Program.
// Make sure you set the program input and output to ssh.Session.
func myCustomBubbleTeaMiddleware() wish.Middleware {
	newProg := func(m tea.Model, opts ...tea.ProgramOption) *tea.Program {
		p := tea.NewProgram(m, opts...)
		go func() {
			for {
				<-time.After(1 * time.Second)
				p.Send(timeMsg(time.Now()))
			}
		}()
		return p
	}
	teaHandler := func(s ssh.Session) *tea.Program {
		pty, _, active := s.Pty()
		if !active {
			wish.Fatalln(s, "no active terminal, skipping")
			return nil
		}
		m := model{
			term:     pty.Term,
			width:    pty.Window.Width,
			height:   pty.Window.Height,
			time:     time.Now(),
			viewport: viewport.New(pty.Window.Width, pty.Window.Height-4),
			help:     help.New(),
			keys:     keys,
		}
		return newProg(m, tea.WithInput(s), tea.WithOutput(s), tea.WithAltScreen())
	}
	return bm.MiddlewareWithProgramHandler(teaHandler, termenv.ANSI256)
}

func updateModel(m model) (model, error) {
	m.viewport.Height = m.height
	m.viewport.Width = m.width
	m.viewport.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingRight(2)

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithPreservedNewLines(),
		glamour.WithWordWrap(m.width-4),
	)
	if err != nil {
		return m, nil
	}

	str, err := renderer.Render(websiteContent)
	if err != nil {
		return m, nil
	}

	m.viewport.SetContent(str)
	return m, nil
}

type model struct {
	term     string
	width    int
	height   int
	time     time.Time
	viewport viewport.Model
	help     help.Model
	keys     keyMap
}

// keyMap defines a set of keybindings. To work for help it must satisfy
// key.Map. It could also very easily be a map[string]key.Binding.
type keyMap struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
	View  key.Binding
	Help  key.Binding
	Quit  key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.View, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right}, // first column
		{k.Help, k.View, k.Quit},        // second column
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "move left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "move right"),
	),
	View: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "toggle view"),
	),
	Help: key.NewBinding(
		key.WithKeys("?", "/"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

type timeMsg time.Time

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case timeMsg:
		m.time = time.Time(msg)
	case tea.WindowSizeMsg:
		m.height = msg.Height - 4
		m.width = msg.Width
		m, _ = updateModel(m)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m model) View() string {
	s := "Your term is %s\n"
	s += "Your window size is x: %d y: %d\n"
	s += "Time: " + m.time.Format(time.RFC1123) + "\n\n"
	s += "Press 'q' to quit\n"
	if m.width < 80 || m.height < 24 {
		s = "Terminal too small to display\n"
		return fmt.Sprintf(s)
	}
	help := lipgloss.NewStyle().PaddingLeft(2).Render(helpStyle(m.help.View(m.keys)))
	return m.viewport.View() + "\n" + help
	//return fmt.Sprintf(s, m.term, m.width, m.height)
}
