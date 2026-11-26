package services

import (
	"bytes"
	"os"
	"testing"
)

var png1x1 = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

func TestCheckImage(t *testing.T) {
	if m, err := checkImage(png1x1, 1000); err != nil || m != "image/png" {
		t.Fatalf("png: %q %v", m, err)
	}
	if _, err := checkImage(png1x1, 10); err == nil {
		t.Fatal("over the size limit must fail")
	}
	if _, err := checkImage([]byte("<html><script>alert(1)</script></html>"), 1000); err == nil {
		t.Fatal("html pretending to be an image must fail")
	}
}

func TestDiskStoreStaysInsideItsDirectory(t *testing.T) {
	d, err := NewDiskStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Put("../escape.png", png1x1); err == nil {
		t.Fatal("a path outside the directory must be refused")
	}
	if err := d.Put("a.png", png1x1); err != nil {
		t.Fatal(err)
	}
	f, err := d.Open("a.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, _ := os.ReadFile(f.Name())
	if !bytes.Equal(got, png1x1) {
		t.Fatal("file contents changed")
	}
	d.Remove("a.png")
	if _, err := d.Open("a.png"); err == nil {
		t.Fatal("file should be gone")
	}
}
