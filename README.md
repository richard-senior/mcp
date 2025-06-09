# github.com/richard-senior/mcp

A Model Context Protocol (MCP) server implementation for Amazon Q.

## Project Structure

This project follows the standard Go project layout:

- `/cmd` - Main application entry point
- `/internal` - Private application and library code
- `/pkg` - Library code that's ok to use by external applications
- `/api` - API specifications and protocol definitions
- `/web` - Web application specific components
- `/configs` - Configuration files
- `/test` - Test applications and test data

## Getting Started

1. Build the project:
   ```
   ./build.sh
   ```

2. Run the application:
   ```
   ./run.sh
   ```

## Features

This MCP server will provide tools for Amazon Q to use, such as:

- An image downloading tool using Wikipedia thumbnails
- A Google Search tool providing the link, title and description of the top 'n' results
- An HTML to Markup tool for getting a 'precis' of a web page

## Development

This project is in the initial setup phase.
