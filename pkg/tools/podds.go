package tools

import (
	"fmt"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/protocol"
	"github.com/richard-senior/mcp/pkg/util"
)

func PoddsTool() protocol.Tool {
	return protocol.Tool{
		Name: "podds_tool",
		Description: `
		A tool that provides a set of commands for interacting with an EFL League football prediction and statistics resource.
		Currently can get interesting data on past matches
		`,
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"command": {
					Type: "string",
					Description: `
					The command you want to run:
					- last_match_stats
						- get data on the last match played by 'Man U' etc.
					`,
				},
				"text": {
					Type:        "string",
					Description: "The text of the meme, this should be something amusing, witty or edgy and related to the searchterm in some clever way. If the user does not supply this for you then you should create the text yourself. It should be no longer than 40 characters",
				},
				"filepath": {
					Type:        "string",
					Description: "The absolute filepath in which to store the resulting svg file. If omitted will default to the present working directory.",
				},
			},
			Required: []string{"searchterm", "text"},
		},
	}
}

// given a raster image, creates a cheezy meme for demonstration purposes
func HandlePoddsTool(params any) (any, error) {

	if params == nil {
		return nil, fmt.Errorf("no params given")
	}
	// Convert params to map[string]any
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Couldn't format the parmeters as a map of strings")
	}
	searchTerm, ok := paramsMap["searchterm"].(string)
	if !ok {
		return nil, fmt.Errorf("No command parameter was sent")
	}
	text, ok := paramsMap["text"].(string)
	if !ok {
		return nil, fmt.Errorf("No text parameter was sent")
	}
	filepath, ok := paramsMap["filepath"].(string)
	// search term for image contained in
	bytes, _, err := WikipediaImageSearch(searchTerm, 400)
	if err != nil {
		return nil, err
	}
	svg, err := util.NewSVGFromRasterContent(bytes)
	if err != nil {
		return nil, err
	}
	if filepath != "" {
		logger.Info("Saving meme to " + filepath)
	}

	// Calculate font size based on image width
	// Assuming average word length of 5 characters and targeting 5 words per line
	// Each character is approximately 0.6 times the font size in width
	averageWordLength := 5
	targetWordsPerLine := 5
	charactersPerLine := averageWordLength * targetWordsPerLine

	// Calculate font size: imageWidth / (charactersPerLine * 0.6)
	// The 0.6 factor is an approximation of character width to font size ratio
	// We subtract 60 from width to account for margins (30px on each side)
	fontSize := (svg.Width - 60) / (charactersPerLine * 6 / 10)

	// Set minimum and maximum font sizes
	if fontSize < 18 {
		fontSize = 18 // Minimum font size for readability
	} else if fontSize > 48 {
		fontSize = 60 // Maximum font size to prevent excessive text size
	}

	fontStyle := fmt.Sprintf("font-weight: bold; font-size: %dpx; font-family: 'Comic Sans MS'; fill: red;", fontSize)
	logger.Inform("Using font size for image width: ", fontSize, svg.Width)

	// make text appear at the bottom at approximately 4/5ths of the image height
	textYPosition := int(float64(svg.Height) * 0.8)
	logger.Inform("Placing text at Y position: ", textYPosition, " (image height: ", svg.Height, ")")

	svg.AddWrappedText("cheezymeme", text, fontStyle, 20, textYPosition, svg.Width-60, 30, 1)

	outputPath := "./cheezymeme.svg"
	if filepath != "" {
		outputPath = filepath
		logger.Info("Saving meme to " + filepath)
	}

	svg.ToSVGFile(outputPath)

	return map[string]any{
		"location": outputPath,
	}, nil
}
