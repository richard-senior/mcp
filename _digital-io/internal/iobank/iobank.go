package iobank

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/richard-senior/mcp/_digital-io/internal/logger"
)

// MCPMessage represents an MCP message received by the system
type MCPMessage struct {
	Timestamp time.Time `json:"timestamp"`
	ToolName  string    `json:"tool_name"`
	Message   string    `json:"message"`
}

// IOBank represents a simulated I/O bank with digital and analog ports
type IOBank struct {
	mu sync.RWMutex

	// Digital I/O - 8 inputs and 16 outputs
	digitalInputs  [8]bool
	digitalOutputs [16]bool

	// Analog I/O - 4 inputs and 4 outputs (0-3)
	// Values range from 0.0 to 5.0 (representing 0-5V)
	analogInputs  [4]float64
	analogOutputs [4]float64

	// MCP message tracking
	mcpMessages    []MCPMessage
	lastMCPMessage *MCPMessage

	// Simulation parameters
	simulationRunning bool
	stopChan          chan bool
}

// NewIOBank creates a new I/O bank simulation
func NewIOBank() *IOBank {
	bank := &IOBank{
		stopChan:    make(chan bool),
		mcpMessages: make([]MCPMessage, 0),
	}

	// Initialize with realistic starting values for inputs
	rand.Seed(time.Now().UnixNano())
	
	// Set all digital inputs to false (off)
	for i := 0; i < 8; i++ {
		bank.digitalInputs[i] = false
	}
	
	// Set realistic analog input values
	bank.analogInputs[0] = 0.0  // AI 00: 0V
	bank.analogInputs[1] = 1.0  // Kettle Water Temperature: 20°C (1V = 20°C if 5V = 100°C)
	bank.analogInputs[2] = 0.0  // Cup Weight: 0g (no cup present initially, 0-5V = 0-1000g)
	bank.analogInputs[3] = 0.1  // Kettle Weight: 40g (empty kettle, 0.1V = 40g if 5V = 2000g)

	logger.Info("IOBank initialized with realistic starting values")
	return bank
}

// StartSimulation starts the background simulation that updates input values
func (io *IOBank) StartSimulation() {
	io.mu.Lock()
	if io.simulationRunning {
		io.mu.Unlock()
		return
	}
	io.simulationRunning = true
	io.mu.Unlock()

	go io.simulationLoop()
	logger.Info("IOBank simulation started")
}

// StopSimulation stops the background simulation
func (io *IOBank) StopSimulation() {
	io.mu.Lock()
	if !io.simulationRunning {
		io.mu.Unlock()
		return
	}
	io.simulationRunning = false
	io.mu.Unlock()

	io.stopChan <- true
	logger.Info("IOBank simulation stopped")
}

// simulationLoop runs in the background and periodically updates input values
func (io *IOBank) simulationLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-io.stopChan:
			return
		case <-ticker.C:
			io.updateInputs()
		}
	}
}

// updateInputs simulates tea-making machine physics
func (io *IOBank) updateInputs() {
	io.mu.Lock()
	defer io.mu.Unlock()

	// Simulation runs every 0.5 seconds, so calculate rates per 0.5-second interval
	const updateInterval = 0.5 // seconds
	
	// Tea-making machine physics simulation
	
	// AI1 = Kettle Water Temperature (0-5V representing 0-100°C, so 1V = 20°C)
	// DO3 = Kettle Power Relay (heating element)
	if io.digitalOutputs[3] { // Kettle heating
		// Heat at 100°C/min regardless of water level - hardware doesn't know better
		tempIncrease := 100.0 * (updateInterval / 60.0) // degrees per update
		currentTempC := io.analogInputs[1] * 20.0 // Convert V to °C (1V = 20°C)
		newTempC := currentTempC + tempIncrease
		if newTempC > 100.0 { // Cap at boiling point
			newTempC = 100.0
		}
		io.analogInputs[1] = newTempC / 20.0 // Convert back to volts
		logger.Debug("Kettle heating: %.1f°C (%.2fV)", newTempC, io.analogInputs[1])
	} else {
		// Natural cooling when not heating (lose ~1°C/min for more realistic cooling)
		coolingRate := 1.0 * (updateInterval / 60.0) // degrees per update
		currentTempC := io.analogInputs[1] * 20.0
		newTempC := currentTempC - coolingRate
		if newTempC < 20.0 { // Room temperature minimum
			newTempC = 20.0
		}
		io.analogInputs[1] = newTempC / 20.0
	}
	
	// AI3 = Kettle Weight (0-5V representing 0-2000g, so 1V = 400g)
	// DO1 = Kettle Water Inlet Valve
	if io.digitalOutputs[1] { // Water filling kettle
		// Fill at 2000g/60s = 2000g/min
		waterIncrease := 2000.0 * (updateInterval / 60.0) // grams per update
		currentWeightG := io.analogInputs[3] * 400.0 // Convert V to grams (1V = 400g)
		newWeightG := currentWeightG + waterIncrease
		if newWeightG > 2000.0 { // Kettle capacity is 2L = 2000g
			newWeightG = 2000.0 // Overflow protection
		}
		io.analogInputs[3] = newWeightG / 400.0 // Convert back to volts
		logger.Debug("Kettle filling: %.0fg (%.2fV)", newWeightG, io.analogInputs[3])
	}
	
	// AI2 = Cup Weight (0-5V representing 0-1000g, so 1V = 200g)
	// Multiple outputs can affect cup weight
	
	// DO2 = Kettle Water Outlet Valve (pouring into cup)
	if io.digitalOutputs[2] && io.analogInputs[3] > 0 { // Pouring from kettle (if kettle has water)
		// Pour at 250g/min (reduced from 500g/min for slower, more controlled pouring)
		pourRate := 250.0 * (updateInterval / 60.0) // grams per update
		
		// Check current kettle weight
		currentKettleG := io.analogInputs[3] * 400.0
		
		// Calculate how much can actually be poured from kettle
		actualPour := pourRate
		if actualPour > currentKettleG {
			actualPour = currentKettleG // Can't pour more than kettle contains
		}
		
		if actualPour > 0 {
			// Update kettle weight (decrease)
			newKettleG := currentKettleG - actualPour
			if newKettleG < 0 {
				newKettleG = 0
			}
			io.analogInputs[3] = newKettleG / 400.0
			
			// Only add to cup weight if cup is present (DI1 = true)
			if io.digitalInputs[1] {
				currentCupG := io.analogInputs[2] * 200.0 // Convert V to grams (1V = 200g for 0-1000g range)
				maxPourToCup := 300.0 - currentCupG // Cup capacity is 300ml/300g
				actualToCup := actualPour
				
				if actualToCup > maxPourToCup {
					actualToCup = maxPourToCup // Can't add more than cup can hold, rest spills
				}
				
				if actualToCup > 0 {
					newCupG := currentCupG + actualToCup
					if newCupG > 300.0 {
						newCupG = 300.0 // Cup overflow, excess spills
					}
					io.analogInputs[2] = newCupG / 200.0 // Convert back to volts
					logger.Debug("Pouring water: Kettle %.0fg, Cup %.0fg", newKettleG, newCupG)
				}
			} else {
				// No cup present - water just spills (kettle still empties)
				logger.Debug("Pouring water: Kettle %.0fg, no cup - water spilling", newKettleG)
			}
		}
	}
	
	// Cup weight should be zero when no cup is present (DI1 = false)
	if !io.digitalInputs[1] {
		io.analogInputs[2] = 0.0 // No cup = no weight reading
		// Also ensure teaspoon can't be "in cup", "stirring", or "squashing" if no cup present
		if io.digitalInputs[2] {
			io.digitalInputs[2] = false
			logger.Info("Cup removed while teaspoon was in cup - DI2 now false")
		}
		if io.digitalInputs[3] {
			io.digitalInputs[3] = false
			logger.Info("Cup removed while teaspoon was stirring - DI3 now false")
		}
		if io.digitalInputs[4] {
			io.digitalInputs[4] = false
			logger.Info("Cup removed while teaspoon was squashing - DI4 now false")
		}
		// Reset teabag state when cup is removed
		if io.digitalInputs[5] {
			io.digitalInputs[5] = false
			logger.Info("Cup removed while teabag was in cup - DI5 (Teabag In) now false")
		}
	}
}

// Digital Input Methods
func (io *IOBank) GetDigitalInput(pin int) (bool, error) {
	if pin < 0 || pin > 7 {
		return false, fmt.Errorf("digital input pin %d out of range (0-7)", pin)
	}

	io.mu.RLock()
	defer io.mu.RUnlock()
	
	value := io.digitalInputs[pin]
	logger.Debug("Read digital input %d: %v", pin, value)
	return value, nil
}

func (io *IOBank) GetAllDigitalInputs() [8]bool {
	io.mu.RLock()
	defer io.mu.RUnlock()
	return io.digitalInputs
}

// Digital Output Methods
// SetDigitalOutput sets a digital output and handles special logic for dispensers
func (io *IOBank) SetDigitalOutput(pin int, value bool) error {
	if pin < 0 || pin > 15 {
		return fmt.Errorf("digital output pin %d out of range (0-15)", pin)
	}

	io.mu.Lock()
	defer io.mu.Unlock()
	
	// Handle special dispenser logic
	switch pin {
	case 4: // Cup Dispenser Solenoid
		if io.digitalOutputs[4] && !value { // Falling edge (pulse end)
			// Dispense a cup - set DI1 to true (cup present)
			io.digitalInputs[1] = true
			logger.Info("Cup dispensed - DI1 now true")
		}
	case 5: // Teabag Dispenser Solenoid  
		if io.digitalOutputs[5] && !value { // Falling edge (pulse end)
			if io.digitalInputs[1] { // Cup present - add to cup
				currentCupG := io.analogInputs[2] * 200.0 // Convert V to grams (1V = 200g)
				newCupG := currentCupG + 2.0 // Add 2g for teabag
				if newCupG > 300.0 {
					newCupG = 300.0 // Cup capacity limit
				}
				io.analogInputs[2] = newCupG / 200.0 // Convert back to volts
				
				// Set DI5 (Teabag In) to true when teabag is dispensed into cup
				io.digitalInputs[5] = true
				logger.Info("Teabag dispensed into cup - Cup weight now %.0fg, DI5 (Teabag In) now true", newCupG)
			} else {
				// No cup - teabag falls to floor, DI5 remains false
				logger.Info("Teabag dispensed - no cup, teabag falls to floor")
			}
		}
	case 6: // Sugar Dispenser Solenoid
		if io.digitalOutputs[6] && !value { // Falling edge (pulse end)
			if io.digitalInputs[1] { // Cup present - add to cup
				currentCupG := io.analogInputs[2] * 200.0 // Convert V to grams (1V = 200g)
				newCupG := currentCupG + 7.0 // Add 7g for one sugar
				if newCupG > 300.0 {
					newCupG = 300.0 // Cup capacity limit
				}
				io.analogInputs[2] = newCupG / 200.0 // Convert back to volts
				logger.Info("Sugar dispensed into cup - Cup weight now %.0fg", newCupG)
			} else {
				// No cup - sugar falls to floor
				logger.Info("Sugar dispensed - no cup, sugar falls to floor")
			}
		}
	case 7: // Milk Dispenser Solenoid (discrete 4g splashes)
		if !io.digitalOutputs[7] && value { // Rising edge (activation)
			if io.digitalInputs[1] { // Cup present - add milk to cup
				currentCupG := io.analogInputs[2] * 200.0 // Convert V to grams (1V = 200g)
				milkAmount := 4.0 // Exactly 4g per activation
				
				// Check if cup can hold the milk
				maxMilkToCup := 300.0 - currentCupG // Cup capacity is 300g
				actualMilk := milkAmount
				
				if actualMilk > maxMilkToCup {
					actualMilk = maxMilkToCup // Can't add more than cup can hold, rest spills
				}
				
				if actualMilk > 0 {
					newCupG := currentCupG + actualMilk
					if newCupG > 300.0 {
						newCupG = 300.0 // Cup overflow protection
					}
					io.analogInputs[2] = newCupG / 200.0 // Convert back to volts
					logger.Info("Milk splash dispensed - Added %.0fg, Cup weight now %.0fg", actualMilk, newCupG)
				} else {
					logger.Info("Milk splash dispensed - Cup full, milk spilled")
				}
			} else {
				// No cup present - milk just spills
				logger.Info("Milk splash dispensed - no cup, milk spilling")
			}
		}
	case 8: // Teaspoon Height Actuator (high = lower spoon, low = raise spoon)
		// Update DI2 (teaspoon in cup) based on teaspoon position and cup presence
		if io.digitalInputs[1] { // Cup is present
			if value { // DO8 high = lower spoon
				io.digitalInputs[2] = true // Teaspoon now in cup
				logger.Info("Teaspoon lowered into cup - DI2 now true")
			} else { // DO8 low = raise spoon
				// Check if teabag extraction is happening (spoon was squashing when raised)
				if io.digitalInputs[2] && io.digitalInputs[4] { // Was in cup and squashing
					// Extract teabag - reduce cup weight by 4g (2g dry + 2g wet)
					currentCupG := io.analogInputs[2] * 200.0 // Convert V to grams
					newCupG := currentCupG - 4.0 // Remove 4g for wet teabag extraction
					if newCupG < 0 {
						newCupG = 0
					}
					io.analogInputs[2] = newCupG / 200.0 // Convert back to volts
					
					// Set DI5 (Teabag In) to false when teabag is extracted
					io.digitalInputs[5] = false
					logger.Info("Teabag extracted by raising squashing spoon - Cup weight reduced by 4g, now %.0fg, DI5 (Teabag In) now false", newCupG)
				}
				
				io.digitalInputs[2] = false // Teaspoon now raised
				io.digitalInputs[3] = false // Can't stir if not in cup
				io.digitalInputs[4] = false // Can't squash if not in cup
				logger.Info("Teaspoon raised from cup - DI2 now false, DI3 and DI4 now false")
			}
		} else {
			// No cup present - teaspoon can't be "in cup", "stirring", or "squashing"
			io.digitalInputs[2] = false
			io.digitalInputs[3] = false
			io.digitalInputs[4] = false
			if value {
				logger.Info("Teaspoon lowered but no cup present - DI2, DI3, and DI4 remain false")
			}
		}
	case 9: // Teaspoon Stir Actuator (high = stirring, low = stop stirring)
		// Update DI3 (teaspoon stirring) based on stir state and teaspoon position
		if io.digitalInputs[2] { // Teaspoon is in cup
			if value { // DO9 high = stirring
				io.digitalInputs[3] = true // Teaspoon now stirring
				logger.Info("Teaspoon stirring activated - DI3 now true")
			} else { // DO9 low = stop stirring
				io.digitalInputs[3] = false // Teaspoon not stirring
				logger.Info("Teaspoon stirring deactivated - DI3 now false")
			}
		} else {
			// Teaspoon not in cup - can't be stirring
			io.digitalInputs[3] = false
			if value {
				logger.Info("Stirring activated but teaspoon not in cup - DI3 remains false")
			}
		}
	case 10: // Teaspoon Squash Actuator (high = squash, low = return to center)
		// Update DI4 (teaspoon squashing) based on squash state and teaspoon position
		if io.digitalInputs[2] { // Teaspoon is in cup
			if value { // DO10 high = squashing
				io.digitalInputs[4] = true // Teaspoon now squashing
				logger.Info("Teaspoon squashing activated - DI4 now true")
			} else { // DO10 low = return to center
				io.digitalInputs[4] = false // Teaspoon not squashing
				logger.Info("Teaspoon squashing deactivated - DI4 now false")
			}
		} else {
			// Teaspoon not in cup - can't be squashing
			io.digitalInputs[4] = false
			if value {
				logger.Info("Squashing activated but teaspoon not in cup - DI4 remains false")
			}
		}
	}
	
	io.digitalOutputs[pin] = value
	logger.Info("Set digital output %d to %v", pin, value)
	return nil
}

func (io *IOBank) GetDigitalOutput(pin int) (bool, error) {
	if pin < 0 || pin > 15 {
		return false, fmt.Errorf("digital output pin %d out of range (0-15)", pin)
	}

	io.mu.RLock()
	defer io.mu.RUnlock()
	
	value := io.digitalOutputs[pin]
	logger.Debug("Read digital output %d: %v", pin, value)
	return value, nil
}

func (io *IOBank) GetAllDigitalOutputs() [16]bool {
	io.mu.RLock()
	defer io.mu.RUnlock()
	return io.digitalOutputs
}

// Analog Input Methods
func (io *IOBank) GetAnalogInput(pin int) (float64, error) {
	if pin < 0 || pin > 3 {
		return 0, fmt.Errorf("analog input pin %d out of range (0-3)", pin)
	}

	io.mu.RLock()
	defer io.mu.RUnlock()
	
	value := io.analogInputs[pin]
	logger.Debug("Read analog input %d: %.3fV", pin, value)
	return value, nil
}

func (io *IOBank) GetAllAnalogInputs() [4]float64 {
	io.mu.RLock()
	defer io.mu.RUnlock()
	return io.analogInputs
}

// Analog Output Methods
func (io *IOBank) SetAnalogOutput(pin int, value float64) error {
	if pin < 0 || pin > 3 {
		return fmt.Errorf("analog output pin %d out of range (0-3)", pin)
	}
	if value < 0 || value > 5.0 {
		return fmt.Errorf("analog output value %.3f out of range (0.0-5.0V)", value)
	}

	io.mu.Lock()
	defer io.mu.Unlock()
	
	io.analogOutputs[pin] = value
	logger.Info("Set analog output %d to %.3fV", pin, value)
	return nil
}

func (io *IOBank) GetAnalogOutput(pin int) (float64, error) {
	if pin < 0 || pin > 3 {
		return 0, fmt.Errorf("analog output pin %d out of range (0-3)", pin)
	}

	io.mu.RLock()
	defer io.mu.RUnlock()
	
	value := io.analogOutputs[pin]
	logger.Debug("Read analog output %d: %.3fV", pin, value)
	return value, nil
}

func (io *IOBank) GetAllAnalogOutputs() [4]float64 {
	io.mu.RLock()
	defer io.mu.RUnlock()
	return io.analogOutputs
}

// Reset resets the I/O bank to initial values
func (io *IOBank) Reset() error {
	io.mu.Lock()
	defer io.mu.Unlock()

	// Reset all digital inputs to false
	for i := 0; i < 8; i++ {
		io.digitalInputs[i] = false
	}

	// Reset all digital outputs to false
	for i := 0; i < 16; i++ {
		io.digitalOutputs[i] = false
	}

	// Reset all analog outputs to 0V
	for i := 0; i < 4; i++ {
		io.analogOutputs[i] = 0.0
	}

	// Reset analog inputs to initial realistic values
	io.analogInputs[0] = 0.0  // AI 00: 0V
	io.analogInputs[1] = 1.0  // Kettle Water Temperature: 20°C (1V = 20°C if 5V = 100°C)
	io.analogInputs[2] = 0.0  // Cup Weight: 0g (no cup present initially, 0-5V = 0-1000g)
	io.analogInputs[3] = 0.1  // Kettle Weight: 40g (empty kettle, 0.1V = 40g if 5V = 2000g)

	logger.Info("System reset to initial values - all outputs off, inputs at startup values")
	return nil
}

// Status returns a summary of all I/O states
func (io *IOBank) GetStatus() map[string]interface{} {
	io.mu.RLock()
	defer io.mu.RUnlock()

	return map[string]interface{}{
		"digital_inputs":     io.digitalInputs,
		"digital_outputs":    io.digitalOutputs,
		"analog_inputs":      io.analogInputs,
		"analog_outputs":     io.analogOutputs,
		"simulation_running": io.simulationRunning,
		"last_mcp_message":   io.lastMCPMessage,
		"mcp_messages":       io.mcpMessages,
	}
}

// AddMCPMessage records an MCP message received by the system
func (io *IOBank) AddMCPMessage(toolName, message string) {
	io.mu.Lock()
	defer io.mu.Unlock()

	mcpMsg := MCPMessage{
		Timestamp: time.Now(),
		ToolName:  toolName,
		Message:   message,
	}

	// Keep only the last 10 messages to prevent memory growth
	io.mcpMessages = append(io.mcpMessages, mcpMsg)
	if len(io.mcpMessages) > 10 {
		io.mcpMessages = io.mcpMessages[1:]
	}

	// Update the last message pointer
	io.lastMCPMessage = &mcpMsg

	logger.Info("MCP message recorded:", toolName, "-", message)
}
