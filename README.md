# github.com/richard-senior/mcp

A Model Context Protocol (MCP) server implementation for Amazon Q / Claude etc.
A single cross platform executable containing multiple tools/resources/prompts etc.

## Project Structure

This project follows the standard Go project layout:

- `/cmd` - Main application entry point
- `/internal` - Private application and library code
- `/pkg` - Library code that's ok to use by external applications
- `/configs` - Configuration files
- `/test` - Test applications and test data

## Getting Started

1. Build the project:
   ```bash
   ./build.sh
   ```
2. Configure Amazon Q Chat
   1. Create directory : ~/.aws/amazonq
   2. Create file : ~/.aws/amazonq/mcp.json
      ```json
      {
      "mcpServers": {
         "mcp": {
            "name": "mcp",
            "command": "$$ The absolute path to compiled MCP binary $$",
            "description": "Custom MCP server",
            "timeout": 2000
         }
      }
      }
      ```
   3. Stop and restart Q chat
   4. In the Q chat prompt use /tools
      This should show the list of tools published by MCP
   5. Ask Q chat to 'get an image of Elvis Presley into the local directory'

## Current tools
### Google Search
Using a private API key (which you should change in config)
The tool can get the description, link and name of the top 'n' links
found by google search for the search term.
For example ask Q Chat 'Please use google to find information about Elvis Presley'
### Html to Markdown
LLM's prefer markdown as a format, so we need a tool to convert html to markdown
This allows the LLM to 'precis' a web page.
For example ask Q Chat 'please precis the information in https://en.wikipedia.org/wiki/Elvis_Presley'
or 'Use the web to get information about Elvis Presley'
### Image Finder
Uses Wikipedia to get binary images (photo's etc) by search term
for example ask Q Chat to 'get an image of Elvis Presley into the local directory'

## Development

This project is in the initial setup phase.
