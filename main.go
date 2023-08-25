package main

// An example Bubble Tea server. This will put an ssh session into alt screen
// and continually print up-to-date terminal information.

import (
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/glamour"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/muesli/termenv"
)

const (
	host = "localhost"
	port = 23234
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render

const content = `
# ssh.nikitatarasov.dev

## Moin

I'm Nikita Tarasov, a self-taught developer from Germany

with experience in Full-Stack Development using Java and JavaScript. 

 - Let's get in touch -> [hi@nikitatarasov.dev]()

## Background

 Hi!

I'm a Software Developer and Electronics Technician based in Germany

During the final year of my training as an Electronics Technician for Devices and Systems,
I was introduced to modern software development. 
I created a "Cocktail Machine" using Java on a Raspberry Pi 4, controlling pumps via GPIO pins,
guided by an experienced developer who offered insights and best practices.

After completing my training, I started working as a Test Field Technician. 
However, a portion of my time I contributed to improving and further developing the company's internal error database. 
This project involved a JavaEE server, a Spring MVC web interface, and a JavaFX Client.

After switching to a different company, I didn't have the opportunity to work on any software projects 
so I decided to start learning on my own. During free time, I started learning web development. 
I started with HTML, CSS, and JavaScript, and later I moved on to Vue.js. 
I also started learning Kotlin for Backend and Android development and Python for small scripts. 

## About Me

Alongside my development work, I also have a solid foundation in electronics and hardware.
I love tinkering with electronics and building small projects using microcontrollers like the Arduino and microcomputers like the Raspberry Pi.

I also enjoy watching movies and TV shows in my free time. 
My favorite genres are science fiction and superhero movies. You can follow my watch progress on [Trakt](https://trakt.tv/users/nikita-t1).

Also I'm really into reading books. 
I mostly read non-fiction books about social sciences, psychology, and politics, but occasionally I also read some science fiction or even romance novels. 
You can follow my reading progress on [Goodreads](https://www.goodreads.com/user/show/143627750-nikita).


## Skills

| Languages  | Frameworks            | Tools         |
|------------|-----------------------|---------------|
| JavaScript | Vue                   | VS Code       |
| TypeScript | Node.js               | WebStorm      |
| HTML       | Nuxt                  | Gradle        |
| CSS        | Prisma                | Maven         |
| Java       | Spring Boot           | IntelliJ IDEA |
| Kotlin     | Ktor                  | Eclipse IDE   |
| Python     | Compose Multiplatform | Git           |
| SQL        | JavaFX                | Bash          |
|            | Hibernate             | Docker        |

# Bon appétit!
`

func middlewareWithLogger() wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			ct := time.Now()
			hpk := s.PublicKey() != nil
			pty, _, _ := s.Pty()
			log.Info("New Connection", "user", s.User(), "remote_addr", s.RemoteAddr().String(), "public_key", hpk, "command", s.Command(), "term", pty.Term, "width", pty.Window.Width, "height", pty.Window.Height)
			sh(s)
			log.Info("Connection closed", "remote_addr", s.RemoteAddr().String(), "duration", time.Since(ct))
		}
	}
}

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithMiddleware(
			myCustomBubbleteaMiddleware(),
			middlewareWithLogger(),
		),
	)
	if err != nil {
		log.Error("could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("could not stop server", "error", err)
	}
}

// You can write your own custom bubbletea middleware that wraps tea.Program.
// Make sure you set the program input and output to ssh.Session.
func myCustomBubbleteaMiddleware() wish.Middleware {
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
			viewport: viewport.New(pty.Window.Width, pty.Window.Height),
		}
		return newProg(m, tea.WithInput(s), tea.WithOutput(s), tea.WithAltScreen())
	}
	return bm.MiddlewareWithProgramHandler(teaHandler, termenv.ANSI256)
}

func initModel(m model) (model, error) {
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

	str, err := renderer.Render(content)
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
		m.height = msg.Height
		m.width = msg.Width
		m, _ = initModel(m)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
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
	return m.viewport.View()
	//return fmt.Sprintf(s, m.term, m.width, m.height)
}
