package test

import (
	"testing"

	"github.com/richard-senior/mcp/pkg/util/podds"
)

// TestDatasource tests the datasource functionality
func TestDatasource(t *testing.T) {
	p := podds.NewPodds()
	err := p.Update()
	if err != nil {
		t.Error(err)
	}
}
