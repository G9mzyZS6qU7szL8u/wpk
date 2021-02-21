package bulk

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

// PackDir is wrapper for package to hold WPK-file whole content as a slice.
// Gives access to directory in package with prefix "pref".
// http.FileSystem interface implementation.
type PackDir struct {
	*wpk.Package
	bulk []byte
	pref string
}

// NamedTags returns tags set referred by offset at TAT field.
func (pack *PackDir) NamedTags(key string) (wpk.TagSlice, bool) {
	var tagpos, is = pack.TAT[key]
	return pack.bulk[tagpos:], is
}

// OpenWPK opens WPK-file package by given file name.
func (pack *PackDir) OpenWPK(fname string) (err error) {
	var bulk []byte
	if bulk, err = ioutil.ReadFile(fname); err != nil {
		return
	}

	if pack.Package == nil {
		pack.Package = &wpk.Package{}
	}
	pack.bulk = bulk
	pack.pref = ""

	if err = pack.Read(bytes.NewReader(bulk)); err != nil {
		return
	}
	return
}

// Close does nothing, there is no any opened handles.
// Useful for interface compatibility.
func (pack *PackDir) Close() error {
	return nil
}

// SubDir clones object and gives access to pointed subdirectory.
func (pack *PackDir) SubDir(pref string) wpk.Packager {
	pref = wpk.ToKey(pref)
	if len(pref) > 0 && pref[len(pref)-1] != '/' {
		pref += "/"
	}
	return &PackDir{
		pack.Package,
		pack.bulk,
		pack.pref + pref,
	}
}

// OpenFile creates file object to give access to nested into package file by given tagset.
func (pack *PackDir) OpenFile(ts wpk.TagSlice) (http.File, error) {
	var offset, size = ts.Offset(), ts.Size()
	return &wpk.File{
		TagSlice: ts,
		Reader:   *bytes.NewReader(pack.bulk[offset : offset+size]),
		Pack:     pack,
	}, nil
}

// Extract returns slice with nested into package file content.
func (pack *PackDir) Extract(key string) ([]byte, error) {
	var offset, size int64
	if ts, is := pack.NamedTags(key); is {
		offset, size = ts.Offset(), ts.Size()
	} else {
		return nil, &wpk.ErrKey{What: wpk.ErrNotFound, Key: key}
	}
	return pack.bulk[offset : offset+size], nil
}

// Open implements access to nested into package file or directory by keyname.
func (pack *PackDir) Open(kname string) (http.File, error) {
	var kpath = pack.pref + strings.TrimPrefix(kname, "/")
	if kpath == "" {
		return wpk.OpenDir(pack, kpath)
	} else if kpath == "wpk" {
		var buf bytes.Buffer
		var tags = wpk.Tagset{
			wpk.TIDfid:    wpk.TagUint32(0),
			wpk.TIDoffset: wpk.TagUint64(0),
			wpk.TIDsize:   wpk.TagUint64(uint64(len(pack.bulk))),
		}
		tags.WriteTo(&buf)
		return &wpk.File{
			TagSlice: buf.Bytes(),
			Reader:   *bytes.NewReader(pack.bulk),
			Pack:     pack,
		}, nil
	}

	var key = wpk.ToKey(kpath)
	if ts, is := pack.NamedTags(key); is {
		return pack.OpenFile(ts)
	}
	return wpk.OpenDir(pack, kpath)
}

// The End.
