package wpk

import (
	"io"
	"io/fs"
	"path"
	"strconv"
	"strings"
)

// Void is empty structure to release of the set of keys.
type Void = struct{}

// UnionDir is helper to get access to union directories,
// that contains files from all packages with same dir if it present.
type UnionDir struct {
	*TagsetRaw
	*Union
}

// fs.ReadDirFile interface implementation.
func (f *UnionDir) Stat() (fs.FileInfo, error) {
	return f, nil
}

// fs.ReadDirFile interface implementation.
func (f *UnionDir) Read([]byte) (int, error) {
	return 0, io.EOF
}

// fs.ReadDirFile interface implementation.
func (f *UnionDir) Close() error {
	return nil
}

// fs.ReadDirFile interface implementation.
func (f *UnionDir) ReadDir(n int) ([]fs.DirEntry, error) {
	var dir = f.Path()
	if f.workspace != "." && f.workspace != "" {
		if !strings.HasPrefix(dir, f.workspace) {
			return nil, ErrOtherSubdir
		}
		dir = strings.TrimPrefix(dir, f.workspace)
		if len(dir) > 0 {
			if dir[0] != '/' {
				return nil, ErrOtherSubdir
			}
			dir = dir[1:]
		}
	}
	return f.ReadDirN(dir, n)
}

type Union struct {
	List      []Packager
	workspace string // workspace directory in package
}

// Close call Close-function for all included into the union packages.
// io.Closer implementation.
func (u *Union) Close() (err error) {
	for _, pack := range u.List {
		if err1 := pack.Close(); err1 != nil {
			err = err1
		}
	}
	return
}

// AllKeys returns list of all accessible files in union of packages.
// If union have more than one file with the same name, only first
// entry will be included to result.
func (u *Union) AllKeys() (res []string) {
	var found = map[string]Void{}
	for _, pack := range u.List {
		pack.Enum(func(fkey string, ts *TagsetRaw) bool {
			if _, ok := found[fkey]; !ok {
				res = append(res, fkey)
				found[fkey] = Void{}
			}
			return true
		})
	}
	return
}

// Glob returns the names of all files in union matching pattern or nil
// if there is no matching file.
func (u *Union) Glob(pattern string) (res []string, err error) {
	pattern = path.Join(u.workspace, Normalize(pattern))
	if _, err = path.Match(pattern, ""); err != nil {
		return
	}
	var found = map[string]Void{}
	for _, pack := range u.List {
		pack.Enum(func(fkey string, ts *TagsetRaw) bool {
			if _, ok := found[fkey]; !ok {
				if matched, _ := path.Match(pattern, fkey); matched {
					res = append(res, fkey)
				}
				found[fkey] = Void{}
			}
			return true
		})
	}
	return
}

// Sub clones object and gives access to pointed subdirectory.
// fs.SubFS implementation.
func (u *Union) Sub(dir string) (fs.FS, error) {
	var u1 Union
	u1.workspace = path.Join(u.workspace, dir)
	for _, pack := range u.List {
		if sub1, err1 := pack.Sub(dir); err1 == nil {
			u1.List = append(u1.List, sub1.(Packager))
		}
	}
	if len(u1.List) == 0 {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: fs.ErrNotExist}
	}
	return &u1, nil
}

// Stat returns a fs.FileInfo describing the file.
// If union have more than one file with the same name, info of the first will be returned.
// fs.StatFS implementation.
func (u *Union) Stat(name string) (fs.FileInfo, error) {
	var fullname = path.Join(u.workspace, name)
	var ts *TagsetRaw
	var is bool
	for _, pack := range u.List {
		if ts, is = pack.Tagset(fullname); is {
			return ts, nil
		}
	}
	return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}

// ReadFile returns slice with nested into union of packages file content.
// If union have more than one file with the same name, first will be returned.
// fs.ReadFileFS implementation.
func (u *Union) ReadFile(name string) ([]byte, error) {
	var fullname = path.Join(u.workspace, name)
	var ts *TagsetRaw
	var is bool
	for _, pack := range u.List {
		if ts, is = pack.Tagset(fullname); is {
			var f, err = pack.OpenTagset(ts)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			var size = ts.Size()
			var buf = make([]byte, size)
			_, err = f.Read(buf)
			return buf, err
		}
	}
	return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
func (u *Union) ReadDirN(dir string, n int) (list []fs.DirEntry, err error) {
	var fullname = path.Join(u.workspace, dir)
	var prefix string
	if fullname != "." {
		prefix = Normalize(fullname) + "/" // set terminated slash
	}
	var found = map[string]Void{}
	for _, pack := range u.List {
		pack.Enum(func(fkey string, ts *TagsetRaw) bool {
			if strings.HasPrefix(fkey, prefix) {
				var suffix = fkey[len(prefix):]
				var sp = strings.IndexByte(suffix, '/')
				if sp < 0 { // file detected
					if _, ok := found[fkey]; !ok {
						list = append(list, ts)
						found[fkey] = Void{}
						n--
					}
				} else { // dir detected
					var subdir = path.Join(prefix, suffix[:sp])
					if _, ok := found[subdir]; !ok {
						var fpath = ts.Path() // extract not normalized path
						var dts = MakeTagset(nil, 2, 2).
							Put(TIDpath, StrTag(fpath[:len(subdir)]))
						var f = &UnionDir{
							TagsetRaw: dts,
							Union:     u,
						}
						list = append(list, f)
						found[subdir] = Void{}
						n--
					}
				}
			}
			return n != 0
		})
		if n == 0 {
			break
		}
	}
	if n > 0 {
		err = io.EOF
	}
	return
}

// Open implements access to nested into union of packages file or directory by keyname.
// If union have more than one file with the same name, first will be returned.
// fs.FS implementation.
func (u *Union) Open(dir string) (fs.File, error) {
	var fullname = path.Join(u.workspace, dir)
	if strings.HasPrefix(fullname, PackName+"/") {
		var idx, err = strconv.ParseUint(dir[len(PackName)+1:], 10, 32)
		if err != nil {
			return nil, &fs.PathError{Op: "open", Path: dir, Err: err}
		}
		if idx >= uint64(len(u.List)) {
			return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrNotExist}
		}
		return u.List[idx].Open(PackName)
	}

	// try to get the file
	for _, pack := range u.List {
		if ts, is := pack.Tagset(fullname); is {
			return pack.OpenTagset(ts)
		}
	}

	// try to get the folder
	var prefix string
	if fullname != "." {
		prefix = Normalize(fullname) + "/" // set terminated slash
	}
	for _, pack := range u.List {
		var f *UnionDir
		pack.Enum(func(fkey string, ts *TagsetRaw) bool {
			if strings.HasPrefix(fkey, prefix) {
				var dts = MakeTagset(nil, 2, 2).
					Put(TIDpath, StrTag(fullname))
				f = &UnionDir{
					TagsetRaw: dts,
					Union:     u,
				}
				return false
			}
			return true
		})
		if f != nil {
			return f, nil
		}
	}
	// on case if not found
	return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrNotExist}
}

// The End.
