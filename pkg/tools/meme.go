package tools

import (
	"fmt"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/protocol"
	"github.com/richard-senior/mcp/pkg/util"
)

func NewMemeTool() protocol.Tool {
	return protocol.Tool{
		Name: "meme_tool",
		Description: `
		Creates memes designed to amuse in a whimsical manner.
		A photograph of something with some text underneath it.
		If the user does not specify what the text should be then you should decide for yourself.
		Returns the location of the created image if successful.
		`,
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"searchterm": {
					Type: "string",
					Description: `
					The subject of the meme.
					This will result in a picture being downloaded and used as the background of the meme.
					- Do not embelish the search term unless it fails to yield a result.
					  For example if asked for 'Noel Edmonds' then don't add 'TV presenter' unless the plain search term fails
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
func HandleMemeTool(params any) (any, error) {

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

	// Calculate optimal font size and positioning to ensure text fits within image bounds
	margin := 20 // Margin from edges
	textAreaWidth := svg.Width - (2 * margin)
	
	// Estimate number of lines needed based on text length and available width
	// Average character width is approximately 0.6 times font size
	words := len(text) / 5 // Rough estimate of word count (5 chars per word average)
	if words < 1 {
		words = 1
	}
	
	// Start with a reasonable font size and adjust
	fontSize := 24
	maxFontSize := svg.Height / 8 // Don't let font be more than 1/8 of image height
	minFontSize := 12
	
	// Calculate how much vertical space we want to reserve for text (bottom 25% of image)
	textAreaHeight := svg.Height / 4
	if textAreaHeight < 60 {
		textAreaHeight = 60 // Minimum text area height
	}
	
	// Iteratively find the best font size that fits
	for fontSize > minFontSize {
		avgCharWidth := float64(fontSize) * 0.6
		charsPerLine := int(float64(textAreaWidth) / avgCharWidth)
		
		if charsPerLine > 0 {
			// Estimate number of lines needed
			estimatedLines := (len(text) + charsPerLine - 1) / charsPerLine // Ceiling division
			if estimatedLines < 1 {
				estimatedLines = 1
			}
			
			// Calculate total text height (including line spacing)
			lineHeight := int(float64(fontSize) * 1.2) // 1.2 line spacing
			totalTextHeight := estimatedLines * lineHeight
			
			// Check if text fits in our reserved area
			if totalTextHeight <= textAreaHeight && fontSize <= maxFontSize {
				break
			}
		}
		
		fontSize -= 2 // Reduce font size and try again
	}
	
	// Ensure minimum font size
	if fontSize < minFontSize {
		fontSize = minFontSize
	}
	
	logger.Inform("Using font size: ", fontSize, " for image dimensions: ", svg.Width, "x", svg.Height)

	// Create font style with calculated size
	fontStyle := fmt.Sprintf("font-weight: bold; font-size: %dpx; font-family: 'Impact', 'Arial Black', sans-serif; fill: white; stroke: black; stroke-width: 1px;", fontSize)
	
	// Position text in the bottom area of the image
	// Calculate Y position to ensure text doesn't overflow
	lineHeight := int(float64(fontSize) * 1.2)
	
	// Estimate how many lines we'll actually have with this font size
	avgCharWidth := float64(fontSize) * 0.6
	charsPerLine := int(float64(textAreaWidth) / avgCharWidth)
	estimatedLines := 1
	if charsPerLine > 0 {
		estimatedLines = (len(text) + charsPerLine - 1) / charsPerLine
	}
	
	totalTextHeight := estimatedLines * lineHeight
	
	// Position text so it's in the bottom portion but doesn't overflow
	// Start from bottom and work up, leaving some margin
	textYPosition := svg.Height - margin - totalTextHeight + lineHeight // +lineHeight because SVG text Y is baseline
	
	// Ensure text doesn't start too high up (maintain some separation from image content)
	minYPosition := svg.Height * 2 / 3 // Don't start text higher than 2/3 down the image
	if textYPosition < minYPosition {
		textYPosition = minYPosition
	}
	
	logger.Inform("Placing text at Y position: ", textYPosition, " (estimated lines: ", estimatedLines, ", total text height: ", totalTextHeight, ")")

	// Add the text with wrapping
	svg.AddWrappedText("cheezymeme", text, fontStyle, margin, textYPosition, textAreaWidth, lineHeight, 1)

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
