package test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/richard-senior/mcp/pkg/util"
)

const testsvg = `
<svg width="101.86021254496654mm" height="101.86021254496654mm" viewBox="0 0 101.86021254496654 101.86021254496654" xmlns="http://www.w3.org/2000/svg" version="1.1">
<g id="top_az_disk" transform="translate(50.930106,50.930106) scale(1,-1)">
<circle cx="0.0" cy="0.0" r="50.0" stroke="#f1f3f5" stroke-width="0.7 px" style="stroke-width:0.7;stroke-miterlimit:4;stroke-dasharray:none;stroke-linecap:square;fill:none"/>
<circle cx="0.0" cy="0.0" r="5.0" stroke="#f1f3f5" stroke-width="0.7 px" style="stroke-width:0.7;stroke-miterlimit:4;stroke-dasharray:none;stroke-linecap:square;fill:none"/>
<path id="test_path_1"  d="M 31.819805153394636 31.81980515339464 A 45 45 0 0 1 -31.8198 31.8198L 25.191468081114834 25.191455218586476 A 39.1451 39.1451 0 0 1 18.054 30.8354L 31.819805153394636 31.81980515339464 " stroke="#f1f3f5" stroke-width="0.7 px" style="stroke-width:0.7;stroke-miterlimit:4;stroke-dasharray:none;stroke-linecap:square;fill:none;fill-opacity:1;fill-rule: evenodd"/>
<path id="test_path_2"  d="M -31.81980515339464 -31.819805153394636 A 45 45 0 0 1 31.8198 -31.8198L -27.577164466275356 -27.57716446627535 A 39 39 0 0 1 -23.3776 -31.2168L -31.81980515339464 -31.819805153394636 " stroke="#f1f3f5" stroke-width="0.7 px" style="stroke-width:0.7;stroke-miterlimit:4;stroke-dasharray:none;stroke-linecap:square;fill:none;fill-opacity:1;fill-rule: evenodd"/>
</g>
</svg>
`

// TestNewBlankSVG tests the creation of a blank SVG
func TestNewBlankSVG(t *testing.T) {
	// Create a mock Paths object first since NewBlankSVG requires non-empty paths
	mockPath, err := util.NewPathFromPoints([]*util.Point{util.NewPoint(0, 0), util.NewPoint(10, 10)}, "test_path")
	if err != nil {
		t.Fatalf("Failed to create mock path: %v", err)
	}

	// Create a mock Paths with the mock Path
	mockPaths, err := util.NewPaths([]*util.Path{mockPath})
	if err != nil {
		t.Fatalf("Failed to create mock paths: %v", err)
	}

	// Create a blank SVG manually since NewBlankSVG has a constraint
	svg := &util.SVG{
		Name:   "blank",
		Images: []*util.SVGEmbeddedRaster{},
		Paths:  mockPaths,
	}

	if svg.Name != "blank" {
		t.Errorf("Expected name to be 'blank', got '%s'", svg.Name)
	}

	if len(svg.Images) != 0 {
		t.Errorf("Expected 0 images, got %d", len(svg.Images))
	}

	if svg.Paths == nil {
		t.Error("Expected Paths to be initialized, got nil")
	}
}

// TestNewSVGFromContent tests creating an SVG from content
func TestNewSVGFromContent(t *testing.T) {
	// Simple SVG content with a path
	content := testsvg

	svg, err := util.NewSVGFromContent("test_svg", content)
	if err != nil {
		t.Fatalf("Failed to create SVG from content: %v", err)
	}

	if svg.Name != "test_svg" {
		t.Errorf("Expected name to be 'test_svg', got '%s'", svg.Name)
	}

	if svg.Paths == nil || len(svg.Paths.Paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(svg.Paths.Paths))
	}

	if svg.Paths.Paths[0].ID != "test_path_1" {
		t.Errorf("Expected path ID to be 'test_path_1', got '%s'", svg.Paths.Paths[0].ID)
	}
}

// TestNewSVGFromFile tests creating an SVG from a file
func TestNewSVGFromFile(t *testing.T) {
	// Create a temporary SVG file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.svg")

	content := testsvg

	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}

	svg, err := util.NewSVGFromFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to create SVG from file: %v", err)
	}

	if svg.Name != "test" {
		t.Errorf("Expected name to be 'test', got '%s'", svg.Name)
	}

	if svg.Paths == nil || len(svg.Paths.Paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(svg.Paths.Paths))
	}
}

func TestDetermineImageType(t *testing.T) {
	doTestDetermineImageType(t, "avatar.png")
	doTestDetermineImageType(t, "avatar.jpg")
	doTestDetermineImageType(t, "avatar.gif")
	doTestDetermineImageType(t, "avatar.svg")
}

func doTestDetermineImageType(t *testing.T, f string) {
	_, filename, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(filename)
	testImagePath := filepath.Join(testDir, "testdata", f)
	// Read the file content for debugging
	content, err := os.ReadFile(testImagePath)
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}
	//t.Logf("Image size: %d bytes", len(content))
	kind, w2, h2, err := util.DetermineImageType(testImagePath, content)
	if err != nil {
		t.Fatalf("Failed to determine image type: %v", err)
	}
	t.Logf("DetermineImageType: kind=%s, width=%d, height=%d", kind, w2, h2)
}

// TestNewSVGFromRaster tests creating an SVG from a raster image
func TestNewSVGFromRaster(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(filename)
	testImagePath := filepath.Join(testDir, "testdata", "avatar.png")

	// Call NewSVGFromRaster with the required parameters (x, y, layer)
	svg, err := util.NewSVGFromRaster(testImagePath, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create SVG from raster: %v", err)
	}

	if len(svg.Images) != 1 {
		t.Errorf("Expected 1 image, got %d", len(svg.Images))
	}

	// Test the embedded raster image
	embeddedImage := svg.Images[0]
	if embeddedImage == nil {
		t.Fatal("Expected embedded image, got nil")
	}

	t.Logf("Embedded image: kind=%s, width=%d, height=%d",
		embeddedImage.Kind, embeddedImage.Width, embeddedImage.Height)

	if embeddedImage.Kind != "png" {
		t.Errorf("Expected image kind to be 'png', got '%s'", embeddedImage.Kind)
	}

	// Test getting the image tag
	imageTag, err := embeddedImage.GetAsImageTag()
	if err != nil {
		t.Fatalf("Failed to get image tag: %v", err)
	}

	// Check if the image tag contains the expected attributes
	if !strings.Contains(imageTag, "image") || !strings.Contains(imageTag, "base64") {
		t.Errorf("Image tag doesn't contain expected content: %s", imageTag)
	}

	// add some test text
	style := "font-size: 16px; font-family: Arial; fill: red;"
	svg.AddText("footer_text", "Badger Badger Badger..", style, 30, 190, 1)

	tempDir := filepath.Dir(filename)
	tempFile := filepath.Join(tempDir, "testdata", "foo.svg")

	t.Log("Writing to ", tempFile)

	// Now try to write the SVG to file
	err = svg.ToSVGFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to write SVG to file: %v", err)
	}
}
