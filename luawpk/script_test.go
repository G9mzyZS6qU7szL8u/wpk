package luawpk_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/schwarzlichtbezirk/wpk"
	lw "github.com/schwarzlichtbezirk/wpk/luawpk"
)

var scrdir = wpk.Envfmt("${GOPATH}/src/github.com/schwarzlichtbezirk/wpk/test/")
var mediadir = scrdir + "media/"

// Test package content on nested and external files equivalent.
func CheckPackage(t *testing.T, wpkname string) {
	var err error
	var pack wpk.Writer
	var fwpk *os.File

	// open temporary file for read/write
	if fwpk, err = os.Open(wpkname); err != nil {
		t.Fatal(err)
	}
	defer fwpk.Close()

	if err = pack.Read(fwpk); err != nil {
		t.Fatal(err)
	}

	for _, tags := range pack.Tags {
		var kpath = tags.Path()
		if _, is := tags[wpk.TIDcreated]; !is {
			t.Logf("found packed data #%d '%s'", tags.FID(), kpath)
			continue // skip packed data
		}

		var link, is = tags[wpk.TIDlink]
		if !is {
			t.Fatalf("found file without link #%d '%s'", tags.FID(), kpath)
		}
		var offset, size = tags.Offset(), tags.Size()

		var orig []byte
		if orig, err = os.ReadFile(mediadir + string(link)); err != nil {
			t.Fatal(err)
		}

		if size != int64(len(orig)) {
			t.Errorf("size of file '%s' (%d) in package is defer from original (%d)",
				kpath, size, len(orig))
		}

		var extr = make([]byte, size)
		var n int
		if n, err = fwpk.ReadAt(extr, offset); err != nil {
			t.Fatal(err)
		}
		if n != len(extr) {
			t.Errorf("can not extract content of file '%s' completely", kpath)
		}
		if !bytes.Equal(orig, extr) {
			t.Errorf("content of file '%s' is defer from original", kpath)
		}

		if t.Failed() {
			break
		}

		t.Logf("checkup #%d '%s' is ok", tags.FID(), kpath)
	}
}

// Test packdir script call.
func TestPackdir(t *testing.T) {
	var wpkname = filepath.Join(os.TempDir(), "packdir.wpk")
	defer os.Remove(wpkname)

	if err := lw.RunLuaVM(scrdir + "packdir.lua"); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, wpkname)
}

// Test append package ability by scripts.
func TestSteps(t *testing.T) {
	var wpkname = filepath.Join(os.TempDir(), "steps.wpk")
	defer os.Remove(wpkname)

	if err := lw.RunLuaVM(scrdir + "step1.lua"); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, wpkname)

	if err := lw.RunLuaVM(scrdir + "step2.lua"); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, wpkname)
}

// The End.
