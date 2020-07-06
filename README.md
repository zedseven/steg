# steg
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) [![GoDoc](https://godoc.org/github.com/zedseven/steg?status.svg)](https://godoc.org/github.com/zedseven/steg)

A full-featured [steganography](https://en.wikipedia.org/wiki/Steganography) toolkit.

It's currently a work-in-progress, and it's operation is still subject to change.

## Using it as a package
To include it in a project, simply use:
```go
import "github.com/zedseven/steg"
```

Then in code, simply use the `steg.Hide()` and `steg.Dig()` methods. See [the GoDoc manual](https://godoc.org/github.com/zedseven/steg) for documentation.

## Using it as a standalone tool

To build and use the executable (from the project base directory):

```bash
go install ./cmd/steg
```

### Running the installed executable

Hiding data in images:

```bash
steg hide -img="<path to host image>" -file="<path to file to hide>" -pattern="<path to unique file>" -out="<path to output file to>"
```

Extracting data from images:

```bash
steg dig -img="<path to host image>" -pattern="<path to unique file (same as used when hiding)>" -out="<path to output file to>"
```
