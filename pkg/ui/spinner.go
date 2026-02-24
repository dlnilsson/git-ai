package ui

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

type SignalForwarder interface {
	ForwardSignal(sig os.Signal)
}

type spinnerDoneMsg struct{}

type spinnerReasoningMsg string

type spinnerModel struct {
	spinner           spinner.Model
	message           string
	backend           string
	reasoning         string
	reasoningRendered string
	done              bool
	start             time.Time
	forwarder         SignalForwarder
}

type spinnerHandle struct {
	program  *tea.Program
	reasonCh chan string
	doneCh   chan struct{}
}

var spinnerMessages = []string{
	"Generating commit message...",
	"Summarizing staged changes...",
	"Drafting Conventional Commit...",
	"Giving birth to skynet",
	"Analyzing diff hunks...",
	"Composing commit summary...",
	"Buying Sam Altman a new ferrari...",
}

var spinnerStyles = []spinner.Spinner{
	spinner.Line,
	spinner.Dot,
	spinner.MiniDot,
	spinner.Jump,
	spinner.Pulse,
	spinner.Points,
	spinner.Globe,
	spinner.Moon,
	spinner.Monkey,
}

var (
	reasoningStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
	markdownRenderer *glamour.TermRenderer
	activeSpinner    *spinnerHandle
)

var (
	terminalOutput     io.Writer
	terminalOutputOnce sync.Once
)

func getTerminalOutput() io.Writer {
	terminalOutputOnce.Do(func() {
		if runtime.GOOS == "windows" {
			terminalOutput = os.Stderr
			return
		}
		f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
		if err != nil {
			terminalOutput = io.Discard
			return
		}
		terminalOutput = f
	})
	return terminalOutput
}

func StartSpinner(message string, backend string, forwarder SignalForwarder) func() {
	_ = os.Setenv("CLICOLOR_FORCE", "1")
	lipgloss.SetColorProfile(termenv.ANSI)
	markdownRenderer = newMarkdownRenderer()
	p := tea.NewProgram(newSpinnerModel(message, backend, forwarder), tea.WithOutput(getTerminalOutput()))
	handle := &spinnerHandle{
		program:  p,
		reasonCh: make(chan string, 8),
		doneCh:   make(chan struct{}),
	}
	activeSpinner = handle
	done := make(chan struct{})
	go func() {
		_, _ = p.Run()
		close(done)
	}()
	go func() {
		for {
			select {
			case text := <-handle.reasonCh:
				if strings.TrimSpace(text) != "" {
					handle.program.Send(spinnerReasoningMsg(text))
				}
			case <-handle.doneCh:
				return
			}
		}
	}()
	var stopOnce sync.Once
	return func() {
		stopOnce.Do(func() {
			handle.program.Send(spinnerDoneMsg{})
			<-done
			close(handle.doneCh)
			activeSpinner = nil
		})
	}
}

func SendSpinnerReasoning(text string) {
	if activeSpinner == nil {
		return
	}
	select {
	case activeSpinner.reasonCh <- text:
	default:
	}
}

func RandomSpinnerMessage() string {
	if len(spinnerMessages) == 0 {
		return "Generating commit message with Codex..."
	}
	seed := time.Now().UnixNano()
	return spinnerMessages[int(seed%int64(len(spinnerMessages)))]
}

func newSpinnerModel(message string, backend string, forwarder SignalForwarder) spinnerModel {
	s := spinner.New()
	s.Spinner = randomSpinnerStyle()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return spinnerModel{spinner: s, message: message, backend: backend, start: time.Now(), forwarder: forwarder}
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerDoneMsg:
		m.done = true
		return m, tea.Quit
	case spinnerReasoningMsg:
		m.reasoning = string(msg)
		m.reasoningRendered = renderReasoning(m.reasoning)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" && m.forwarder != nil {
			m.forwarder.ForwardSignal(os.Interrupt)
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() string {
	if m.done {
		return "\r\033[2K"
	}
	elapsed := time.Since(m.start).Seconds()
	elapsedStr := fmt.Sprintf("%.1fs", elapsed)
	backendTag := ""
	if m.backend != "" {
		backendTag = " " + reasoningStyle("(using "+m.backend+")")
	}
	if strings.TrimSpace(m.reasoningRendered) != "" {
		return fmt.Sprintf("\n  %s %s%s (%s)\n  %s\n", m.spinner.View(), m.message, backendTag, elapsedStr, m.reasoningRendered)
	}
	return fmt.Sprintf("\n  %s %s%s (%s)\n", m.spinner.View(), m.message, backendTag, elapsedStr)
}

func newMarkdownRenderer() *glamour.TermRenderer {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0),
	)
	if err != nil {
		return nil
	}
	return renderer
}

func renderReasoning(text string) string {
	if markdownRenderer == nil {
		return reasoningStyle(text)
	}
	out, err := markdownRenderer.Render(text)
	if err != nil {
		return reasoningStyle(text)
	}
	return out
}

func randomSpinnerStyle() spinner.Spinner {
	if len(spinnerStyles) == 0 {
		return spinner.Dot
	}
	seed := time.Now().UnixNano()
	return spinnerStyles[int(seed%int64(len(spinnerStyles)))]
}
