package test

import (
	"testing"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/util/podds"
)

// TestNewBlankSVG tests the creation of a blank SVG
func TestFotmobDatasource(t *testing.T) {
	fmds := podds.NewFotmobDatasource()
	matches, err := fmds.GetMatches(47, "2024/2025")
	if err != nil {
		t.Error(err)
	}
	if len(matches) == 0 {
		return
	}
	logger.Info("Matches Found : ", len(matches))
	logger.Info("first match : \n", matches[0])
}
