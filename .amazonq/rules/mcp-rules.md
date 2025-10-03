# Rules for this project (mcp)
This project is an implementation of an MCP tools and resources server designed to be compatible with Amazon Q Chat.
*Important* I should not carry out work that is not directly requested. I should stick to doing only what is asked and nothing more.

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


# Debugging
- I can build the application using the build.sh file
- I can invoke the application in a shell using /Users/richard/mcp/mcp
- I can see the output of the application at /tmp/mcp.log
- I can see amazon q chat's mcp config file locations by executing the shell command 'q mcp list'
- I can determine if the MCP tools have loaded by issuing the command : env Q_LOG_LEVEL=trace bash -c '(sleep 10; echo "/tools") | aq'
- I can write a unit test in /test and run test.sh. Observe one of the existing tests to see how to do that.

Generally if the tools exposed by this application are not being loaded by Amazon Q Chat I should:
1) Verify this is the case by issuing the bash command : env Q_LOG_LEVEL=trace bash -c '(sleep 10; echo "/tools") | q chat'
2) Check the MCP tool logs at /tmp/mcp.log
3) Check $TMPDIR/qlog/mcp.log
4) Try to determine the issue
5) Rebuild the MCP application
6) Repeat from 1) until the issue is resolved