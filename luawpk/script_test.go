package luawpk_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/schwarzlichtbezirk/wpk"
	lw "github.com/schwarzlichtbezirk/wpk/luawpk"
)

const scrdir = "../testdata/"
const mediadir = scrdir + "media/"

// Test package content on nested and external files equivalent.
func CheckPackage(t *testing.T, wptname, wpfname string) {
	var err error

	// Open package files tags table
	var pkg *wpk.Package
	if pkg, err = wpk.OpenPackage(wptname); err != nil {
		t.Fatal(err)
	}

	// open temporary file for read/write
	var fwpf *os.File
	if wpfname != "" && wptname != wpfname {
		if fwpf, err = os.Open(wpfname); err != nil {
			t.Fatal(err)
		}
	} else {
		if fwpf, err = os.Open(wptname); err != nil {
			t.Fatal(err)
		}
	}
	defer fwpf.Close()

	if ts, ok := pkg.GetInfo(); ok {
		var offset, size = ts.Pos()
		var label, _ = ts.TagStr(wpk.TIDlabel)
		t.Logf("package info: offset %d, size %d, label '%s'", offset, size, label)
	}
	var n = 0
	pkg.Enum(func(fkey string, ts wpk.TagsetRaw) bool {
		var ok bool
		var offset, size = ts.Pos()
		n++

		if ok = ts.Has(wpk.TIDmtime); !ok {
			t.Logf("found packed data #%d '%s'", n, fkey)
			return true // skip packed data
		}

		var link wpk.TagRaw
		if link, ok = ts.Get(wpk.TIDlink); !ok {
			t.Fatalf("found file without link #%d '%s'", n, fkey)
		}

		var orig []byte
		if orig, err = os.ReadFile(mediadir + wpk.B2S(link)); err != nil {
			t.Fatal(err)
		}

		if size != wpk.Uint(len(orig)) {
			t.Errorf("size of file '%s' (%d) in package is defer from original (%d)",
				fkey, size, len(orig))
		}

		var extr = make([]byte, size)
		var readed int
		if readed, err = fwpf.ReadAt(extr, int64(offset)); err != nil {
			t.Fatal(err)
		}
		if readed != len(extr) {
			t.Errorf("can not extract content of file '%s' completely", fkey)
		}
		if !bytes.Equal(orig, extr) {
			t.Errorf("content of file '%s' is defer from original", fkey)
		}

		if t.Failed() {
			return false
		}

		t.Logf("checkup #%d '%s' is ok", n, fkey)
		return true
	})
}

// Test packdir script call.
func TestPackdir(t *testing.T) {
	var wpkname = wpk.TempPath("packdir.wpk")
	defer os.Remove(wpkname)

	if err := lw.RunLuaVM(scrdir + "packdir.lua"); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, wpkname, "")
}

// Test append package ability by scripts.
func TestSteps(t *testing.T) {
	var wpkname = wpk.TempPath("steps.wpk")
	defer os.Remove(wpkname)

	if err := lw.RunLuaVM(scrdir + "step1.lua"); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, wpkname, "")

	if err := lw.RunLuaVM(scrdir + "step2.lua"); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, wpkname, "")
}

// Test splitted package forming.
func TestSplitted(t *testing.T) {
	var wptname = wpk.TempPath("build.wpt")
	var wpfname = wpk.TempPath("build.wpf")
	defer os.Remove(wptname)
	defer os.Remove(wpfname)

	if err := lw.RunLuaVM(scrdir + "split.lua"); err != nil {
		t.Fatal(err)
	}

	// make package file check up
	CheckPackage(t, wptname, wpfname)
}

// The End.
