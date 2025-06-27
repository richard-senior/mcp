package test

import (
	"testing"

	"github.com/richard-senior/mcp/pkg/util/podds"
)

// TestNewBlankSVG tests the creation of a blank SVG
func TestFotmobDatasource(t *testing.T) {
	p := podds.NewPodds()
	err := p.Update()
	if err != nil {
		t.Error(err)
	}
}
