# steg
A basic [steganography](https://en.wikipedia.org/wiki/Steganography) library/tool. Mostly just for practice with Go.

It's currently a work-in-progress, and as such contains **known bugs** (as well as at least a few unknown ones).

## Library Usage
To include it in a project, simply use:
```go
import "github.com/zedseven/steg"
```

Then in code, simply use the `steg.Hide()` and `steg.Dig()` methods.

## Executable Usage

To build and use the executable (from the project base directory):

```bash
go install ./cmd/steg
```

### Using the installed executable

Hiding data in images:

```bash
steg hide -img="<path to host image>" -file="<path to file to hide>" -pattern="<path to unique file>" -out="<path to output file to>"
```

Extracting data from images:

```bash
steg dig -img="<path to host image>" -pattern="<path to unique file (same as used when hiding)>" -out="<path to output file to>"
```
