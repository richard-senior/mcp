# Rules for this project (mcp)
This project is an implementation of an MCP tools and resources server designed to be compatible with Amazon Q Chat.

# Tool creation
- A tool is a single go file stored in the ./pkg/tools directory
- A tool has the following structure:
  - A function such as :
    ```go
    func NewMCPTool() protocol.Tool {
        return protocol.Tool{
            Name: "new_mcp_tool",
            Description: `
            A thorough description of this tool including what user input would invoke usage of the tool,
            and what the tool returns
            `,
            InputSchema: protocol.InputSchema{
                Type: "object",
                Properties: map[string]protocol.ToolProperty{
                    "property1": {
                        Type:        "string",
                        Description: "An example of something that the client should pass as a parameter",
                    },
                    "property2": {
                        Type: "integer",
                        Description: `
                            Another more thorough example of something that the client should pass as a parameter
                            using multiline strings.
                        `,
                    },
                },
                Required: []string{"property1"},
            },
        }
    }
    ```
    Where the function name is changed appropriately from NewMCPTool to something like 'NewGoogleSearchTool' etc.
    This function is used when registering the tool with the MCP application and should be near the top of the file
    - One or more public function such as :
    ```go
    func HandleNewTool(params any) (any, error) {
        // parse params
        // carry out necessary operations
        // return a response structure or any error etc.
    }
    ```
    Again the name of the function should be changed appropriately to HandleNEWTOOLNAMETool etc.
    This function should be near the bottom of the file.
    This is the function that will be registered with the MCP application
    If this particular tool has multiple possible invocations then this function
    should handle calling the correct sub function etc.
- The new tool should be regsitered with the MCP application by adding a new section to
  pkg/server/server.go#RegisterDefaultTools()

# Tool Usage
- Use these tools silently, that is, do not echo out the "Using tool:" blocks in responses.
- The MCP server is not a 'server' (like a http server that stays resident) in that it does not need to be called to initialise it. Instead simply calling it will carry out any desired operations


# Debugging
- You can build the application using the build.sh file
- You can invoke the application in a shell using /Users/richard/mcp/mcp
- You can see the output of the application at /tmp/mcp.log
- You can see amazon q chat's mcp config file locations by executing the shell command 'q mcp list'
- You can write a unit test in /test and run test.sh. Observe one of the existing tests to see how