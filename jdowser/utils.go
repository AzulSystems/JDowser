// Copyright 2020 Azul Systems, Inc. All rights reserved.
// Use of this source code is governed by the 3-Clause BSD
// license that can be found in the LICENSE file.

package jdowser

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
)

type JSONArrayWriter struct {
	out   io.Writer
	first bool
}

func NewJSONArrayWriter(f io.Writer) *JSONArrayWriter {
	return &JSONArrayWriter{f, true}
}

func (w *JSONArrayWriter) Write(data []byte) (int, error) {
	var s string
	if w.first {
		w.first = false
		s = "[\n  "
	} else {
		s = ",\n  "
	}

	n1, e := w.out.Write([]byte(s))
	if e != nil {
		return n1, e
	}
	if data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}
	n2, e := w.out.Write(data)
	return n2 + n1, e
}

func (w *JSONArrayWriter) Close() {
	if !w.first {
		_, _ = w.out.Write([]byte("\n]\n"))
	}
}

func findFiles(config *Config, callback func(fname string)) error {
	command := []string{config.Root}
	l := len(config.SkipFs)
	fs := &config.SkipFs
	if l > 0 {
		command = append(command, "(")
		for i := 0; i < l-1; i++ {
			command = append(command, "-fstype", (*fs)[i], "-o")
		}
		command = append(command, "-fstype", (*fs)[l-1], ")", "-prune", "-o")
	}
	command = append(command, "-xdev", "-type", "f", "-name", config.LibJVMFileName, "-print")

	cmd := exec.Command("find", command...)

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()
	e := cmd.Start()
	if e != nil {
		return e
	}

	outputScanner := bufio.NewScanner(bufio.NewReader(stdoutIn))
	outputScanner.Split(bufio.ScanLines)
	errorScanner := bufio.NewScanner(bufio.NewReader(stderrIn))
	errorScanner.Split(bufio.ScanLines)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for outputScanner.Scan() {
			callback(outputScanner.Text())
		}
		wg.Done()
	}()

	for errorScanner.Scan() {
		// Ignore find errors for now
		// println("ERROR", errorScanner.Text());
	}
	wg.Wait()
	_ = cmd.Wait()
	return nil
}

func closeFile(file *os.File) {
	_ = file.Close()
}

func fileIsEmpty(f *os.File) bool {
	fileInfo, e := f.Stat()
	return e != nil || fileInfo.Size() == 0
}

func findInUseLibJVM() map[string]int {
	res := make(map[string]int)

	procDir, e := os.ReadDir("/proc")
	if e != nil {
		return res
	}

	for _, entry := range procDir {
		if entry.IsDir() {
			if mapsFile, e := os.Open(path.Join("/proc", entry.Name(), "maps")); e == nil {
				scan := bufio.NewScanner(mapsFile)
				for scan.Scan() {
					str := scan.Text()
					if strings.Contains(str, "libjvm.so") {
						if idx := strings.IndexByte(str, '/'); idx > 0 {
							_, e := os.Stat(str[idx:])
							if e == nil {
								res[str[idx:]]++
							}
						}
						break
					}
				}
			}
		}
	}

	return res
}
