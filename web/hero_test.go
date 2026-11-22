package web

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const (
	heroPath     = "static/images/hero.webp"
	heroWidth    = 1600
	heroHeight   = 900
	heroMaxBytes = 400 * 1024
)

// webpDimensions reads the canvas size from a WebP file header. It understands the
// lossy (VP8), lossless (VP8L) and extended (VP8X) container layouts.
func webpDimensions(data []byte) (width, height int, err error) {
	if len(data) < 20 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return 0, 0, errors.New("not a RIFF/WEBP file")
	}
	if riffSize := binary.LittleEndian.Uint32(data[4:8]); int(riffSize)+8 != len(data) {
		return 0, 0, errors.New("RIFF size does not match the file length")
	}
	chunk := string(data[12:16])
	payload := data[20:]
	switch chunk {
	case "VP8 ":
		// 3-byte frame tag, 3-byte start code, then 14-bit width and height.
		if len(payload) < 10 {
			return 0, 0, errors.New("VP8 chunk too short")
		}
		if payload[3] != 0x9d || payload[4] != 0x01 || payload[5] != 0x2a {
			return 0, 0, errors.New("VP8 start code missing")
		}
		width = int(binary.LittleEndian.Uint16(payload[6:8]) & 0x3fff)
		height = int(binary.LittleEndian.Uint16(payload[8:10]) & 0x3fff)
	case "VP8L":
		// 1-byte signature, then 14-bit (width-1) and 14-bit (height-1).
		if len(payload) < 5 {
			return 0, 0, errors.New("VP8L chunk too short")
		}
		if payload[0] != 0x2f {
			return 0, 0, errors.New("VP8L signature missing")
		}
		bits := binary.LittleEndian.Uint32(payload[1:5])
		width = int(bits&0x3fff) + 1
		height = int((bits>>14)&0x3fff) + 1
	case "VP8X":
		// 4 bytes of flags and reserved bits, then 24-bit (width-1) and (height-1).
		if len(payload) < 10 {
			return 0, 0, errors.New("VP8X chunk too short")
		}
		width = int(uint32(payload[4])|uint32(payload[5])<<8|uint32(payload[6])<<16) + 1
		height = int(uint32(payload[7])|uint32(payload[8])<<8|uint32(payload[9])<<16) + 1
	default:
		return 0, 0, errors.New("unknown WebP chunk " + chunk)
	}
	return width, height, nil
}

// TestHeroImage checks the delivered landing page hero (plan 8.8.4): it exists, is a
// WebP file, is exactly 1600x900 and stays within the size limit.
func TestHeroImage(t *testing.T) {
	data, err := os.ReadFile(filepath.FromSlash(heroPath))
	if err != nil {
		t.Fatalf("read %s: %v (generate it with python graphics/generate_graphics.py hero)", heroPath, err)
	}
	if len(data) > heroMaxBytes {
		t.Errorf("%s is %d bytes, the limit is %d", heroPath, len(data), heroMaxBytes)
	}
	width, height, err := webpDimensions(data)
	if err != nil {
		t.Fatalf("%s: %v", heroPath, err)
	}
	if width != heroWidth || height != heroHeight {
		t.Errorf("%s is %dx%d, want %dx%d", heroPath, width, height, heroWidth, heroHeight)
	}
}

// heroRIFF wraps a single chunk into a RIFF/WEBP container with a correct size field.
func heroRIFF(chunk string, payload []byte) []byte {
	body := append([]byte("WEBP"+chunk), make([]byte, 4)...)
	binary.LittleEndian.PutUint32(body[8:12], uint32(len(payload)))
	body = append(body, payload...)
	out := append([]byte("RIFF"), make([]byte, 4)...)
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(body)))
	return append(out, body...)
}

func TestHeroWebPDimensions(t *testing.T) {
	vp8 := []byte{0x10, 0x02, 0x00, 0x9d, 0x01, 0x2a, 0x40, 0x06, 0x84, 0x03}
	// VP8L: 14-bit width-1 = 1599, 14-bit height-1 = 899, alpha and version bits zero.
	vp8lBits := uint32(1599) | uint32(899)<<14
	vp8l := []byte{0x2f, 0, 0, 0, 0}
	binary.LittleEndian.PutUint32(vp8l[1:5], vp8lBits)
	vp8x := []byte{0x10, 0, 0, 0, 0x3f, 0x06, 0x00, 0x83, 0x03, 0x00}

	tests := []struct {
		name          string
		data          []byte
		width, height int
		wantErr       bool
	}{
		{name: "lossy", data: heroRIFF("VP8 ", vp8), width: 1600, height: 900},
		{name: "lossless", data: heroRIFF("VP8L", vp8l), width: 1600, height: 900},
		{name: "extended", data: heroRIFF("VP8X", vp8x), width: 1600, height: 900},
		{name: "png", data: []byte("\x89PNG\r\n\x1a\n0000000000000000"), wantErr: true},
		{name: "too short", data: []byte("RIFF"), wantErr: true},
		{name: "bad riff size", data: append(heroRIFF("VP8 ", vp8), 0), wantErr: true},
		{name: "unknown chunk", data: heroRIFF("ALPH", vp8), wantErr: true},
		{name: "short vp8", data: heroRIFF("VP8 ", vp8[:6]), wantErr: true},
		{name: "vp8 without start code", data: heroRIFF("VP8 ", make([]byte, 10)), wantErr: true},
		{name: "short vp8l", data: heroRIFF("VP8L", vp8l[:3]), wantErr: true},
		{name: "vp8l without signature", data: heroRIFF("VP8L", make([]byte, 5)), wantErr: true},
		{name: "short vp8x", data: heroRIFF("VP8X", vp8x[:6]), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			width, height, err := webpDimensions(tc.data)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want an error, got %dx%d", width, height)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if width != tc.width || height != tc.height {
				t.Errorf("got %dx%d, want %dx%d", width, height, tc.width, tc.height)
			}
		})
	}
}
