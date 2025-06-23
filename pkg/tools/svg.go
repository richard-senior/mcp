package tools

import (
	"fmt"

	"github.com/richard-senior/mcp/pkg/protocol"
	"github.com/richard-senior/mcp/pkg/util"
)

func NewSvgTool() protocol.Tool {
	return protocol.Tool{
		Name:        "svg_tool",
		Description: `provides a suite of functions for processing svg files`,
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"command": {
					Type: "string",
					Description: `
					The command to execute. One of:
					create_from_raster:
						Create a new SVG from the given raster image which is provided absolute filepath
						- Use sourcepath to specify the location of the raster image
						- Use destpath to specify the location of the output SVG file.
						- If destpath is not given then the present working directory is used.
						- Returns the full filepath of the created file
					add_text_to_svg:
					    Adds the given text to the SVG file at the given absolute filepath
						- Use sourcepath to specify the location of the SVG file to alter
						- Use text to specify the text to be added to the SVG
						- Use x and y to specify where the text should be placed
						- Use style to specify the CSS styling to use on the created <text> element
					create_cheesy_meme:
						Creates an svg image accompanied by a witty quote or joke
						- Should be used if the user asks something like "Please create a meme about Elvis Preslety" etc.
						- The user may ask simply for you to create a meme, in which case you should choose the search topic for the image
						- use 'sourcepath' to pass a search term to use to find a picture for this meme such as: 'elvis presely'
						- use text to supply the a short witty joke (text) for the meme which will appear in the lower part of the SVG under the image
						- The text should be clever and amusing and related to the search term (the image).
						  For example if the user passes 'elvis presley' then the text could be 'Uh huh huh' etc.
						- Text should be no longer than 30 characters including spaces
						- Returns the location of the created SVG file
					`,
				},
				"sourcepath": {
					Type:        "string",
					Description: "The absolute filepath to an image or other resource that is to be worked on",
				},
				"url": {
					Type:        "string",
					Description: "The URL of the SVG file",
				},
				"destpath": {
					Type:        "string",
					Description: "The absolute filepath of the output image to be created or modified",
				},
				"text": {
					Type:        "string",
					Description: "Text which should appear in the svg",
				},
				"style": {
					Type:        "string",
					Description: "The CCS styling to use when CSS styling is called for",
				},
				"x": {
					Type:        "int",
					Description: "The X Coordinate to use when called for",
				},
				"y": {
					Type:        "int",
					Description: "The Y Coordinate to use when called for",
				},
			},
			Required: []string{"command"},
		},
	}
}

func HandleSvgTool(params any) (any, error) {
	if params == nil {
		return nil, fmt.Errorf("no params given")
	}
	// Convert params to map[string]any
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Couldn't format the parmeters as a map of strings")
	}
	command, ok := paramsMap["command"].(string)
	if !ok {
		return nil, fmt.Errorf("No command parameter was sent")
	}
	switch c := command; c {
	case "create_from_raster":
		return HandleCreateFromRaster(params)
	case "create_cheesy_meme":
		return HandleCreateCheesyMeme(params)
	case "add_text_to_svg":
		return HandleAddTextToSvg(params)
	default:
		return nil, fmt.Errorf("command %s not currently supported", c)
	}
}

func HandleCreateFromRaster(params any) (any, error) {
	return nil, nil
}

// Loads and modifies the given SVG file by adding the given text
func HandleAddTextToSvg(params any) (any, error) {
	return nil, nil
}

// given a raster image, creates a cheezy meme for demonstration purposes
func HandleCreateCheesyMeme(params any) (any, error) {
	if params == nil {
		return nil, fmt.Errorf("no params given")
	}
	// Convert params to map[string]any
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Couldn't format the parmeters as a map of strings")
	}
	searchTerm, ok := paramsMap["sourcepath"].(string)
	if !ok {
		return nil, fmt.Errorf("No command parameter was sent")
	}
	// search term for image contained in
	bytes, _, err := WikipediaImageSearch(searchTerm, 200)
	if err != nil {
		return nil, err
	}
	foo, err := util.NewSVGFromRasterContent(bytes)
	if err != nil {
		return nil, err
	}

	// add text "Loves BT soooo much" at the bottom of the image
	return foo, nil
}
