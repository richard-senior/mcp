# MCP CLI Application

This is a CLI application for the Model Context Protocol (MCP) that can be invoked by Amazon Q.

## Overview

MCP is an open protocol that standardizes how applications provide context to LLMs. This CLI application implements the MCP protocol to extend Amazon Q's capabilities by providing additional tools and context.

## Project Structure

- `/cmd/mcp`: Main application entry point
- `/internal`: Private application and library code
  - `/internal/logger`: Logging utilities
  - `/internal/processor`: MCP request processing logic
- `/pkg`: Library code that's ok to use by external applications
- `/api`: OpenAPI/Swagger specs, JSON schema files, protocol definition files
- `/configs`: Configuration file templates or default configs
- `/test`: Additional external test apps and test data

## Building and Running

To build the application:

```bash
./build.sh
```

To build and run the application:

```bash
./run.sh [options]
```

### Command Line Options

- `--debug`: Enable debug logging
- `--input <file>`: Input file path (if not provided, stdin will be used)
- `--output <file>`: Output file path (if not provided, stdout will be used)

## Usage with Amazon Q

This CLI application is designed to be invoked by Amazon Q. When Amazon Q needs additional context or tools, it will call this application with a JSON request on stdin and expect a JSON response on stdout.

Example usage:

```bash
echo '{"query": "example query", "requestId": "123"}' | ./bin/mcp
```

## Development

This project follows the standard Go project layout and coding standards.
