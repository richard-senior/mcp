# Digital I/O Bank Simulator

A Go-based REST API that simulates a USB-connected I/O bank with:
- 8 Digital Inputs (read-only, simulated)
- 16 Digital Outputs (read/write)
- 4 Analog Inputs (read-only, simulated, 0-5V range)
- 4 Analog Outputs (read/write, 0-5V range)

## Features

- **REST API** for simple HTTP requests
- **MCP Server Mode** for direct integration with Amazon Q Chat and other LLMs
- **Web interface** for visual monitoring and control
- **Custom I/O labels** for meaningful pin identification
- **Real-time simulation** of input values that change over time
- **Thread-safe** operations with proper locking
- **Comprehensive logging** with different log levels
- **Graceful shutdown** handling
- **Example clients** in Python and Node.js

## Getting Started

1. Build the project:
   ```bash
   ./build.sh
   ```

2. Run the server in HTTP mode:
   ```bash
   ./run.sh
   ```

3. The server will start on `http://localhost:8080`

4. Open a web browser and navigate to `http://localhost:8080` to access the web interface

## MCP Server Mode (Model Context Protocol)

The project now supports running as an MCP server over STDIO, which allows direct integration with Amazon Q Chat and other LLMs that support the Model Context Protocol.

### Running in MCP Mode

```bash
./digital-io-server -mcp
```

In MCP mode, the server:
- Communicates via STDIO using JSON-RPC
- Redirects all logs to stderr to avoid interfering with JSON-RPC communication
- Provides tools for controlling and monitoring the I/O bank

### Amazon Q Chat Integration

To use with Amazon Q Chat:

1. Create or update your Amazon Q Chat MCP configuration file:
   ```bash
   mkdir -p ~/.aws/amazonq
   ```

2. Create or edit `~/.aws/amazonq/mcp.json`:
   ```json
   {
     "mcpServers": {
       "digital-io": {
         "name": "digital-io",
         "command": "/path/to/digital-io-server",
         "args": ["-mcp"],
         "description": "Digital I/O Bank Simulator - provides tools for controlling and monitoring digital and analog I/O pins",
         "timeout": 5000
       }
     }
   }
   ```

3. Restart Amazon Q Chat to load the new MCP server

### Available MCP Tools

The MCP server provides the following tools:

- `get_digital_input` - Read the state of a digital input pin (0-7) - Pin 0 should be avoided due to MCP truthy issues
- `set_digital_output` - Set a digital output pin to HIGH/TRUE (0-15) - Pin 0 should be avoided due to MCP truthy issues
- `unset_digital_output` - Set a digital output pin to LOW/FALSE (0-15) - Pin 0 should be avoided due to MCP truthy issues  
- `get_digital_output` - Read the current state of a digital output pin (0-15) - Pin 0 should be avoided due to MCP truthy issues
- `get_analog_input` - Read the voltage of an analog input pin (0-3) - Pin 0 should be avoided due to MCP truthy issues
- `get_converted_analog_input` - Get analog input value converted to real-world units (°C, g, etc.)
- `set_analog_output` - Set the voltage of an analog output pin (0-3) - Pin 0 should be avoided due to MCP truthy issues
- `get_analog_output` - Read the current voltage of an analog output pin (0-3) - Pin 0 should be avoided due to MCP truthy issues
- `get_system_status` - Get complete system status including all I/O states and labels

**Important Note**: While pins are 0-based (0-15 for digital outputs, 0-7 for digital inputs, 0-3 for analog), **pin 0 should be avoided** due to potential truthy issues in MCP systems. Use pins 1-15 for digital outputs, 1-7 for digital inputs, and 1-3 for analog I/O.

**Digital Output Control**: Digital outputs use separate tools for setting and unsetting:
- `set_digital_output` - Sets pin to HIGH/TRUE (only requires pin parameter)
- `unset_digital_output` - Sets pin to LOW/FALSE (only requires pin parameter)

**Discrete Dispensers**: Some dispensers work on discrete pulses rather than continuous flow:
- **Cup Dispenser (Pin 4)**: Dispenses one cup per set/unset cycle
- **Teabag Dispenser (Pin 5)**: Dispenses one teabag (2g) per set/unset cycle  
- **Sugar Dispenser (Pin 6)**: Dispenses one sugar portion (7g) per set/unset cycle
- **Milk Dispenser (Pin 7)**: Dispenses exactly 4g of milk per set operation (no unset needed)
  - For multiple splashes: set pin 7, unset pin 7, set pin 7 again
  - Each set operation adds another 4g splash

### Testing MCP Mode

A Python test script is included to verify MCP functionality:

```bash
python3 test_mcp.py
```

This script tests the initialize, tools/list, and tools/call methods.

## REST API Endpoints

### REST Endpoints
- `GET /status` - Get all I/O states
- `GET /digital/input/{pin}` - Get digital input value
- `GET /digital/output/{pin}` - Get digital output value
- `POST /digital/output/{pin}` - Set digital output value
- `GET /analog/input/{pin}` - Get analog input value
- `GET /analog/output/{pin}` - Get analog output value
- `POST /analog/output/{pin}` - Set analog output value
- `GET /labels` - Get all I/O labels
- `POST /labels/{type}/{pin}` - Update an I/O label

## Custom I/O Labels

The simulator supports custom labels for all I/O pins, allowing you to give meaningful names to each input and output:

1. **Editing Labels via Web Interface**:
   - Click the "Edit Labels" button in the web interface
   - Click the edit icon (✎) on any I/O pin
   - Enter a custom name (e.g., "Water Valve", "Temperature Sensor")
   - Click Save

2. **Editing Labels via Configuration File**:
   - Edit the file at `/Users/richard/mcp/_digital-io/configs/io_labels.json`
   - The file contains sections for each I/O type with pin numbers as keys
   - Example:
     ```json
     {
       "digital_outputs": {
         "0": "Water Valve",
         "1": "Heater"
       }
     }
     ```

3. **API for Label Management**:
   - `GET /labels` - Get all labels
   - `POST /labels/{type}/{pin}` - Update a label

Labels are displayed in the web interface and included in the status response.

## REST API Examples

```bash
# Get system status
curl http://localhost:8080/status

# Get digital input 5
curl http://localhost:8080/digital/input/5

# Set digital output 10 to true
curl -X POST http://localhost:8080/digital/output/10 \
  -H "Content-Type: application/json" \
  -d '{"value": true}'

# Get analog input 2
curl http://localhost:8080/analog/input/2

# Set analog output 4 to 2.5V
curl -X POST http://localhost:8080/analog/output/4 \
  -H "Content-Type: application/json" \
  -d '{"value": 2.5}'

# Update a label
curl -X POST http://localhost:8080/labels/digital_output/0 \
  -H "Content-Type: application/json" \
  -d '{"label": "Water Valve"}'
```

## Example Clients

The project includes example clients in Python, Node.js, and Go to demonstrate how to interact with the API:

### Python Client

```bash
cd examples
python3 python_client.py --demo        # Run demo sequence
python3 python_client.py --monitor 30  # Monitor inputs for 30 seconds
```

### Node.js Client

```bash
cd examples
node nodejs_client.js --demo           # Run demo sequence
node nodejs_client.js --interactive    # Start interactive mode
node nodejs_client.js --monitor 30     # Monitor inputs for 30 seconds
```

### Go Client

```bash
cd examples
go run go_client.go --demo             # Run demo sequence
go run go_client.go --monitor 30       # Monitor inputs for 30 seconds
```

## Pin Ranges

- **Digital Inputs**: 0-7 (8 pins) - **Avoid pin 0 due to MCP truthy issues, use pins 1-7**
- **Digital Outputs**: 0-15 (16 pins) - **Avoid pin 0 due to MCP truthy issues, use pins 1-15**
- **Analog Inputs**: 0-3 (4 pins, 0-5V range) - **Avoid pin 0 due to MCP truthy issues, use pins 1-3**
- **Analog Outputs**: 0-3 (4 pins, 0-5V range) - **Avoid pin 0 due to MCP truthy issues, use pins 1-3**

## Simulation

The simulator automatically updates input values:
- Digital inputs randomly change state (10% chance every 2 seconds)
- Analog inputs have small random variations (±0.1V noise every 2 seconds)
- Output values remain as set until explicitly changed

## Web Interface

The web interface provides:
- Real-time visualization of all I/O states
- Control of digital and analog outputs
- Custom labels for all I/O pins
- Simulation control (start/stop)
- Label editing capabilities

## Development

Run tests:
```bash
./test.sh
```

The project follows standard Go project layout:
- `/cmd` - Main application
- `/internal` - Private application code
- `/pkg` - Public library code
- `/test` - Test files
- `/web` - Web interface files
- `/examples` - Example client applications
- `/configs` - Configuration files (including I/O labels)
