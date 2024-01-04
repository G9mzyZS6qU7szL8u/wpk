# WPK

Library to build and use data files packages.

[![Go](https://github.com/schwarzlichtbezirk/wpk/actions/workflows/go.yml/badge.svg)](https://github.com/schwarzlichtbezirk/wpk/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/schwarzlichtbezirk/wpk.svg)](https://pkg.go.dev/github.com/schwarzlichtbezirk/wpk)
[![Go Report Card](https://goreportcard.com/badge/github.com/schwarzlichtbezirk/wpk)](https://goreportcard.com/report/github.com/schwarzlichtbezirk/wpk)
[![Hits-of-Code](https://hitsofcode.com/github/schwarzlichtbezirk/wpk?branch=master)](https://hitsofcode.com/github/schwarzlichtbezirk/wpk/view?branch=master)

and later append to exising package new files at *step2* call:

```batch
go run github.com/schwarzlichtbezirk/wpk/util/build ${GOPATH}/src/github.com/schwarzlichtbezirk/wpk/testdata/step2.lua
```

[packdir.lua](https://github.com/schwarzlichtbezirk/wpk/blob/master/testdata/packdir.lua) script has function that can be used to put to package directory with original tree hierarchy.

## WPK API usage

See [godoc](https://pkg.go.dev/github.com/schwarzlichtbezirk/wpk) with API description, and [wpk_test.go](https://github.com/schwarzlichtbezirk/wpk/blob/master/wpk_test.go) for usage samples.

On your program initialisation open prepared wpk-package by [Package.OpenFile](https://pkg.go.dev/github.com/schwarzlichtbezirk/wpk#Package.OpenFile) call. It reads tags sets of package at once, then you can get access to filenames and it's tags. [TagsetRaw](https://pkg.go.dev/github.com/schwarzlichtbezirk/wpk#TagsetRaw) structure helps you to get tags associated to files, and also it provides file information by standard interfaces implementation. To get access to package nested files, create some [Tagger](https://pkg.go.dev/github.com/schwarzlichtbezirk/wpk#Tagger) object. Modules `wpk/bulk`, `wpk/mmap` and `wpk/fsys` provides this access by different ways. `Package` object have all `io/fs` file system interfaces implementations, and can be used by anyway where they needed.
