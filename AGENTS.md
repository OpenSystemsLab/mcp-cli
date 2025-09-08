# Agent Instructions for mcp-cli

This document provides instructions for AI agents working on the `mcp-cli` codebase.

## Project Overview

`mcp-cli` is a command-line tool for testing Model Context Protocol (MCP) servers. It's written in Go and uses `cobra` for the command-line interface and `bubbletea` for the terminal user interface (TUI).

The primary goal of this tool is to provide a user-friendly way to interact with MCP servers, select tools, provide arguments, and view results, all within the terminal.

## Tech Stack

-   **Go:** The programming language used for the project.
-   **Cobra:** A library for creating powerful modern CLI applications in Go.
-   **Bubble Tea:** A framework for building terminal user interfaces. It follows The Elm Architecture (Model-View-Update).
-   **Lip Gloss:** A library for styling terminal output, used for the TUI layout.
-   **Bubbles:** A library of ready-to-use components for Bubble Tea applications (e.g., `list`, `textinput`, `viewport`).

## How to Build and Run

-   **Build:** To build the executable, run:
    ```sh
    go build
    ```
    This will create an `mcp-cli` binary in the root directory.

-   **Run:** The application has three main commands:
    -   `mcp-cli stdio "<command>"`
    -   `mcp-cli sse <url>`
    -   `mcp-cli http <url>`

-   **Debugging:** The `-v` or `--verbose` flag enables logging to a `debug.log` file. This is useful for debugging the TUI's behavior.

## Code Structure

All the application logic is contained in `main.go`.

### Key Components

-   **`AppModel`:** This is the main struct that holds the state of the application. It includes the current view, the list of tools, the input fields, the debug log, and the viewport for the debug panel.

-   **`initialModel`:** This function initializes the `AppModel` when the application starts.

-   **`Update` function:** This is the heart of the Bubble Tea application. It handles all incoming messages (key presses, window resize events, tool results) and updates the model accordingly. It follows a switch-case structure to handle different message types and application states.

-   **`View` function:** This function is responsible for rendering the TUI based on the current state of the `AppModel`. It uses `lipgloss` to create a two-column layout, with the main content on the left and the debug panel on the right.

-   **`logf` function:** A helper function on `AppModel` to append messages to the debug log and update the debug viewport. Use this function for all logging that should be visible in the debug panel.

### TUI Views

The application has three main views, represented by the `viewState` enum:

1.  `toolSelectionView`: The initial view that shows a list of available tools.
2.  `argumentInputView`: The view for entering arguments for a selected tool.
3.  `resultView`: The view for displaying the result of a tool call.

The debug panel is always visible on the right side of the screen.

## Development Workflow

1.  **Understand the Request:** Carefully read the user's request and identify the required changes.
2.  **Explore the Code:** Use `read_file` to examine `main.go` and understand the current implementation.
3.  **Formulate a Plan:** Create a clear, step-by-step plan to implement the changes.
4.  **Implement Changes:** Use `replace_with_git_merge_diff` or other tools to modify the code.
5.  **Test:** After making changes, run `go build` to ensure the code compiles.
6.  **Review and Submit:** Use `request_code_review` to get feedback on your changes before submitting.

## Important Considerations

-   **State Management:** The `AppModel` is the single source of truth for the application's state. All changes to the UI should be a result of changes to the `AppModel`.
-   **Pointer Receivers:** Methods on `AppModel` that modify its state (like `Update` and `logf`) must have a pointer receiver (`*AppModel`) to ensure the changes are applied to the original model.
-   **Command Handling:** The `Update` function returns a `tea.Cmd`, which represents a command to be executed by the Bubble Tea runtime (e.g., waiting for a tool result). Be sure to handle and batch commands correctly.
-   **Styling:** Use `lipgloss` for all TUI styling.
-   **Dependencies:** Use `go mod tidy` to manage dependencies.
-   **`.gitignore`:** The `mcp-cli` executable is ignored by `.gitignore`.
