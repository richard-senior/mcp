package util

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

///////////////////////////////////////////////////////////////////////////////
/// SVGEmbeddedRaster
///////////////////////////////////////////////////////////////////////////////

// Holds information about raster images that are embedded into SVG files
type SVGEmbeddedRaster struct {
	Layer         int
	X, Y          int
	Name          string
	FilePath      string
	Kind          string
	Width, Height int
	Content       []byte
}

func NewSVGEmbeddedRasterContent(content []byte) (*SVGEmbeddedRaster, error) {
	// First determine the image type and dimensions from the raw content
	kind, width, height, err := DetermineImageType("", content)
	if err != nil {
		return nil, fmt.Errorf("failed to determine image type: %w", err)
	}
	// Then base64 encode the content
	encodedContent := []byte(base64.StdEncoding.EncodeToString(content))

	ret := &SVGEmbeddedRaster{
		X:        0,
		Y:        0,
		Layer:    1,
		Name:     "svgfromcontent",
		Kind:     kind,
		Width:    width,
		Height:   height,
		FilePath: "",
		Content:  encodedContent,
	}
	return ret, nil
}

/**
* Creates a new SVGEmbeddedRaster object for embedding into the SVG object.
* Containins the base64 encoded contents of the raster file at the given path
* @param rasterFilePath string the absolute file path of the raster image to insert
* @param x, y the x and y coordinates (top left corner) of the raster image in the SVG file (default 0,0)
* @param layer the z depth of the embeded image in the svg, default to bottom-most (zero)
* @return An SVGEmbeddedRaster object, or error if the file could not be read
 */
func NewSVGEmbeddedRaster(rasterFilePath string, x, y, layer int) (*SVGEmbeddedRaster, error) {
	if rasterFilePath == "" {
		return nil, fmt.Errorf("rasterFilePath cannot be empty")
	}

	// Read the file content
	content, err := os.ReadFile(rasterFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SVG file: %w", err)
	}

	// Extract the filename without extension to use as the SVG name
	baseName := filepath.Base(rasterFilePath)

	// First determine the image type and dimensions from the raw content
	kind, width, height, err := DetermineImageType(baseName, content)
	if err != nil {
		return nil, fmt.Errorf("failed to determine image type: %w", err)
	}

	// Then base64 encode the content
	encodedContent := []byte(base64.StdEncoding.EncodeToString(content))

	ret := &SVGEmbeddedRaster{
		X:        x,
		Y:        y,
		Layer:    layer,
		Name:     baseName[:len(baseName)-len(filepath.Ext(baseName))],
		Kind:     kind,
		Width:    width,
		Height:   height,
		FilePath: rasterFilePath,
		Content:  encodedContent,
	}
	return ret, nil
}

/*
*
* Returns this object as an SVG image tag for embedding into an SVG file
* @return An SVG <image> tag containing the raster image
TODO decided which layer to put this on?
*/
func (s *SVGEmbeddedRaster) GetAsImageTag() (string, error) {
	if s.Content == nil {
		return "", fmt.Errorf("content is nil")
	}
	if s.Width == 0 || s.Height == 0 {
		return "", fmt.Errorf("width or height is zero")
	}
	ret := fmt.Sprintf(
		`<image x="%d" y="%d" width="%d" height="%d" xlink:href="data:image/%s;base64,%s" />`,
		s.X, s.Y, s.Width, s.Height, s.Kind, s.Content)
	return ret, nil
}

///////////////////////////////////////////////////////////////////////////////
/// SVGEmbeddedText
///////////////////////////////////////////////////////////////////////////////

// Holds information about text that is embedded into SVG files
type SVGEmbeddedText struct {
	Layer       int
	X, Y        int
	Name        string
	Content     string
	Style       string
	MaxWidth    int      // Maximum width for text wrapping
	LineSpacing float64  // Spacing between lines when wrapped
	Lines       []string // Text split into lines for wrapping
}

func NewSVGEmbeddedText(name, text, style string, x, y, layer int) (*SVGEmbeddedText, error) {
	// start by creating the embedded text
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}
	if style == "" {
		style = "font-size: 12px; font-family: Arial; fill: white;"
	}

	ret := &SVGEmbeddedText{
		Layer:       layer,
		X:           x,
		Y:           y,
		Name:        name,
		Content:     text,
		Style:       style,
		MaxWidth:    0,     // Default: no wrapping
		LineSpacing: 1.2,   // Default line spacing factor
		Lines:       []string{text}, // Default: single line
	}
	return ret, nil
}

///////////////////////////////////////////////////////////////////////////////
/// SVG
///////////////////////////////////////////////////////////////////////////////

const SvgHeader string = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<svg width="" height=""
    version="1.1"
	xmlns="http://www.w3.org/2000/svg"
	xmlns:svg="http://www.w3.org/2000/svg"
	xmlns:xlink="http://www.w3.org/1999/xlink">
`
const SvgFooter string = `
</svg>
`

// An object for holding, parsing, manipulating and writing SVG files
// We are interested only in Path primatives
type SVG struct {
	Filepath      string
	Name          string
	Images        []*SVGEmbeddedRaster
	Paths         *Paths
	Text          []*SVGEmbeddedText
	Width, Height int
}

func NewBlankSVG() (*SVG, error) {
	paths, err := NewPaths([]*Path{})
	if err != nil {
		return nil, err
	}
	return &SVG{
		Name:     "blank",
		Images:   []*SVGEmbeddedRaster{},
		Filepath: "",
		Paths:    paths,
		Text:     []*SVGEmbeddedText{},
	}, nil
}

// NewSVGFromFile reads an SVG file from the given filepath and creates a new SVG object
func NewSVGFromFile(filePath string) (*SVG, error) {
	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SVG file: %w", err)
	}

	// Extract the filename without extension to use as the SVG name
	baseName := filepath.Base(filePath)
	name := baseName[:len(baseName)-len(filepath.Ext(baseName))]

	// Use the existing NewSVG constructor with the file content
	return NewSVGFromContent(name, string(content))
}

func NewSVGFromRasterContent(content []byte) (*SVG, error) {
	// start by creating the embedded image
	i, err := NewSVGEmbeddedRasterContent(content)
	if err != nil {
		return nil, err
	}
	ret, err := NewBlankSVG()
	if err != nil {
		return nil, err
	}
	// Make our SVG the same size as the background image
	ret.Width = i.Width
	ret.Height = i.Height
	// Add the image to the SVG
	ret.Images = append(ret.Images, i)
	return ret, nil
}

/**
* Creates a new SVG image and embeds into it an <image> tag
* containing the base64 encoded contents of the raster file at the given path
* @param rasterFilePath string the absolute file path of the raster image to insert
* @return An SVG file containing the raster image, or an error if the file could not be read
 */
func NewSVGFromRaster(rasterFilePath string, x, y, layer int) (*SVG, error) {
	// start by creating the embedded image
	i, err := NewSVGEmbeddedRaster(rasterFilePath, x, y, layer)
	if err != nil {
		return nil, err
	}
	ret, err := NewBlankSVG()
	if err != nil {
		return nil, err
	}
	// Make our SVG the same size as the background image
	ret.Width = i.Width
	ret.Height = i.Height
	// Add the image to the SVG
	ret.Images = append(ret.Images, i)
	return ret, nil
}

// Converts the given svg file content into various structures
func NewSVGFromContent(name string, svgContent string) (*SVG, error) {
	// Regular expression to match the <path> tags
	pathRegex := regexp.MustCompile(`(?i)<path[^>]*>`)
	// Find all matches
	matches := pathRegex.FindAllString(svgContent, -1)

	// If no matches found, return an empty slice
	if len(matches) == 0 {
		return nil, fmt.Errorf("no <path> tags found in SVG content")
	}

	ret, err := NewBlankSVG()
	if err != nil {
		return nil, err
	}
	ret.Name = name

	// Parse each path tag into a Path object
	for _, pathTag := range matches {
		path, err := NewPathFromSvgTag(pathTag)
		if err != nil {
			// Log the error but continue processing other paths
			fmt.Printf("Warning: Failed to parse path tag: %v\n", err)
			continue
		}
		ret.Paths.AddPath(path)
	}

	if ret.Paths.NumPaths() == 0 {
		return nil, fmt.Errorf("failed to parse any valid paths from SVG content")
	}
	return ret, nil
}
func (s *SVG) AddText(name, text, style string, x, y, layer int) error {
	// start by creating the embedded text
	i, err := NewSVGEmbeddedText(name, text, style, x, y, layer)
	if err != nil {
		return err
	}
	s.Text = append(s.Text, i)
	return nil
}

// AddWrappedText adds text with automatic wrapping based on maxWidth
func (s *SVG) AddWrappedText(name, text, style string, x, y, maxWidth, lineSpacing, layer int) error {
	i, err := NewSVGEmbeddedText(name, text, style, x, y, layer)
	if err != nil {
		return err
	}
	
	i.MaxWidth = maxWidth
	i.LineSpacing = float64(lineSpacing) / 10.0 // Convert integer to float64 for line spacing
	
	// Split the text into lines based on maxWidth
	// This is a simple implementation - we'll need to estimate character width
	// based on font size from the style
	
	// Extract font size from style
	fontSize := 12 // Default font size
	fontSizeRegex := regexp.MustCompile(`font-size:\s*(\d+)px`)
	matches := fontSizeRegex.FindStringSubmatch(style)
	if len(matches) > 1 {
		if size, err := fmt.Sscanf(matches[1], "%d", &fontSize); err != nil || size == 0 {
			fontSize = 12 // Default if parsing fails
		}
	}
	
	// Estimate average character width (roughly 0.6 times font size)
	avgCharWidth := float64(fontSize) * 0.6
	
	// Calculate how many characters can fit in maxWidth
	charsPerLine := int(float64(maxWidth) / avgCharWidth)
	
	if charsPerLine > 0 && len(text) > charsPerLine {
		// Split text into words
		words := regexp.MustCompile(`\s+`).Split(text, -1)
		lines := []string{}
		currentLine := ""
		
		for _, word := range words {
			// Check if adding this word would exceed the line width
			if len(currentLine)+len(word)+1 <= charsPerLine || currentLine == "" {
				if currentLine != "" {
					currentLine += " "
				}
				currentLine += word
			} else {
				// Start a new line
				lines = append(lines, currentLine)
				currentLine = word
			}
		}
		
		// Add the last line if not empty
		if currentLine != "" {
			lines = append(lines, currentLine)
		}
		
		i.Lines = lines
	} else {
		i.Lines = []string{text}
	}
	
	s.Text = append(s.Text, i)
	return nil
}

func (s *SVG) ToSVGFile(filePath string) error {
	svgContent, err := s.ToSVG()
	if err != nil {
		return err
	}
	err = os.WriteFile(filePath, []byte(svgContent), 0644)
	if err != nil {
		return err
	}
	return nil
}

func (s *SVG) ToSVG() (string, error) {
	// Start with the SVG header
	ret := SvgHeader
	// alter SVG width and height
	ret = regexp.MustCompile(`width=""`).ReplaceAllString(ret, fmt.Sprintf(`width="%d"`, s.Width))
	ret = regexp.MustCompile(`height=""`).ReplaceAllString(ret, fmt.Sprintf(`height="%d"`, s.Height))
	// Add all images
	for _, image := range s.Images {
		imageTag, err := image.GetAsImageTag()
		if err != nil {
			return "", err
		}
		ret += imageTag
	}

	// Add all paths
	allpaths, err := s.Paths.ToSVG()
	if err != nil {
		return "", err
	}
	ret += allpaths

	// Extract font size for line spacing calculation
	getFontSize := func(style string) int {
		fontSize := 24 // Default font size
		fontSizeRegex := regexp.MustCompile(`font-size:\s*(\d+)px`)
		matches := fontSizeRegex.FindStringSubmatch(style)
		if len(matches) > 1 {
			fmt.Sscanf(matches[1], "%d", &fontSize)
		}
		return fontSize
	}

	// Add all text elements with wrapping support
	for _, text := range s.Text {
		if len(text.Lines) <= 1 {
			// Single line text
			ret += fmt.Sprintf(`<text x="%d" y="%d" style="%s">%s</text>`, 
				text.X, text.Y, text.Style, text.Content)
		} else {
			// Multi-line text
			fontSize := getFontSize(text.Style)
			lineHeight := int(float64(fontSize) * float64(text.LineSpacing))
			
			for i, line := range text.Lines {
				yPos := text.Y + (i * lineHeight)
				ret += fmt.Sprintf(`<text x="%d" y="%d" style="%s">%s</text>`, 
					text.X, yPos, text.Style, line)
			}
		}
	}
	
	// Add the SVG footer
	ret += SvgFooter
	return ret, nil
}

func (s *SVG) ToGRBL() (string, error) {
	return "", nil
}
