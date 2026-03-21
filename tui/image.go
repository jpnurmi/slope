package tui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"golang.org/x/image/draw"
)

func renderImage(data []byte, width, height int) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW == 0 || srcH == 0 {
		return "", fmt.Errorf("empty image")
	}

	// Scale to fit terminal, preserving aspect ratio.
	// Each character cell shows 2 pixel rows via half-blocks.
	maxH := height * 2
	dstW := width
	dstH := (srcH * dstW) / srcW
	if dstH > maxH {
		dstH = maxH
		dstW = (srcW * dstH) / srcH
	}
	if dstW < 1 {
		dstW = 1
	}
	if dstH < 2 {
		dstH = 2
	}

	// Ensure even height for half-block pairing
	if dstH%2 != 0 {
		dstH++
	}

	resized := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.BiLinear.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)

	var b strings.Builder
	for y := 0; y < dstH; y += 2 {
		for x := 0; x < dstW; x++ {
			top := rgbColor(resized.At(x, y))
			bot := rgbColor(resized.At(x, y+1))
			// Upper half block: fg = top pixel, bg = bottom pixel
			fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀",
				top[0], top[1], top[2],
				bot[0], bot[1], bot[2])
		}
		b.WriteString("\x1b[0m\n")
	}
	return b.String(), nil
}

func rgbColor(c color.Color) [3]uint8 {
	r, g, b, _ := c.RGBA()
	return [3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
}
