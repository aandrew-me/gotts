package cmd

import (
	"fmt"
	"path"
	"strings"

	"os"

	"github.com/aandrew-me/gotts/tts"
	"github.com/spf13/cobra"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func init() {
	rootCmd.AddCommand(interactiveCmd)
}

var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Interative mode",
	Run: func(cmd *cobra.Command, args []string) {
		voices := tts.Voices // your imported voices slice

		p := tea.NewProgram(initialModel(voices))
		if err := p.Start(); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	},
}

type voiceItem struct {
	tts.Voice
}

func (v voiceItem) Title() string { return v.Name }
func (v voiceItem) Description() string {
	return fmt.Sprintf("%s | %s | %s", v.Gender, v.Language, v.Country)
}
func (v voiceItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s %s", v.Name, v.Gender, v.Language, v.Country)
}

type state int

const (
	stateSelectingVoice state = iota
	stateTypingText
	stateProcessing
)

type ttsDoneMsg struct {
	success  bool
	text     string
	filename string
}

type chatMessage struct {
	Role   string // "user" | "system"
	Text   string
	Status string // optional: "processing", "done", "failed"
}

type model struct {
	state       state
	allItems    []list.Item
	filtered    []list.Item
	list        list.Model
	searchInput textinput.Model

	selected  voiceItem
	textInput textinput.Model

	messages []chatMessage
}

func initialModel(voices []tts.Voice) model {
	// Wrap voices
	items := make([]list.Item, len(voices))
	for i, v := range voices {
		items[i] = voiceItem{v}
	}

	search := textinput.New()
	search.Placeholder = " Search voices"
	search.CharLimit = 32
	search.Width = 30

	listDelegate := list.NewDefaultDelegate()
	listDelegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true)
	listDelegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F1F1F1")).
		Background(lipgloss.Color("#7D56F4"))

	l := list.New(items, listDelegate, 50, 24)
	l.Title = "Select a Voice"
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	km := l.KeyMap

	// Disable unwanted bindings
	km.Quit.SetEnabled(false)     // no "q" to quit
	km.CursorUp.SetKeys("up")     // only ↑
	km.CursorDown.SetKeys("down") // only ↓

	textInput := textinput.New()
	textInput.Placeholder = "Type text to speak"
	textInput.CharLimit = 10000
	textInput.Width = 60

	return model{
		state:       stateSelectingVoice,
		allItems:    items,
		filtered:    items,
		list:        l,
		searchInput: search,
		textInput:   textInput,
		messages:    []chatMessage{},
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	m.searchInput.Focus()

	switch m.state {
	case stateSelectingVoice:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.searchInput.SetValue("")
				m.filtered = m.allItems
				m.list.SetItems(m.filtered)
				m.searchInput.Blur()
			case "enter":
				if selected, ok := m.list.SelectedItem().(voiceItem); ok {
					m.selected = selected
					m.state = stateTypingText
					m.textInput.Focus()
				}
			}
		}

		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)

		// Filter items by search query
		query := strings.ToLower(m.searchInput.Value())
		if query == "" {
			m.filtered = m.allItems
		} else {
			m.filtered = []list.Item{}
			for _, it := range m.allItems {
				if strings.Contains(strings.ToLower(it.FilterValue()), query) {
					m.filtered = append(m.filtered, it)
				}
			}
		}
		m.list.SetItems(m.filtered)

		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

	case stateTypingText:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				// Go back to voice selection
				m.state = stateSelectingVoice
				m.searchInput.SetValue("")
				m.filtered = m.allItems
				m.list.SetItems(m.filtered)
				m.list.Select(0)
			case "enter":
				text := m.textInput.Value()
				m.textInput.SetValue("")
				if strings.TrimSpace(text) == "" {
					// ignore empty input
					return m, nil
				}

				// Append user message and a processing system message to the history
				m.messages = append(m.messages, chatMessage{
					Role: "user",
					Text: text,
				})
				m.messages = append(m.messages, chatMessage{
					Role:   "system",
					Text:   "Processing...",
					Status: "processing",
				})

				// switch to processing state
				
				tmpPath := path.Join(os.TempDir(), "generated.mp3")

				// Keep the current text in the input (so the user can edit) — do not clear it.
				// Start the TTS work in a tea.Cmd and return a ttsDoneMsg when finished.
				return m, func() tea.Msg {
					err := tts.GenerateAudio(text, m.selected.Name, tmpPath)
					if err != nil {
						// failed
						return ttsDoneMsg{
							success:  false,
							text:     text,
							filename: "",
						}
					}

					// play in background
					go func() {
						if err := tts.PlayAudio(tmpPath); err != nil {
							fmt.Println("Error playing audio:", err)
						}
					}()


					return ttsDoneMsg{
						success:  true,
						text:     text,
						filename: path.Join(tmpPath),
					}
				}
			}
		}

		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)

	case stateProcessing:
		switch msg := msg.(type) {
		case ttsDoneMsg:
			// Update the most recent system processing message to success/failure
			for i := len(m.messages) - 1; i >= 0; i-- {
				if m.messages[i].Role == "system" && m.messages[i].Status == "processing" {
					if msg.success {
						m.messages[i].Text = "Generated"
						m.messages[i].Status = "done"
					} else {
						m.messages[i].Text = "Failed to generate audio"
						m.messages[i].Status = "failed"
					}
					break
				}
			}

			m.state = stateTypingText
			m.textInput.Focus()

		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderMessages() string {
	var b strings.Builder
	if len(m.messages) == 0 {
		return ""
	}
	for _, mm := range m.messages {
		prefix := ""
		if mm.Role == "user" {
			prefix = "You: "
		} else {
			// system
			switch mm.Status {
			case "processing":
				prefix = ""
			case "done":
				prefix = "✓ "
			case "failed":
				prefix = "✗ "
			default:
				prefix = "System: "
			}
		}
		b.WriteString(prefix)
		b.WriteString(mm.Text)
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) View() string {
	switch m.state {
	case stateSelectingVoice:
		searchBox := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Padding(0, 1).
			Render(m.searchInput.View())
		return fmt.Sprintf("%s\n\n%s\n\n%s",
			"Use arrow keys to navigate, Enter to select voice, Esc to clear search",
			searchBox,
			m.list.View(),
		)

	case stateTypingText:
		history := m.renderMessages()
		return fmt.Sprintf(
			"Selected Voice: %s\n\n%s\nEnter text to speak (Esc to cancel):\n\n%s\n\nPress Enter to speak.\n",
			m.selected.Name,
			history,
			m.textInput.View(),
		)

	case stateProcessing:
		// while processing, show the history (which contains a "Processing..." system message)
		history := m.renderMessages()
		return fmt.Sprintf(
			"Selected Voice: %s\n\n%s\n%s\n",
			m.selected.Name,
			history,
			"",
		)
	}
	return ""
}
