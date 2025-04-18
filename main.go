//usr/bin/go run $0 $@; exit
package main

import (
	"debug/elf"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/docopt/docopt-go"
	"github.com/xyproto/ainur"
)

const (
	versionString = "ELFinfo 1.2.3"
	description   = "Detect the compiler version, given an ELF executable."

	usage = versionString + "\n" + description + `

Usage:
  elfinfo [-l | --long] [-c | --color] <ELF>
  elfinfo -h | --help
  elfinfo --version

Options:
  -c --color       Color the text output (unless NO_COLOR is set).
  -h --help        Show this screen.
  -l --long        Also output stripped status, byte order and target machine.
  --version        Version info.
`
)

// which finds files in the paths in the PATH environment variable.
// If the file exists in $PATH, return the full path.
// If the file exists in the local directory, return that.
// If not, return an empty string.
func which(filename string) (string, error) {
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		return filename, nil
	}
	for _, directory := range strings.Split(os.Getenv("PATH"), ":") {
		fullPath := path.Join(directory, filename)
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			return fullPath, nil
		}
	}
	return "", fmt.Errorf("%s: no such file or directory", filename)
}

// examine tries to detect compiler name and compiler version from a given
// ELF filename.
func examine(filename string, onlyCompilerInfo, noColor bool) {
	f, err := elf.Open(filename)
	if err != nil {
		if strings.Contains(err.Error(), "bad magic number '[") {
			if noColor {
				fmt.Printf("%s: %s\n", filename, "not an ELF")
			} else {
				fmt.Printf("\033[1;33m%s: %s\033[0m\n", filename, "not an ELF")
			}
		} else if strings.Contains(err.Error(), "is a directory") {
			if noColor {
				fmt.Printf("%s: %s\n", filename, "is a directory")
			} else {
				fmt.Printf("\033[1;31m%s: %s\033[0m\n", filename, "is a directory")
			}
		} else {
			if noColor {
				fmt.Printf("%s: %s\n", filename, err)
			} else {
				fmt.Printf("\033[1;31m%s: %s\033[0m\n", filename, err)
			}
		}
		os.Exit(1)
	}
	defer f.Close()

	if onlyCompilerInfo {
		if noColor {
			fmt.Printf("%v\n", ainur.Compiler(f))
		} else {
			fmt.Printf("\033[1;34m%v\033[0m\n", ainur.Compiler(f))
		}
		return
	}

	// Use the short version of LittleEndian and BigEndian
	byteOrder := strings.Replace(strings.Replace(f.ByteOrder.String(), "LittleEndian", "LE", 1), "BigEndian", "BE", 1)

	fmt.Printf("%s: stripped=%v, compiler=%v, static=%v, byteorder=%v, machine=%v\n", filename, ainur.Stripped(f), ainur.Compiler(f), ainur.Static(f), byteOrder, ainur.Describe(f.Machine))
}

func main() {
	arguments, err := docopt.ParseDoc(usage)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if arguments["--version"].(bool) {
		fmt.Println(versionString)
		os.Exit(0)
	}

	filepath, err := which(arguments["<ELF>"].(string))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Respect the NO_COLOR environment variable
	noColor := os.Getenv("NO_COLOR") != ""

	examine(filepath, !arguments["--long"].(bool), noColor || !arguments["--color"].(bool))
}
