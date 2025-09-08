package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "mcp-cli",
	Short: "A CLI tool for testing MCP servers",
	Long:  `A comprehensive CLI tool for testing MCP servers, including stdio, sse, and streamable http protocols.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

var stdioCmd = &cobra.Command{
	Use:   "stdio [command]",
	Short: "Connect to an MCP server over stdio",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		command := args[0]
		if verbose {
			log.Printf("Command: %s", command)
		}

		ctx := context.Background()
		client := mcp.NewClient(&mcp.Implementation{Name: "mcp-cli", Version: "v0.1.0"}, nil)

		cmdParts := strings.Fields(command)
		transport := &mcp.CommandTransport{Command: exec.Command(cmdParts[0], cmdParts[1:]...)}
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			log.Fatalf("Failed to connect to stdio server: %v", err)
		}
		defer session.Close()

		if verbose {
			log.Println("Connected to stdio server")
		}

		handleSession(ctx, session)
	},
}

var sseCmd = &cobra.Command{
	Use:   "sse [url]",
	Short: "Connect to an MCP server over SSE",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		if verbose {
			log.Printf("URL: %s", url)
		}

		ctx := context.Background()
		client := mcp.NewClient(&mcp.Implementation{Name: "mcp-cli", Version: "v0.1.0"}, nil)

		transport := &mcp.SSEClientTransport{Endpoint: url}
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			log.Fatalf("Failed to connect to SSE server: %v", err)
		}
		defer session.Close()

		if verbose {
			log.Println("Connected to SSE server")
		}

		handleSession(ctx, session)
	},
}

var httpCmd = &cobra.Command{
	Use:   "http [url]",
	Short: "Connect to an MCP server over streamable HTTP",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		if verbose {
			log.Printf("URL: %s", url)
		}

		ctx := context.Background()
		client := mcp.NewClient(&mcp.Implementation{Name: "mcp-cli", Version: "v0.1.0"}, nil)

		transport := &mcp.StreamableClientTransport{Endpoint: url}
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			log.Fatalf("Failed to connect to streamable HTTP server: %v", err)
		}
		defer session.Close()

		if verbose {
			log.Println("Connected to streamable HTTP server")
		}

		handleSession(ctx, session)
	},
}

// -- Bubble Tea TUI -----------------------------------------------------------

type viewState int

const (
	toolSelectionView viewState = iota
	argumentInputView
	resultView
)

type AppModel struct {
	state         viewState
	ctx           context.Context
	session       *mcp.ClientSession
	toolList      list.Model
	argInputs     []textinput.Model
	argOrder      []string
	argFocus      int
	selectedTool  *mcp.Tool
	tools         []*mcp.Tool
	result        string
	err           error
	log           []string
	width         int
	height        int
	debugViewport viewport.Model
}

func initialModel(ctx context.Context, session *mcp.ClientSession) *AppModel {
	var err error
	var tools []*mcp.Tool

	// Iterate over the tools using range
	for tool, iterErr := range session.Tools(ctx, nil) {
		if iterErr != nil {
			err = iterErr
			break
		}
		tools = append(tools, tool)
	}

	if err != nil {
		return &AppModel{err: err}
	}

	items := []list.Item{}
	for _, tool := range tools {
		items = append(items, item{title: tool.Name, desc: tool.Description, tool: tool})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select a tool to execute"

	vp := viewport.New(1, 1) // Initial size, will be updated on WindowSizeMsg
	vp.SetContent("Debug log will appear here...")

	return &AppModel{
		state:         toolSelectionView,
		ctx:           ctx,
		session:       session,
		toolList:      l,
		tools:         tools,
		debugViewport: vp,
	}
}

type item struct {
	title, desc string
	tool        *mcp.Tool
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func (m *AppModel) logf(format string, a ...any) {
	m.log = append(m.log, fmt.Sprintf(format, a...))
	m.debugViewport.SetContent(strings.Join(m.log, "\n"))
	m.debugViewport.GotoBottom()
}

func (m AppModel) Init() tea.Cmd {
	return nil
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case toolResult:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if verbose {
			m.logf("Tool result received")
		}
		m.logf("Result:\n%s", msg.result)
		m.result = msg.result
		m.state = resultView
	case tea.KeyMsg:
		if verbose {
			m.logf("Key pressed: %s", msg.String())
		}
		switch msg.Type {
		case tea.KeyEsc:
			m.state = toolSelectionView
			return m, nil
		case tea.KeyCtrlC:
			return m, tea.Quit
		}

		switch m.state {
		case toolSelectionView:
			var model tea.Model
			model, cmd = m.updateToolSelectionView(msg)
			cmds = append(cmds, cmd)
			m.debugViewport, cmd = m.debugViewport.Update(msg)
			cmds = append(cmds, cmd)
			return model, tea.Batch(cmds...)
		case argumentInputView:
			var model tea.Model
			model, cmd = m.updateArgumentInputView(msg)
			cmds = append(cmds, cmd)
			m.debugViewport, cmd = m.debugViewport.Update(msg)
			cmds = append(cmds, cmd)
			return model, tea.Batch(cmds...)
		case resultView:
			var model tea.Model
			model, cmd = m.updateResultView(msg)
			cmds = append(cmds, cmd)
			m.debugViewport, cmd = m.debugViewport.Update(msg)
			cmds = append(cmds, cmd)
			return model, tea.Batch(cmds...)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		debugWidth := m.width / 3
		m.debugViewport.Width = debugWidth
		m.debugViewport.Height = m.height - 2
	}

	m.debugViewport, cmd = m.debugViewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *AppModel) updateToolSelectionView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.toolList, cmd = m.toolList.Update(msg)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEnter {
			selectedItem := m.toolList.SelectedItem().(item)
			m.selectedTool = selectedItem.tool

			if m.selectedTool.InputSchema != nil && len(m.selectedTool.InputSchema.Properties) > 0 {
				if verbose {
					m.logf("State change: toolSelectionView -> argumentInputView")
				}
				m.state = argumentInputView
				m.argInputs = []textinput.Model{}
				m.argOrder = []string{}
				for name := range m.selectedTool.InputSchema.Properties {
					m.argOrder = append(m.argOrder, name)
				}
				sort.Strings(m.argOrder)

				for _, name := range m.argOrder {
					prop := m.selectedTool.InputSchema.Properties[name]
					ti := textinput.New()
					ti.Placeholder = prop.Description
					ti.Focus()
					ti.CharLimit = 256
					ti.Width = 50
					m.argInputs = append(m.argInputs, ti)
				}
				m.argInputs[0].Focus()
			} else {
				if verbose {
					m.logf("No arguments needed, calling tool directly")
				}
				return m.callTool()
			}
		}
	}

	return m, cmd
}

func (m *AppModel) updateResultView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEnter {
			if verbose {
				m.logf("State change: resultView -> argumentInputView")
			}
			m.state = argumentInputView
		}
	}
	return m, nil
}

func (m *AppModel) updateArgumentInputView(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if keyMsg.Type == tea.KeyEnter {
		if m.argFocus == len(m.argInputs)-1 {
			if verbose {
				m.logf("Last argument input, calling tool")
			}
			return m.callTool()
		}
		m.argFocus++
		for i := range m.argInputs {
			if i == m.argFocus {
				m.argInputs[i].Focus()
			} else {
				m.argInputs[i].Blur()
			}
		}
		return m, nil
	}

	if keyMsg.Type == tea.KeyTab {
		m.argFocus = (m.argFocus + 1) % len(m.argInputs)
		for i := range m.argInputs {
			if i == m.argFocus {
				m.argInputs[i].Focus()
			} else {
				m.argInputs[i].Blur()
			}
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.argInputs[m.argFocus], cmd = m.argInputs[m.argFocus].Update(msg)
	return m, cmd
}

func (m AppModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress ctrl+c to quit.", m.err)
	}

	mainWidth := m.width / 3
	//debugWidth := m.width - mainWidth

	var mainContent strings.Builder
	switch m.state {
	case toolSelectionView:
		m.toolList.SetSize(mainWidth-2, m.height-2)
		mainContent.WriteString(m.toolList.View())
	case argumentInputView:
		var b strings.Builder
		b.WriteString(fmt.Sprintf("Enter arguments for %s:\n\n", m.selectedTool.Name))

		for i, name := range m.argOrder {
			b.WriteString(name + "\n")
			b.WriteString(m.argInputs[i].View())
			b.WriteString("\n\n")
		}
		b.WriteString("\nPress Enter to submit, Tab to switch fields, Esc to go back to tool selection.")
		mainContent.WriteString(b.String())
	case resultView:
		mainContent.WriteString(fmt.Sprintf("Tool Result:\n\n%s\n\nPress Enter to return to the argument input.", m.result))
	}

	mainPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(mainWidth).
		Height(m.height - 2).
		Render(mainContent.String())

	debugPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(m.debugViewport.Width).
		Height(m.debugViewport.Height).
		Render(m.debugViewport.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, mainPanel, debugPanel)
}

// toolResult represents the result of a tool call
type toolResult struct {
	result string
	err    error
}

// callToolCmd returns a tea.Cmd that calls the tool
func (m *AppModel) callToolCmd() tea.Cmd {
	return func() tea.Msg {
		args := make(map[string]any)
		for i, name := range m.argOrder {
			args[name] = m.argInputs[i].Value()
		}

		prettyArgs, err := json.MarshalIndent(args, "", "  ")
		if err != nil {
			m.logf("Error marshalling args: %v", err)
		}
		m.logf("Calling tool '%s' with args:\n%s", m.selectedTool.Name, string(prettyArgs))

		params := &mcp.CallToolParams{
			Name:      m.selectedTool.Name,
			Arguments: args,
		}
		result, err := m.session.CallTool(m.ctx, params)
		if err != nil {
			return toolResult{err: err}
		}

		var resultStr strings.Builder
		if result.IsError {
			resultStr.WriteString("Error:\n")
		}

		for _, content := range result.Content {
			switch c := content.(type) {
			case *mcp.TextContent:
				var obj any
				if json.Unmarshal([]byte(c.Text), &obj) == nil {
					prettyJSON, err := json.MarshalIndent(obj, "", "  ")
					if err == nil {
						resultStr.WriteString(string(prettyJSON))
						continue
					}
				}
				resultStr.WriteString(c.Text)
			default:
				prettyJSON, err := json.MarshalIndent(c, "", "  ")
				if err != nil {
					resultStr.WriteString(fmt.Sprintf("Unsupported content type: %T", c))
				} else {
					resultStr.WriteString(string(prettyJSON))
				}
			}
		}

		return toolResult{result: resultStr.String()}
	}
}

func (m *AppModel) callTool() (tea.Model, tea.Cmd) {
	return m, m.callToolCmd()
}

func handleSession(ctx context.Context, session *mcp.ClientSession) {
	if verbose {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}
	p := tea.NewProgram(initialModel(ctx, session), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}

func main() {
	rootCmd.AddCommand(stdioCmd)
	rootCmd.AddCommand(sseCmd)
	rootCmd.AddCommand(httpCmd)
	Execute()
}
