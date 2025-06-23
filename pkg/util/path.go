package util

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
)

///////////////////////////////////////////////////////////////////////////////
/// POINT
///////////////////////////////////////////////////////////////////////////////

// Point represents a 2D point with X and Y coordinates
type Point struct {
	X, Y float64
}

func NewPoint(x float64, y float64) *Point {
	ret := &Point{
		X: x,
		Y: y,
	}
	return ret
}

// /////////////////////////////////////////////////////////////////////////////
// / GCodeParameter
// /////////////////////////////////////////////////////////////////////////////
var order = [...]string{"O", "N", "G", "M", "X", "Y", "Z", "H", "I", "J", "K", "L", "A", "B", "C", "D", "E", "P", "Q", "R", "S", "T", "U", "V", "W", "F"}

// A single GCode parameter such as X8.562 or F100 etc.
type GCodeParameter struct {
	Letter string
	Value  float64
}

func NewGCodeParameter(letter string, value float64) (*GCodeParameter, error) {

	if letter == "" {
		return nil, fmt.Errorf("letter cannot be empty")
	}

	// Capitalize the letter parameter
	letter = strings.ToUpper(letter)

	// Check if the letter exists in the 'order' array
	letterExists := false
	for _, validLetter := range order {
		if letter == validLetter {
			letterExists = true
			break
		}
	}

	// Return error if letter is not valid
	if !letterExists {
		return nil, fmt.Errorf("invalid letter: %s", letter)
	}

	ret := &GCodeParameter{
		Letter: letter,
		Value:  value,
	}
	return ret, nil
}

// /////////////////////////////////////////////////////////////////////////////
// / GCode
// /////////////////////////////////////////////////////////////////////////////

// A Single GCode Command such as G01 X5.387 etc.
// Somewhat similar to an SVG PathCommand
type GCode struct {
	Letter string
	Params []GCodeParameter
}

func NewGCode(params []GCodeParameter, letter string) (*GCode, error) {
	ret := &GCode{
		Letter: letter,
		Params: params,
	}
	// Order the parameters
	err := ret.OrderParameters()
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// Orders the GCode Parameters ensuring that our GCode is easier to read by a human
func (c *GCode) OrderParameters() error {
	if c == nil || len(c.Params) == 0 {
		return fmt.Errorf("GCode is nil or has no parameters")
	}

	// Create a map to store the order index of each letter
	orderMap := make(map[string]int)
	for i, letter := range order {
		orderMap[letter] = i
	}

	// Sort the parameters based on the order defined in the 'order' array
	sort.Slice(c.Params, func(i, j int) bool {
		// Get the order index for each parameter's letter
		iOrder, iExists := orderMap[c.Params[i].Letter]
		jOrder, jExists := orderMap[c.Params[j].Letter]

		// If either letter doesn't exist in the order map, put it at the end
		if !iExists {
			return false
		}
		if !jExists {
			return true
		}

		// Compare the order indices
		return iOrder < jOrder
	})

	return nil
}

// Converts this GRBL block into an SVG Path tag
func (c *GCode) ToSvgPath() (*Path, error) {
	return nil, nil
}

///////////////////////////////////////////////////////////////////////////////
/// PATH COMMAND
///////////////////////////////////////////////////////////////////////////////

/**
* A single SVG Path Command (from the d attribute) such as 'M 5.387,5.387' etc.
 */
type PathCommand struct {
	Letter string
	Params []float64
	Points []*Point
}

/**
* Creates a new PathCommand from the given cmd string
* @param cmd string the command string such as 'M 6,5' etc
 */
func NewPathCommand(cmd string) (*PathCommand, error) {
	if cmd == "" {
		return nil, fmt.Errorf("command string cannot be empty")
	}

	// Extract the first character as the command letter
	if len(cmd) < 1 {
		return nil, fmt.Errorf("command string too short")
	}

	letter := string(cmd[0])

	// Validate that the first character is a valid SVG path command letter
	validLetters := "MLHVCSQTAZmlhvcsqtaz"
	if !strings.Contains(validLetters, letter) {
		return nil, fmt.Errorf("invalid command letter: %s", letter)
	}

	// Extract parameters (numbers) from the command
	// First, remove the command letter and trim spaces
	paramsStr := ""
	if len(cmd) > 1 {
		paramsStr = strings.TrimSpace(cmd[1:])
	}

	// Parse parameters
	var params []float64

	if paramsStr != "" {
		// Replace commas with spaces for consistent splitting
		paramsStr = strings.ReplaceAll(paramsStr, ",", " ")

		// Split by whitespace and parse each number
		parts := regexp.MustCompile(`\s+`).Split(paramsStr, -1)

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// Parse the number
			var val float64
			_, err := fmt.Sscanf(part, "%f", &val)
			if err != nil {
				return nil, fmt.Errorf("invalid parameter value: %s", part)
			}

			params = append(params, val)
		}
	}
	// calculate if there are the correct number of parameters by command letter
	switch c := letter; c {
	case "Z", "z":
		if len(params) != 0 {
			return nil, fmt.Errorf("command %s requires exactly 0 parameters", c)
		}
	case "V", "v", "H", "h":
		if len(params) != 1 {
			return nil, fmt.Errorf("command %s requires exactly 1 parameter", c)
		}
	case "M", "m", "L", "l":
		if len(params) != 2 {
			return nil, fmt.Errorf("command %s requires exactly 2 parameters", c)
		}
	case "Q", "q":
		if len(params) != 4 {
			return nil, fmt.Errorf("command %s requires exactly 4 parameters", c)
		}
	case "A", "a":
		if len(params) != 7 {
			return nil, fmt.Errorf("command %s requires exactly 7 parameters", c)
		}
	default:
		return nil, fmt.Errorf("command letter %s not currently supported", c)
	}

	// Create and return the PathCommand
	return &PathCommand{
		Letter: letter,
		Params: params,
		Points: []*Point{},
	}, nil
}

// calculates the final coordinate (point) of this command
func (pc *PathCommand) GetFinishPoint(prev *PathCommand) (*Point, error) {
	if pc.Letter == "" || pc.Params == nil {
		return nil, fmt.Errorf("This PathCommand is not instantiated correctly yet")
	}

	// get the 'current' point if possible
	var pp *Point
	var pperr error
	pp, pperr = prev.GetFinishPoint((*PathCommand)(nil))

	if StringIsLower(pc.Letter) && (prev == nil || pp == nil || pperr != nil) {
		return nil, fmt.Errorf("Cannot calculate relative positions (lower case commands) without a prevous command")
	}

	// here create a switch case of all possible letters for a Path Command
	switch c := pc.Letter; c {
	case "M", "L":
		return NewPoint(pc.Params[0], pc.Params[1]), nil
	case "m", "l":
		return NewPoint(pp.X+pc.Params[0], pp.Y+pc.Params[1]), nil
	case "H":
		return NewPoint(pp.X, pc.Params[1]), nil
	case "V":
		return NewPoint(pc.Params[0], pp.Y), nil
	case "h":
		return NewPoint(pp.X+pc.Params[0], pc.Params[1]), nil
	case "v":
		return NewPoint(pc.Params[0], pp.Y+pc.Params[1]), nil
	case "Q":
		return NewPoint(pc.Params[2], pc.Params[3]), nil
	case "q":
		return NewPoint(pp.X+pc.Params[2], pp.Y+pc.Params[3]), nil
	case "A":
		return NewPoint(pc.Params[6], pc.Params[7]), nil
	case "a":
		return NewPoint(pp.X+pc.Params[6], pp.Y+pc.Params[7]), nil
	case "Z", "z":
		return nil, fmt.Errorf("Can't calculate the finish point of a Z command without the initial point of the path")
	default:
		return nil, fmt.Errorf("command letter %s not currently supported", c)
	}
}

// populates the PathCommand's Points field by dividing up the path described by this command into
// points along that path no further apart than maxDistance.
func (pc *PathCommand) PointaliseByDistance(prev *PathCommand, maxDistance float64) error {
	if pc.Letter == "" || pc.Params == nil {
		return fmt.Errorf("This PathCommand is not instantiated correctly yet")
	}

	// get the 'current' point if possible
	var pp *Point
	var pperr error
	pp, pperr = prev.GetFinishPoint((*PathCommand)(nil))

	if prev == nil || pp == nil || pperr != nil {
		return fmt.Errorf("Cannot pointalise a path command without knowing the previous command")
	}

	// here create a switch case of all possible letters for a Path Command
	switch c := pc.Letter; c {
	case "M", "m":
		// Move commands don't create paths, they just move the current point
		// Just add the start and end points
		pc.Points = []*Point{pp}
		endPoint, err := pc.GetFinishPoint(prev)
		if err == nil {
			pc.Points = append(pc.Points, endPoint)
		}
		return nil

	case "L", "l":
		// Line commands - create points along a straight line
		startPoint := pp
		endPoint, err := pc.GetFinishPoint(prev)
		if err != nil {
			return err
		}

		// Calculate distance between points
		dx := endPoint.X - startPoint.X
		dy := endPoint.Y - startPoint.Y
		distance := math.Sqrt(dx*dx + dy*dy)

		// Calculate number of points needed
		numPoints := int(math.Ceil(distance/maxDistance)) + 1
		if numPoints < 2 {
			numPoints = 2
		}

		// Generate points along the line
		pc.Points = make([]*Point, numPoints)
		for i := 0; i < numPoints; i++ {
			t := float64(i) / float64(numPoints-1)
			x := startPoint.X + t*dx
			y := startPoint.Y + t*dy
			pc.Points[i] = NewPoint(x, y)
		}
		return nil

	case "H", "h":
		// Horizontal line - create points along a horizontal line
		startPoint := pp
		endPoint, err := pc.GetFinishPoint(prev)
		if err != nil {
			return err
		}

		// Calculate distance
		distance := math.Abs(endPoint.X - startPoint.X)

		// Calculate number of points needed
		numPoints := int(math.Ceil(distance/maxDistance)) + 1
		if numPoints < 2 {
			numPoints = 2
		}

		// Generate points along the horizontal line
		pc.Points = make([]*Point, numPoints)
		for i := 0; i < numPoints; i++ {
			t := float64(i) / float64(numPoints-1)
			x := startPoint.X + t*(endPoint.X-startPoint.X)
			pc.Points[i] = NewPoint(x, startPoint.Y)
		}
		return nil

	case "V", "v":
		// Vertical line - create points along a vertical line
		startPoint := pp
		endPoint, err := pc.GetFinishPoint(prev)
		if err != nil {
			return err
		}

		// Calculate distance
		distance := math.Abs(endPoint.Y - startPoint.Y)

		// Calculate number of points needed
		numPoints := int(math.Ceil(distance/maxDistance)) + 1
		if numPoints < 2 {
			numPoints = 2
		}

		// Generate points along the vertical line
		pc.Points = make([]*Point, numPoints)
		for i := 0; i < numPoints; i++ {
			t := float64(i) / float64(numPoints-1)
			y := startPoint.Y + t*(endPoint.Y-startPoint.Y)
			pc.Points[i] = NewPoint(startPoint.X, y)
		}
		return nil

	case "Q", "q":
		// Quadratic Bezier curve
		startPoint := pp

		// Extract control point and end point
		var controlPoint, endPoint Point

		if c == "Q" {
			// Absolute coordinates
			controlPoint = Point{X: pc.Params[0], Y: pc.Params[1]}
			endPoint = Point{X: pc.Params[2], Y: pc.Params[3]}
		} else {
			// Relative coordinates
			controlPoint = Point{X: pp.X + pc.Params[0], Y: pp.Y + pc.Params[1]}
			endPoint = Point{X: pp.X + pc.Params[2], Y: pp.Y + pc.Params[3]}
		}

		// Use the QuadraticBezierByDistance function to generate points
		bezierPoints := QuadraticBezierByDistance(*startPoint, controlPoint, endPoint, maxDistance)

		// Copy points to the PathCommand
		pc.Points = bezierPoints
		return nil

	case "A", "a":
		// Elliptical arc
		startPoint := pp

		// Extract arc parameters
		var rx, ry, xAxisRotation float64
		var largeArcFlag, sweepFlag bool
		var endPoint Point

		if c == "A" {
			// Absolute coordinates
			rx = pc.Params[0]
			ry = pc.Params[1]
			xAxisRotation = pc.Params[2] * math.Pi / 180.0 // Convert degrees to radians
			largeArcFlag = pc.Params[3] != 0
			sweepFlag = pc.Params[4] != 0
			endPoint = Point{X: pc.Params[5], Y: pc.Params[6]}
		} else {
			// Relative coordinates
			rx = pc.Params[0]
			ry = pc.Params[1]
			xAxisRotation = pc.Params[2] * math.Pi / 180.0 // Convert degrees to radians
			largeArcFlag = pc.Params[3] != 0
			sweepFlag = pc.Params[4] != 0
			endPoint = Point{X: pp.X + pc.Params[5], Y: pp.Y + pc.Params[6]}
		}

		// Create an elliptical arc
		arc := NewEllipticalArc(*startPoint, endPoint, rx, ry, xAxisRotation, sweepFlag, largeArcFlag)

		// Generate points along the arc
		arcPoints := arc.GeneratePointsByDistance(maxDistance)

		// Convert to pointers
		pc.Points = make([]*Point, len(arcPoints))
		for i, p := range arcPoints {
			pc.Points[i] = NewPoint(p.X, p.Y)
		}
		return nil

	case "Z", "z":
		return fmt.Errorf("Z command requires knowing the first point of the path")

	default:
		return fmt.Errorf("command letter %s not currently supported for pointalisation", c)
	}
}

///////////////////////////////////////////////////////////////////////////////
/// PATH
///////////////////////////////////////////////////////////////////////////////

/**
* Represents the information contained in a single SVG '<path>' tag
 */
type Path struct {
	ID          string
	Points      []*Point
	PathTag     string
	CommandsStr string
	Commands    []*PathCommand
	IsClosed    bool
}

func NewPathFromPoints(points []*Point, id string) (*Path, error) {
	if points == nil || len(points) == 0 {
		return nil, fmt.Errorf("must supply an array of Points to this constructor")
	}
	if id == "" {
		id = "pathFromPoints"
	}

	// Create a simple path command string from the points
	var commandsStr strings.Builder

	// Start with a move to the first point
	commandsStr.WriteString(fmt.Sprintf("M %.6f,%.6f ", points[0].X, points[0].Y))

	// Add line commands for the rest of the points
	for i := 1; i < len(points); i++ {
		commandsStr.WriteString(fmt.Sprintf("L %.6f,%.6f ", points[i].X, points[i].Y))
	}

	// Create new Path instance
	ret := &Path{
		ID:          id,
		Points:      points,
		PathTag:     "",
		CommandsStr: commandsStr.String(),
		Commands:    []*PathCommand{},
		IsClosed:    false,
	}

	return ret, nil
}

// Constructor from an SVG <path ... /> tag
func NewPathFromSvgTag(tag string) (*Path, error) {
	if tag == "" {
		return nil, fmt.Errorf("tag cannot be empty")
	}
	// Create new Path instance
	ret := &Path{
		ID:          "",
		Points:      nil,
		PathTag:     tag,
		CommandsStr: "",
		Commands:    []*PathCommand{},
		IsClosed:    false,
	}
	err := ret.ParseSvgPathTag()
	if err != nil {
		return nil, err
	}
	// TODO auto pointalise
	return ret, nil
}

func (p *Path) ParsePathCommands() error {
	if p.CommandsStr == "" {
		return fmt.Errorf("Path must have a populated CommandsStr field before this method is called")
	}

	// Regular expression to match path commands: a letter followed by numbers
	// This regex captures each command letter and its associated parameters
	commandRegex := regexp.MustCompile(`([MLHVCSQTAZmlhvcsqtaz])[\s,]*([^MLHVCSQTAZmlhvcsqtaz]*)`)

	// Find all matches
	matches := commandRegex.FindAllStringSubmatch(p.CommandsStr, -1)

	// If no matches found, return an error
	if len(matches) == 0 {
		return fmt.Errorf("no valid path commands found")
	}

	// Parse each command
	commands := make([]*PathCommand, 0, len(matches))
	for _, match := range matches {
		if len(match) >= 2 {
			cmdStr := match[1]
			if len(match) >= 3 && match[2] != "" {
				cmdStr += " " + strings.TrimSpace(match[2])
			}

			cmd, err := NewPathCommand(cmdStr)
			if err != nil {
				return fmt.Errorf("failed to parse command '%s': %v", cmdStr, err)
			}
			commands = append(commands, cmd)
		}
	}
	// modify the instance
	p.Commands = commands
	return nil
}

func (p *Path) ParseSvgPathTag() error {
	// Validate that it's a path tag using regex
	if p.PathTag == "" {
		return fmt.Errorf("Path object must have a populated PathTag field before this method is called")
	}
	pathRegex := regexp.MustCompile(`(?i)<path[^>]*>`)
	if !pathRegex.MatchString(p.PathTag) {
		return fmt.Errorf("invalid SVG path tag format")
	}

	// Regular expressions to extract d and id attributes
	dr := regexp.MustCompile(`(?i)\sd\s*=\s*[?:'|"]([^"']*)[?:'|"]`)   // d="value"
	idr := regexp.MustCompile(`(?i)\sid\s*=\s*[?:'|"]([^"']*)[?:'|"]`) // id="value"

	// Extract the d attribute (path commands)
	dMatches := dr.FindStringSubmatch(p.PathTag)
	if len(dMatches) >= 2 {
		p.CommandsStr = dMatches[1] // Get the captured group (the actual value)

		// Parse the commands
		err := p.ParsePathCommands()
		if err != nil {
			return fmt.Errorf("failed to parse path commands: %v", err)
		}
	} else {
		return fmt.Errorf("no valid path commands found")
	}

	// Extract the id attribute
	idMatches := idr.FindStringSubmatch(p.PathTag)
	if len(idMatches) >= 2 {
		p.ID = idMatches[1] // Get the captured group (the actual value)
	}

	// Check if path is closed (ends with Z or z)
	if len(p.Commands) > 0 {
		lastCmd := p.Commands[len(p.Commands)-1]
		if lastCmd.Letter == "Z" || lastCmd.Letter == "z" {
			p.IsClosed = true
		}
	}
	// TODO somehow check what the current XY is and see if it is the same
	// as the last path command such that the path is closed
	return nil
}

// Converts this path object to GRBL
func (p *Path) ToGCode() (string, error) {
	return "", nil
}

func (p *Path) ToPathTag() (string, error) {
	// if the path tag is populated, just return it
	if p.PathTag != "" {
		return p.PathTag, nil
	}
	// if the path tag is not populated, try to create it
	if p.CommandsStr == "" {
		err := p.ParsePathCommands()
		if err != nil {
			return "", err
		}
	}
	// if the path tag is not populated, try to create it
	if p.CommandsStr != "" {
		p.PathTag = fmt.Sprintf("<path id=\"%s\" d=\"%s\" />", p.ID, p.CommandsStr)
		return p.PathTag, nil
	}

	if p.Points != nil && len(p.Points) > 0 {
		logger.Warn("Should be compiling path commands from Points array but not implemented yet")
	}

	return "", fmt.Errorf("Path object must have a populated PathTag field or CommandsStr field before this method is called")
}

///////////////////////////////////////////////////////////////////////////////
/// PATHS
///////////////////////////////////////////////////////////////////////////////

// Holds information about paths, which is an array of Path structures
type Paths struct {
	Paths []*Path
}

func NewPaths(paths []*Path) (*Paths, error) {
	ret := &Paths{}
	if paths == nil || len(paths) == 0 {
		ret.Paths = []*Path{}
	} else {
		ret.Paths = paths
	}
	return ret, nil
}

func (p *Paths) NumPaths() int {
	if p.Paths == nil {
		return 0
	}
	return len(p.Paths)
}

func (p *Paths) AddPath(path *Path) {
	p.Paths = append(p.Paths, path)
}

// Renders all paths in this object to a linebreak delimited string
// of SVG <path> tags
func (p *Paths) ToSVG() (string, error) {
	ret := ""
	for _, path := range p.Paths {
		path, err := path.ToPathTag()
		if err != nil {
			return "", err
		}
		ret += path + "\n"
	}
	return ret, nil
}

///////////////////////////////////////////////////////////////////////////////
/// UTIL
///////////////////////////////////////////////////////////////////////////////

func StringIsUpper(s string) bool {
	for _, charNumber := range s {
		if charNumber > 90 || charNumber < 65 {
			return false
		}
	}
	return true
}

func StringIsLower(s string) bool {
	for _, charNumber := range s {
		if charNumber > 122 || charNumber < 97 {
			return false
		}
	}
	return true
}

// GetWorkingDirectory returns the present working directory
func GetWorkingDirectory() (string, error) {
	// Use the os package to get the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return dir, nil
}
