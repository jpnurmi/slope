package tui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func testPNG(w, h int, c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func TestRenderImage(t *testing.T) {
	data := testPNG(4, 4, color.RGBA{255, 0, 0, 255})
	got, err := renderImage(data, 10, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "▀") {
		t.Error("output should contain half-block characters")
	}
	if !strings.Contains(got, "\x1b[") {
		t.Error("output should contain ANSI escape sequences")
	}
	if !strings.Contains(got, "\x1b[0m") {
		t.Error("output should contain reset sequences")
	}

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if !strings.HasSuffix(line, "\x1b[0m") {
			t.Errorf("line should end with reset: %q", line)
		}
	}
}

func TestRenderImageFitsWidth(t *testing.T) {
	data := testPNG(100, 10, color.White)
	got, err := renderImage(data, 50, 40)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	for _, line := range lines {
		// Count half-block characters (each is one column)
		blocks := strings.Count(line, "▀")
		if blocks > 50 {
			t.Errorf("line has %d blocks, want <= 50", blocks)
		}
	}
}

func TestRenderImageFitsHeight(t *testing.T) {
	// Tall image: 10x200, terminal 50 wide, 20 high
	data := testPNG(10, 200, color.White)
	got, err := renderImage(data, 50, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) > 20 {
		t.Errorf("got %d lines, want <= 20", len(lines))
	}
}

func TestRenderImageInvalidData(t *testing.T) {
	_, err := renderImage([]byte("not an image"), 80, 24)
	if err == nil {
		t.Error("expected error for invalid image data")
	}
}

func TestRenderImagePreservesColor(t *testing.T) {
	data := testPNG(1, 2, color.RGBA{0, 128, 255, 255})
	got, err := renderImage(data, 1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should contain the color values in the ANSI sequence
	if !strings.Contains(got, "0;128;255m") {
		t.Errorf("expected color 0;128;255 in output, got %q", got)
	}
}
