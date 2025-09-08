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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/jsonschema"
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
		session, err := client.Connect(ctx, transport)
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
		session, err := client.Connect(ctx, transport)
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
		session, err := client.Connect(ctx, transport)
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

type model struct {
	state        viewState
	ctx          context.Context
	session      *mcp.ClientSession
	toolList     list.Model
	argInputs    []textinput.Model
	argOrder     []string
	argFocus     int
	selectedTool *mcp.Tool
	result       string
	err          error
}

func initialModel(ctx context.Context, session *mcp.ClientSession) model {
	tools, err := session.Tools(ctx, nil).GetAll()
	if err != nil {
		return model{err: err}
	}

	items := []list.Item{}
	for _, tool := range tools {
		items = append(items, item{title: tool.Name, desc: tool.Description, tool: tool})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select a tool to execute"

	return model{
		state:    toolSelectionView,
		ctx:      ctx,
		session:  session,
		toolList: l,
	}
}

type item struct {
	title, desc string
	tool        *mcp.Tool
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if verbose {
			log.Printf("Key pressed: %s", msg.String())
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

		switch m.state {
		case toolSelectionView:
			return m.updateToolSelectionView(msg)
		case argumentInputView:
			return m.updateArgumentInputView(msg)
		case resultView:
			return m.updateResultView(msg)
		}

	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		m.toolList.SetSize(msg.Width-h, msg.Height-v)
	}

	return m, nil
}

func (m model) updateToolSelectionView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.toolList, cmd = m.toolList.Update(msg)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEnter {
			selectedItem := m.toolList.SelectedItem().(item)
			m.selectedTool = selectedItem.tool

			if m.selectedTool.InputSchema != nil && len(m.selectedTool.InputSchema.Properties) > 0 {
				if verbose {
					log.Println("State change: toolSelectionView -> argumentInputView")
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
					log.Println("No arguments needed, calling tool directly")
				}
				return m, m.callTool
			}
		}
	}

	return m, cmd
}

func (m model) updateArgumentInputView(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if keyMsg.Type == tea.KeyEnter {
		if m.argFocus == len(m.argInputs)-1 {
			if verbose {
				log.Println("Last argument input, calling tool")
			}
			return m, m.callTool
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

func (m model) updateResultView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEnter {
			if verbose {
				log.Println("State change: resultView -> toolSelectionView")
			}
			return initialModel(m.ctx, m.session), nil
		}
	}
	return m, nil
}


func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress ctrl+c to quit.", m.err)
	}

	switch m.state {
	case toolSelectionView:
		return m.toolList.View()
	case argumentInputView:
		var b strings.Builder
		b.WriteString(fmt.Sprintf("Enter arguments for %s:\n\n", m.selectedTool.Name))

		for i, name := range m.argOrder {
			b.WriteString(name + "\n")
			b.WriteString(m.argInputs[i].View())
			b.WriteString("\n\n")
		}
		b.WriteString("\nPress Enter to submit, Tab to switch fields, Esc to quit.")
		return b.String()
	case resultView:
		return fmt.Sprintf("Tool Result:\n\n%s\n\nPress Enter to return to tool selection.", m.result)
	}

	return ""
}

func (m model) callTool() (tea.Model, tea.Cmd) {
	args := make(map[string]any)
	for i, name := range m.argOrder {
		args[name] = m.argInputs[i].Value()
	}

	if verbose {
		log.Printf("Calling tool '%s' with args: %v", m.selectedTool.Name, args)
	}

	params := &mcp.CallToolParams{
		Name:      m.selectedTool.Name,
		Arguments: args,
	}
	result, err := m.session.CallTool(m.ctx, params)
	if err != nil {
		m.err = err
		return m, nil
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

	if verbose {
		log.Println("State change: argumentInputView -> resultView")
	}
	m.state = resultView
	m.result = resultStr.String()
	return m, nil
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
