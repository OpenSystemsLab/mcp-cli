# MCP-CLI

MCP-CLI is a command-line tool for interacting with and testing Model Context Protocol (MCP) servers. It provides a user-friendly terminal user interface (TUI) for selecting and executing tools offered by an MCP server.

## Features

- **Multiple Transports:** Connect to MCP servers using `stdio`, `sse` (Server-Sent Events), or `http` (streamable HTTP).
- **Interactive TUI:** A terminal user interface built with `bubbletea` that allows you to:
  - Select a tool from a list of available tools.
  - Enter arguments for the selected tool in a form.
  - View the results of the tool execution.
- **Debug Panel:** A scrollable debug panel on the right side of the TUI that shows:
  - Informational logs (key presses, state changes).
  - The arguments sent to the tool in a pretty-printed JSON format.
  - The results of tool calls.
- **Verbose Logging:** Use the `-v` flag to enable verbose logging to a `debug.log` file for troubleshooting.

## Installation

You can install `mcp-cli` using `go install`:

```sh
go install mcp-cli
```

Alternatively, you can build it from source.

## Building from Source

1.  Clone the repository:
    ```sh
    git clone <repository-url>
    cd mcp-cli
    ```
2.  Build the executable:
    ```sh
    go build
    ```
    This will create an `mcp-cli` executable in the current directory.

## Usage

The `mcp-cli` tool has three main commands, one for each transport protocol.

### `stdio`

Connect to an MCP server that communicates over standard input/output.

```sh
mcp-cli stdio --env "VAR=value" --env "ANOTHER_VAR=another_value" "<command-to-start-server>"
```

The `--env` (or `-e`) flag can be used multiple times to pass environment variables to the server process.

**Example:**

```sh
mcp-cli stdio -e "API_KEY=12345" "python /path/to/mcp/server.py"
```

### `sse`

Connect to an MCP server that uses Server-Sent Events (SSE).

```sh
mcp-cli sse --header "Header-Name: header-value" <server-url>
```

The `--header` (or `-H`) flag can be used multiple times to pass custom HTTP headers to the server.

**Example:**

```sh
mcp-cli sse -H "Authorization: Bearer my-token" http://localhost:8080/mcp
```

### `http`

Connect to an MCP server that uses streamable HTTP.

```sh
mcp-cli http --header "Header-Name: header-value" <server-url>
```

The `--header` (or `-H`) flag can be used multiple times to pass custom HTTP headers to the server.

**Example:**

```sh
mcp-cli http -H "Authorization: Bearer my-token" http://localhost:8080/mcp
```

### Global Flags

- `-v`, `--verbose`: Enable verbose logging to `debug.log`.

## TUI Guide

When you connect to an MCP server, you will be presented with a terminal user interface.

-   **Tool Selection View:** A list of available tools. Use the arrow keys to navigate and press `Enter` to select a tool.
-   **Argument Input View:** A form for entering the arguments for the selected tool. Use `Tab` to switch between fields and `Enter` to submit the tool call.
-   **Result View:** Shows the result of the tool execution. Press `Enter` to return to the argument input view for the same tool, allowing you to easily call it again with different arguments.
-   **Debug Panel:** The right-hand panel shows a scrollable log of events, tool calls, and results. Use the up and down arrow keys to scroll through the log.
-   **Navigation:**
    -   `Esc`: Return to the tool selection view.
    -   `Ctrl+C`: Exit the application.
