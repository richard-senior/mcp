package test

import (
	"testing"

	"github.com/richard-senior/mcp/pkg/util/podds"
)

func TestUpdate(t *testing.T) {
	podds := podds.NewPodds()
	podds.Update()
}
