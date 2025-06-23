package util

/**
 * Tools for accessing local SQLite databases
 */

// A Single GCode Command such as G01 X5.387 etc.
// Somewhat similar to an SVG PathCommand
type SQLiteClient struct {
}

func NewSQlite(dbLocation string) (*SQLiteClient, error) {
	ret := &SQLiteClient{}
	return ret, nil
}

// Orders the GCode Parameters ensuring that our GCode is easier to read by a human
func (c *SQLiteClient) Execute(query string) error {
	return nil
}
