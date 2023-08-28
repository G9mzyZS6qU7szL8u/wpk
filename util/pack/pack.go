package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

// command line settings
var (
	srcpath string
	SrcList []string
	DstFile string
	PutMIME bool
	ShowLog bool
	Split   bool
)

func parseargs() {
	flag.StringVar(&srcpath, "src", "", "full path to folder with source files to be packaged, or list of folders divided by ';'")
	flag.StringVar(&DstFile, "dst", "", "full path to output package file")
	flag.BoolVar(&PutMIME, "mime", false, "put content MIME type defined by file extension")
	flag.BoolVar(&ShowLog, "sl", true, "show process log for each extracting file")
	flag.BoolVar(&Split, "split", false, "write package to splitted files")
	flag.Parse()
}

func checkargs() int {
	var ec = 0 // error counter

	for i, fpath := range strings.Split(srcpath, ";") {
		if fpath == "" {
			continue
		}
		fpath = wpk.ToSlash(wpk.Envfmt(fpath))
		if !strings.HasSuffix(fpath, "/") {
			fpath += "/"
		}
		if ok, _ := wpk.PathExists(fpath); !ok {
			log.Printf("source path #%d '%s' does not exist", i+1, fpath)
			ec++
			continue
		}
		SrcList = append(SrcList, fpath)
	}
	if len(SrcList) == 0 {
		log.Println("source path does not specified")
		ec++
	}

	DstFile = wpk.ToSlash(wpk.Envfmt(DstFile))
	if DstFile == "" {
		log.Println("destination file does not specified")
		ec++
	} else if ok, _ := wpk.PathExists(path.Dir(DstFile)); !ok {
		log.Println("destination path does not exist")
		ec++
	}

	return ec
}

var num, sum int64

func packdirclosure(pkg *wpk.Package, r io.ReadSeeker, ts wpk.TagsetRaw) (err error) {
	var size = ts.Size()
	var fpath = ts.Path()
	num++
	sum += size
	if ShowLog {
		log.Printf("#%-4d %7d bytes   %s", num, size, fpath)
	}

	// adjust tags
	if PutMIME {
		const sniffLen = 512
		var ctype = mime.TypeByExtension(path.Ext(fpath))
		if ctype == "" {
			// rewind to file start
			if _, err = r.Seek(0, io.SeekStart); err != nil {
				return err
			}
			// read a chunk to decide between utf-8 text and binary
			var buf [sniffLen]byte
			var n int64
			if n, err = io.CopyN(bytes.NewBuffer(buf[:]), r, sniffLen); err != nil && err != io.EOF {
				return err
			}
			ctype = http.DetectContentType(buf[:n])
		}
		if ctype != "" {
			pkg.SetupTagset(ts.Put(wpk.TIDmime, wpk.StrTag(ctype)))
		}
	}
	return nil
}

func writepackage() (err error) {
	var fwpk, fwpf wpk.WriteSeekCloser
	var pkgfile, datfile = DstFile, DstFile
	var pkg = wpk.NewPackage()
	if Split {
		pkgfile, datfile = wpk.MakeTagsPath(pkgfile), wpk.MakeDataPath(datfile)
	}

	// open package file to write
	if fwpk, err = os.OpenFile(pkgfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		return
	}
	defer fwpk.Close()

	if Split {
		if fwpf, err = os.OpenFile(datfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
			return
		}
		defer fwpf.Close()

		log.Printf("destination tags part:  %s\n", pkgfile)
		log.Printf("destination files part: %s\n", datfile)
	} else {
		log.Printf("destination file: %s\n", pkgfile)
	}

	// starts new package
	if err = pkg.Begin(fwpk, fwpf); err != nil {
		return
	}

	// data writer
	var w = fwpk
	if fwpf != nil {
		w = fwpf
	}

	// write all source folders
	for i, fpath := range SrcList {
		log.Printf("source folder #%d: %s", i+1, fpath)
		num, sum = 0, 0
		if err = pkg.PackDir(w, fpath, "", packdirclosure); err != nil {
			return
		}
		log.Printf("packed: %d files on %d bytes", num, sum)
	}

	// finalize
	log.Printf("write tags table")
	if err = pkg.Sync(fwpk, fwpf); err != nil {
		return
	}

	return
}

func main() {
	parseargs()
	if checkargs() > 0 {
		return
	}

	log.Println("starts")
	if err := writepackage(); err != nil {
		log.Println(err.Error())
		return
	}
	log.Println("done.")
}

// The End.
