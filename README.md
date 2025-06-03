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

## Tools and Features

The MCP application provides several tools that extend Amazon Q's capabilities:

### 1. Calculator

A calculator tool that can perform basic arithmetic operations.

**Example Usage:**
```bash
./bin/mcp "calculate 2 + 2"
```

**Amazon Q Usage:**
```
Calculate the square root of 144
```

### 2. Prompt Registry

A tool to manage and retrieve prompts from a local prompt registry. Prompts are stored in `~/.mcp/prompts` directory.

**Example Usage:**
```bash
# List all available prompts
./bin/mcp "list_prompts"

# Get a specific prompt by ID
./bin/mcp "get_prompt my-prompt-id"
```

**Amazon Q Usage:**
```
Show me the available prompts in the prompt registry
Get the prompt with ID 'code-review'
```

**How Amazon Q Uses the Prompt Registry:**

Amazon Q can use the prompt registry to access pre-defined prompts for specific tasks. When a user asks Amazon Q to perform a task that matches a stored prompt, Amazon Q can:

1. Retrieve the prompt template from the registry
2. Fill in any variables based on the user's request
3. Use the completed prompt to generate a more accurate and consistent response

This is particularly useful for specialized tasks like code reviews, documentation generation, or domain-specific queries where a carefully crafted prompt can significantly improve the quality of responses.

### 3. Rule Creator

A tool to create and manage development standard rules for different tools.

**Example Usage:**
```bash
# Create a new rule
./bin/mcp "create_rule amazonq my-rule \"Description of the rule\" \"*.go\" true \"Rule content here\""

# List rules for a specific tool
./bin/mcp "list_rules amazonq"
```

**Amazon Q Usage:**
```
Create a new rule for Amazon Q that enforces proper error handling in Go files
List all the rules for the Cursor tool
```

### 4. Rules Processor

A tool to process files against development standard rules.

**Example Usage:**
```bash
# Process a file against rules
./bin/mcp "process_rules /path/to/registry.json /path/to/file.go"

# Get content of a specific rule
./bin/mcp "get_rule_content rule-id /path/to/registry.json"
```

**Amazon Q Usage:**
```
Check if my main.go file follows our development standards
Show me the content of the Go error handling rule
```

### 5. Google Search

A tool to perform Google searches and return the top results.

**Example Usage:**
```bash
# Search with default number of results (5)
./bin/mcp "googlesearch \"quantum computing advances 2025\""

# Search with specified number of results
./bin/mcp "googlesearch \"quantum computing advances 2025\" 10"
```

**Amazon Q Usage:**
```
Search Google for recent advances in quantum computing
Find the top 10 results for sustainable energy solutions
```

### 6. Wikipedia Image

A tool to search for images on Wikipedia.

**Example Usage:**
```bash
# Search for an image with default size
./bin/mcp "wikipediaimage \"Albert Einstein\""

# Search for an image with specified size
./bin/mcp "wikipediaimage \"Albert Einstein\" 800"
```

**Amazon Q Usage:**
```
Find a Wikipedia image of Albert Einstein
Get a large image of the Eiffel Tower from Wikipedia
```

### 7. Wikipedia Image Save

A tool to search for images on Wikipedia and save them to disk.

**Example Usage:**
```bash
./bin/mcp "wikipediaimagesave \"Albert Einstein\" 500 /path/to/save/einstein.jpg"
```

**Amazon Q Usage:**
```
Save a Wikipedia image of the Mona Lisa to my desktop
Download a picture of Mount Everest from Wikipedia to my documents folder
```

## Configuration for Amazon Q

To configure Amazon Q to use this MCP application, add the following to your `~/.aws/amazonq/mcp.json` file:

```json
{
  "mcpServers": {
    "mcp": {
      "command": "/path/to/mcp",
      "description": "MCP server providing various utility tools",
      "tools": {
        "calculator": {
          "description": "A calculator tool that can perform basic arithmetic operations"
        },
        "prompt_registry": {
          "description": "A tool to manage and retrieve prompts from the prompt registry"
        },
        "rule_creator": {
          "description": "A tool to create and manage development standard rules"
        },
        "rules_processor": {
          "description": "A tool to process files against development standard rules"
        },
        "google_search": {
          "description": "A tool to perform Google searches and return the top results"
        },
        "wikipedia_image": {
          "description": "A tool to search for images on Wikipedia"
        },
        "wikipedia_image_save": {
          "description": "A tool to save Wikipedia images to disk"
        }
      }
    }
  }
}
```

## Development

This project follows the standard Go project layout and coding standards.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
