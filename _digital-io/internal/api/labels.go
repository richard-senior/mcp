package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/richard-senior/mcp/_digital-io/internal/config"
	"github.com/richard-senior/mcp/_digital-io/internal/logger"
)

// LabelRequest represents a request to update an I/O label
type LabelRequest struct {
	Label string `json:"label"`
}

// GetLabelsHandler returns all I/O labels
func (h *APIHandler) GetLabelsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config.GetIOLabels())
}

// UpdateLabelHandler updates a label for a specific I/O pin
func (h *APIHandler) UpdateLabelHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ioType := vars["type"]
	pinStr := vars["pin"]
	
	// Validate pin number
	pin, err := strconv.Atoi(pinStr)
	if err != nil {
		http.Error(w, "Invalid pin number", http.StatusBadRequest)
		return
	}
	
	// Validate I/O type and pin range
	var maxPin int
	switch ioType {
	case "digital_input":
		maxPin = 7  // 8 pins: 0-7
	case "digital_output":
		maxPin = 15 // 16 pins: 0-15
	case "analog_input", "analog_output":
		maxPin = 3  // 4 pins: 0-3
	default:
		http.Error(w, "Invalid I/O type", http.StatusBadRequest)
		return
	}
	
	if pin < 0 || pin > maxPin {
		http.Error(w, "Pin number out of range", http.StatusBadRequest)
		return
	}
	
	// Parse request body
	var req LabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Update label
	err = config.UpdateLabel(ioType, pinStr, req.Label)
	if err != nil {
		logger.Error("Failed to update label: %v", err)
		http.Error(w, "Failed to update label", http.StatusInternalServerError)
		return
	}
	
	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"type":    ioType,
		"pin":     pin,
		"label":   req.Label,
	})
}

// ReloadLabelsHandler forces a reload of labels from the config file
func (h *APIHandler) ReloadLabelsHandler(w http.ResponseWriter, r *http.Request) {
	config.ReloadLabels()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Labels reloaded from config file",
	})
}

// AddLabelsToStatus adds labels to the status response
func AddLabelsToStatus(status map[string]interface{}) map[string]interface{} {
	labels := config.GetIOLabels()
	
	// Add labels to the status
	status["labels"] = map[string]interface{}{
		"digital_inputs":  labels.DigitalInputs,
		"digital_outputs": labels.DigitalOutputs,
		"analog_inputs":   labels.AnalogInputs,
		"analog_outputs":  labels.AnalogOutputs,
	}
	
	// Add analog ranges to the status
	status["analog_ranges"] = map[string]interface{}{
		"inputs":  labels.AnalogInputRanges,
		"outputs": labels.AnalogOutputRanges,
	}
	
	return status
}
