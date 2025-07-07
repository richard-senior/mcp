package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/richard-senior/mcp/_digital-io/internal/logger"
)

// AnalogRange defines the conversion range for analog I/O
type AnalogRange struct {
	MinValue string `json:"min_value"`
	MaxValue string `json:"max_value"`
	Unit     string `json:"unit"`
}

// IOLabels holds the custom labels for all I/O pins
type IOLabels struct {
	DigitalInputs       map[string]string      `json:"digital_inputs"`
	DigitalOutputs      map[string]string      `json:"digital_outputs"`
	AnalogInputs        map[string]string      `json:"analog_inputs"`
	AnalogOutputs       map[string]string      `json:"analog_outputs"`
	AnalogInputRanges   map[string]AnalogRange `json:"analog_input_ranges"`
	AnalogOutputRanges  map[string]AnalogRange `json:"analog_output_ranges"`
}

var (
	labels     *IOLabels
	labelsOnce sync.Once
	labelsMu   sync.RWMutex
)

// GetIOLabels loads the I/O labels from the config file
func GetIOLabels() *IOLabels {
	labelsOnce.Do(func() {
		labels = &IOLabels{
			DigitalInputs:      make(map[string]string),
			DigitalOutputs:     make(map[string]string),
			AnalogInputs:       make(map[string]string),
			AnalogOutputs:      make(map[string]string),
			AnalogInputRanges:  make(map[string]AnalogRange),
			AnalogOutputRanges: make(map[string]AnalogRange),
		}
		loadLabels()
	})

	return labels
}

// ReloadLabels forces a reload of labels from the config file
func ReloadLabels() {
	labelsMu.Lock()
	defer labelsMu.Unlock()
	
	// Reset the labels
	labels.DigitalInputs = make(map[string]string)
	labels.DigitalOutputs = make(map[string]string)
	labels.AnalogInputs = make(map[string]string)
	labels.AnalogOutputs = make(map[string]string)
	labels.AnalogInputRanges = make(map[string]AnalogRange)
	labels.AnalogOutputRanges = make(map[string]AnalogRange)
	
	// Reload from file
	loadLabels()
	logger.Info("Labels reloaded from config file")
}

// loadLabels loads the labels from the config file
func loadLabels() {
	configPath, err := GetConfigPath("io_labels.json")
	if err != nil {
		logger.Error("Failed to determine config path: %v", err)
		return
	}
	
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Warn("I/O labels config file not found at %s, using default labels", configPath)
		return
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		logger.Error("Failed to read I/O labels config file: %v", err)
		return
	}

	labelsMu.Lock()
	defer labelsMu.Unlock()

	if err := json.Unmarshal(data, labels); err != nil {
		logger.Error("Failed to parse I/O labels config file: %v", err)
		return
	}

	logger.Info("Loaded I/O labels from %s", configPath)
}

// SaveLabels saves the current labels to the config file
func SaveLabels() error {
	configPath, err := GetConfigPath("io_labels.json")
	if err != nil {
		logger.Error("Failed to determine config path: %v", err)
		return err
	}
	
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error("Failed to create config directory: %v", err)
		return err
	}

	labelsMu.RLock()
	data, err := json.MarshalIndent(labels, "", "  ")
	labelsMu.RUnlock()
	
	if err != nil {
		logger.Error("Failed to marshal I/O labels: %v", err)
		return err
	}

	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		logger.Error("Failed to write I/O labels config file: %v", err)
		return err
	}

	logger.Info("Saved I/O labels to %s", configPath)
	return nil
}

// UpdateLabel updates a label for a specific I/O pin
func UpdateLabel(ioType, pinStr, label string) error {
	labelsMu.Lock()
	defer labelsMu.Unlock()

	switch ioType {
	case "digital_input":
		labels.DigitalInputs[pinStr] = label
	case "digital_output":
		labels.DigitalOutputs[pinStr] = label
	case "analog_input":
		labels.AnalogInputs[pinStr] = label
	case "analog_output":
		labels.AnalogOutputs[pinStr] = label
	default:
		logger.Error("Invalid I/O type: %s", ioType)
		return nil
	}

	return SaveLabels()
}


