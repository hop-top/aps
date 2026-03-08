package mobile

import (
	"fmt"
	"os"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/term"
)

// QRRenderMode determines how the QR code is rendered
type QRRenderMode int

const (
	QRRenderTerminal QRRenderMode = iota
	QRRenderPNG
	QRRenderCodeOnly
)

// QRTerminalSize holds the minimum terminal requirements for QR display
type QRTerminalSize struct {
	MinCols int
	MinRows int
}

// TerminalFits checks if the current terminal can display a QR code
func TerminalFits(moduleCount int) (bool, int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return false, 0, 0
	}
	// Unicode half-blocks double vertical density, so each module = 1 col, 0.5 rows
	// Add quiet zone (4 modules each side)
	requiredCols := moduleCount + 8 // 4 quiet zone each side
	requiredRows := (moduleCount+8)/2 + 1
	return width >= requiredCols && height >= requiredRows, width, height
}

// GenerateQRPNG generates a QR code PNG file
func GenerateQRPNG(content string, filename string, size int) error {
	return qrcode.WriteFile(content, qrcode.Medium, size, filename)
}

// GenerateQRTerminal generates a QR code as a terminal string using Unicode half-blocks
func GenerateQRTerminal(content string) (string, error) {
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code: %w", err)
	}

	bitmap := qr.Bitmap()
	return renderBitmapHalfBlocks(bitmap), nil
}

// renderBitmapHalfBlocks renders a boolean bitmap using Unicode half-block characters.
// Each character represents two vertical pixels, doubling vertical density.
// Uses inverted colors (white background, black modules) for QR readability.
func renderBitmapHalfBlocks(bitmap [][]bool) string {
	var sb strings.Builder
	rows := len(bitmap)

	for y := 0; y < rows; y += 2 {
		sb.WriteString("  ") // left margin
		for x := 0; x < len(bitmap[y]); x++ {
			top := bitmap[y][x]
			bottom := false
			if y+1 < rows {
				bottom = bitmap[y+1][x]
			}

			// QR: true = black module, false = white
			// Terminal: we want black-on-white for contrast
			switch {
			case top && bottom:
				sb.WriteRune('\u2588') // full block (both black)
			case top && !bottom:
				sb.WriteRune('\u2580') // upper half block
			case !top && bottom:
				sb.WriteRune('\u2584') // lower half block
			default:
				sb.WriteRune(' ') // both white
			}
		}
		sb.WriteRune('\n')
	}

	return sb.String()
}

// QRModuleCount returns the number of modules for a QR code with the given content
func QRModuleCount(content string) (int, error) {
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return 0, err
	}
	return len(qr.Bitmap()), nil
}
