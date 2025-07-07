package test

import (
	"testing"
	"time"

	"github.com/richard-senior/mcp/_digital-io/internal/iobank"
)

func TestIOBankCreation(t *testing.T) {
	bank := iobank.NewIOBank()
	if bank == nil {
		t.Fatal("Failed to create IOBank")
	}
}

func TestDigitalOutputs(t *testing.T) {
	bank := iobank.NewIOBank()

	// Test setting and getting digital outputs
	testCases := []struct {
		pin   int
		value bool
	}{
		{0, true},
		{15, false},
		{31, true},
	}

	for _, tc := range testCases {
		err := bank.SetDigitalOutput(tc.pin, tc.value)
		if err != nil {
			t.Errorf("Failed to set digital output %d: %v", tc.pin, err)
		}

		value, err := bank.GetDigitalOutput(tc.pin)
		if err != nil {
			t.Errorf("Failed to get digital output %d: %v", tc.pin, err)
		}

		if value != tc.value {
			t.Errorf("Digital output %d: expected %v, got %v", tc.pin, tc.value, value)
		}
	}
}

func TestDigitalOutputBounds(t *testing.T) {
	bank := iobank.NewIOBank()

	// Test invalid pin numbers
	err := bank.SetDigitalOutput(-1, true)
	if err == nil {
		t.Error("Expected error for pin -1, got nil")
	}

	err = bank.SetDigitalOutput(32, true)
	if err == nil {
		t.Error("Expected error for pin 32, got nil")
	}
}

func TestAnalogOutputs(t *testing.T) {
	bank := iobank.NewIOBank()

	// Test setting and getting analog outputs
	testCases := []struct {
		pin   int
		value float64
	}{
		{0, 0.0},
		{3, 2.5},
		{7, 5.0},
	}

	for _, tc := range testCases {
		err := bank.SetAnalogOutput(tc.pin, tc.value)
		if err != nil {
			t.Errorf("Failed to set analog output %d: %v", tc.pin, err)
		}

		value, err := bank.GetAnalogOutput(tc.pin)
		if err != nil {
			t.Errorf("Failed to get analog output %d: %v", tc.pin, err)
		}

		if value != tc.value {
			t.Errorf("Analog output %d: expected %.3f, got %.3f", tc.pin, tc.value, value)
		}
	}
}

func TestAnalogOutputBounds(t *testing.T) {
	bank := iobank.NewIOBank()

	// Test invalid pin numbers
	err := bank.SetAnalogOutput(-1, 2.5)
	if err == nil {
		t.Error("Expected error for pin -1, got nil")
	}

	err = bank.SetAnalogOutput(8, 2.5)
	if err == nil {
		t.Error("Expected error for pin 8, got nil")
	}

	// Test invalid voltage values
	err = bank.SetAnalogOutput(0, -0.1)
	if err == nil {
		t.Error("Expected error for voltage -0.1V, got nil")
	}

	err = bank.SetAnalogOutput(0, 5.1)
	if err == nil {
		t.Error("Expected error for voltage 5.1V, got nil")
	}
}

func TestDigitalInputs(t *testing.T) {
	bank := iobank.NewIOBank()

	// Test getting digital inputs (should not error)
	for pin := 0; pin < 32; pin++ {
		_, err := bank.GetDigitalInput(pin)
		if err != nil {
			t.Errorf("Failed to get digital input %d: %v", pin, err)
		}
	}

	// Test invalid pin
	_, err := bank.GetDigitalInput(32)
	if err == nil {
		t.Error("Expected error for pin 32, got nil")
	}
}

func TestAnalogInputs(t *testing.T) {
	bank := iobank.NewIOBank()

	// Test getting analog inputs (should not error)
	for pin := 0; pin < 8; pin++ {
		value, err := bank.GetAnalogInput(pin)
		if err != nil {
			t.Errorf("Failed to get analog input %d: %v", pin, err)
		}

		// Value should be in valid range
		if value < 0 || value > 5.0 {
			t.Errorf("Analog input %d value %.3f out of range (0-5V)", pin, value)
		}
	}

	// Test invalid pin
	_, err := bank.GetAnalogInput(8)
	if err == nil {
		t.Error("Expected error for pin 8, got nil")
	}
}

func TestSimulation(t *testing.T) {
	bank := iobank.NewIOBank()

	// Get initial status
	status1 := bank.GetStatus()
	if status1 == nil {
		t.Fatal("Failed to get initial status")
	}

	// Start simulation
	bank.StartSimulation()
	defer bank.StopSimulation()

	// Wait a bit for simulation to potentially change values
	time.Sleep(100 * time.Millisecond)

	// Get status again
	status2 := bank.GetStatus()
	if status2 == nil {
		t.Fatal("Failed to get status after simulation start")
	}

	// Verify simulation is running
	running, ok := status2["simulation_running"].(bool)
	if !ok || !running {
		t.Error("Simulation should be running")
	}
}

func TestGetAllMethods(t *testing.T) {
	bank := iobank.NewIOBank()

	// Test getting all digital inputs
	digitalInputs := bank.GetAllDigitalInputs()
	if len(digitalInputs) != 32 {
		t.Errorf("Expected 32 digital inputs, got %d", len(digitalInputs))
	}

	// Test getting all digital outputs
	digitalOutputs := bank.GetAllDigitalOutputs()
	if len(digitalOutputs) != 32 {
		t.Errorf("Expected 32 digital outputs, got %d", len(digitalOutputs))
	}

	// Test getting all analog inputs
	analogInputs := bank.GetAllAnalogInputs()
	if len(analogInputs) != 8 {
		t.Errorf("Expected 8 analog inputs, got %d", len(analogInputs))
	}

	// Test getting all analog outputs
	analogOutputs := bank.GetAllAnalogOutputs()
	if len(analogOutputs) != 8 {
		t.Errorf("Expected 8 analog outputs, got %d", len(analogOutputs))
	}
}
