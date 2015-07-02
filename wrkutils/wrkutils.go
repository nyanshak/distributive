package wrkutils

import (
	"bytes"
	"fmt"
	"github.com/CiscoCloud/distributive/tabular"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// InitializeLogrus simply sets the log level appropriately
func InitializeLogrus(lvl log.Level) {
	log.SetLevel(lvl)
}

// Worker is the type of function that takes a list of string params and returns
// an error code and an exit message to be printed to stdout.
// Generally, if exitCode == 0, exitMessage == "".
type Worker func(parameters []string) (exitCode int, exitMessage string)

//// STRING UTILITIES

// CommandOutput returns a string version of the ouput of a given command,
// and reports errors effectively.
func CommandOutput(cmd *exec.Cmd) string {
	out, err := cmd.CombinedOutput()
	outStr := string(out)
	if err != nil {
		ExecError(cmd, outStr, err)
	}
	return outStr
}

// CommandColumnNoHeader returns a specified column of the output of a command,
// without that column's header. Useful for parsing the output of shell commands,
// which many of the Checks require.
// TODO for some reason, this + route -n doesn't work with probabalistic.
func CommandColumnNoHeader(col int, cmd *exec.Cmd) []string {
	out := tabular.ProbabalisticSplit(CommandOutput(cmd))
	return tabular.GetColumnNoHeader(col, out)
}

// GetByteUnits returns: b | kb | mb | gb | tb, from a string containing
// some form of any of the above. It is for normalization.
// NOTE: this doesn't differentiate between kb and kib, and I don't know how
// `free` does.
func GetByteUnits(str string) string {
	regexps := map[string]*regexp.Regexp{
		"b":  regexp.MustCompile("^bytes{0,1}|^[bB]{1}"),
		"kb": regexp.MustCompile("kilo(bytes){0,1}|[kK]{1}[iI]{0,1}[bB]{1}"),
		"mb": regexp.MustCompile("mega(bytes){0,1}|[mM]{1}[iI]{0,1}[bB]{1}"),
		"gb": regexp.MustCompile("giga(bytes){0,1}|[gG]{1}[iI]{0,1}[bB]{1}"),
		"tb": regexp.MustCompile("terra(bytes){0,1}|[tT]{1}[iI]{0,1}[bB]{1}"),
	}
	for unit, re := range regexps {
		if re.MatchString(str) {
			return unit
		}
	}
	// warn the user that the string couldn't be matched
	units := []string{}
	regexpStrings := []string{}
	for unit, re := range regexps {
		units = append(units, unit)
		regexpStrings = append(regexpStrings, re.String())
	}
	log.WithFields(log.Fields{
		"string":  str,
		"seeking": units,
		"regexps": regexpStrings,
	}).Warn("Couldn't extract byte units from string")
	return ""
}

//// ERROR UTILITIES

// PathError is an abstraction of CouldntReadError and CouldntWriteError
func PathError(path string, err error, read bool) {
	// is it a read or write error?
	readOrWrite := "write"
	if read {
		readOrWrite = "read"
	}

	if err != nil {
		log.WithFields(log.Fields{
			"path":  path,
			"error": err.Error(),
		}).Fatal("Couldn't " + readOrWrite + " file")
	}
}

// CouldntWriteError logs.Fatal an error relating to writing a file
func CouldntWriteError(path string, err error) {
	PathError(path, err, false)
}

// CouldntReadError logs.Fatal an error related to reading a file
func CouldntReadError(path string, err error) {
	PathError(path, err, true)
}

// GenericError is a general error where the requested variable was not found in
// a given list of variables. This is pure DRY.
func GenericError(msg string, name string, actual []string) (exitCode int, exitMessage string) {
	// get a list of strings, append them, truncate them
	actualStrSlc := []string{}
	for _, val := range actual {
		actualStrSlc = append(actualStrSlc, fmt.Sprint(val))
	}
	actualStr := strings.Join(actualStrSlc, ", ")
	msg += ":\n\tSpecified: " + name
	msg += "\n\tActual: " + actualStr
	return 1, msg
}

// ExecError logs.Fatal with a useful message
func ExecError(cmd *exec.Cmd, out string, err error) {
	msg := "Failed to execute command"
	if strings.Contains(out, "permission denied") {
		msg = "Permission denied when running command"
	}
	if err != nil && strings.Contains(err.Error(), "not found in $PATH") {
		msg = "Couldn't find executable when running command"
	}
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd.Args,
			"path":    cmd.Path,
			"output":  out,
			"error":   err.Error(),
		}).Fatal(msg)
	}
}

// IO UTILITIES

// ParseUserRegex either returns a regex from a string, or displays an
// appropriate error to the user
func ParseUserRegex(regexString string) *regexp.Regexp {
	re, err := regexp.Compile(regexString)
	if err != nil {
		log.WithFields(log.Fields{
			"regexp": regexString,
			"error":  err.Error(),
		}).Fatal("Bad configuration - couldn't parse golang regexp")
	}
	return re
}

// FileToBytes reads a file and handles the error
func FileToBytes(path string) []byte {
	data, err := ioutil.ReadFile(path)
	CouldntReadError(path, err)
	return data
}

// FileToString reads in a file at a path, handles errors, and returns that file
// as a string
func FileToString(path string) string {
	return string(FileToBytes(path))
}

// FileToLines reads in a file at a path, handles errors, splits it into lines,
// and returns those lines as byte slices
func FileToLines(path string) [][]byte {
	return bytes.Split(FileToBytes(path), []byte("\n"))
}

// ParseMyInt parses an int or logs the error
func ParseMyInt(str string) int {
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		log.WithFields(log.Fields{
			"given string": str,
			"error":        err.Error(),
		}).Fatal("Probable configuration error - could not parse integer")
	}
	return int(i)
}

// GetFilesWithExtension returns the paths to all the files in the given dir
// that end with the given file extension (with or without dot)
func GetFilesWithExtension(path string, ext string) (paths []string) {
	finfos, err := ioutil.ReadDir(path) // list of os.FileInfo
	if err != nil {
		CouldntReadError(path, err)
	}
	for _, finfo := range finfos {
		name := finfo.Name()
		if strings.HasSuffix(name, ext) {
			// TODO path.Join these suckers
			paths = append(paths, path+"/"+name)
		}
	}
	return paths
}