# Pin Numbering Truth Document

## The Definitive Truth About Pin Numbering

**Pins are 0-based throughout the entire system**, but **pin 0 should be avoided** due to potential truthy issues in MCP systems.

### Pin Ranges (0-based)
- **Digital Inputs**: 0-7 (8 pins total)
- **Digital Outputs**: 0-15 (16 pins total)  
- **Analog Inputs**: 0-3 (4 pins total)
- **Analog Outputs**: 0-3 (4 pins total)

### Recommended Usage (Avoiding Pin 0)
- **Digital Inputs**: Use pins 1-7 (avoid pin 0)
- **Digital Outputs**: Use pins 1-15 (avoid pin 0)
- **Analog Inputs**: Use pins 1-3 (avoid pin 0)
- **Analog Outputs**: Use pins 1-3 (avoid pin 0)

### Why Avoid Pin 0?
Pin 0 exists and is technically valid, but should be avoided due to potential "truthy" issues in the MCP (Model Context Protocol) system where pin 0 might be interpreted as false/null/undefined in certain contexts.

### Digital Output Control
Digital outputs use separate tools for binary control:
- `set_digital_output` - Sets pin to HIGH/TRUE (only requires pin parameter)
- `unset_digital_output` - Sets pin to LOW/FALSE (only requires pin parameter)

This simplifies control by eliminating the need for a boolean value parameter.

### Discrete Dispensers
Some dispensers work on discrete pulses rather than continuous flow:
- **Cup Dispenser (Pin 4)**: One cup per set/unset cycle
- **Teabag Dispenser (Pin 5)**: One teabag (2g) per set/unset cycle
- **Sugar Dispenser (Pin 6)**: One sugar portion (7g) per set/unset cycle  
- **Milk Dispenser (Pin 7)**: Exactly 4g per set operation (discrete splash)
  - Each `set_digital_output pin: 7` adds 4g of milk
  - For multiple splashes: set, unset, set again
  - No continuous monitoring needed

### Source of Truth
The definitive source of truth for pin assignments is:
`/Users/richard/mcp/_digital-io/configs/io_labels.json`

This file shows all pins starting from 0, confirming the 0-based nature of the system.

### Tea Making Application
The tea-making rules in `/Users/richard/mcp/.amazonq/rules/tea-making-rules.md` correctly use:
- Digital outputs 1-15 (avoiding pin 0)
- Analog inputs 1-3 (avoiding pin 0)
- Set/unset pattern for digital outputs

### Code Implementation
- All internal arrays are 0-based
- MCP tools accept 0-based pin numbers directly (no conversion)
- Tool descriptions warn about avoiding pin 0
- Error messages show correct 0-based ranges
- Pin validation allows 0 but recommends avoiding it
- Digital outputs use separate set/unset tools for simplicity

This document serves as the definitive reference to prevent future confusion about pin numbering.
