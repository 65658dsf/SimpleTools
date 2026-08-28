package tools

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

func TestApplyOrientationRotate90(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	src.Set(1, 0, color.RGBA{B: 255, A: 255})
	dst := ApplyOrientation(src, 6)
	if dst.Bounds().Dx() != 1 || dst.Bounds().Dy() != 2 {
		t.Fatalf("bounds=%v", dst.Bounds())
	}
	r, g, b, a := dst.At(0, 0).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("top pixel=%v", dst.At(0, 0))
	}
}

func TestJPEGOrientationAbsent(t *testing.T) {
	if got := JPEGOrientation([]byte{0xff, 0xd8, 0xff, 0xd9}); got != 1 {
		t.Fatalf("got %d", got)
	}
}

func TestJPEGOrientationLittleEndian(t *testing.T) {
	data := []byte{0xff, 0xd8, 0xff, 0xe1, 0, 26, 'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 0x01, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
	binary.BigEndian.PutUint16(data[4:6], uint16(len(data)-4))
	if got := JPEGOrientation(data); got != 6 {
		t.Fatalf("got %d", got)
	}
}
