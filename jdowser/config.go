// Copyright 2020 Azul Systems, Inc. All rights reserved.
// Use of this source code is governed by the 3-Clause BSD
// license that can be found in the LICENSE file.

package jdowser

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var VERSION = "private build"

type CommandType string

const (
	CMD_START     CommandType = "start"
	CMD_STOP      CommandType = "stop"
	CMD_STATUS    CommandType = "status"
	CMD_REPORT    CommandType = "report"
	CMD_UNDEFINED CommandType = "undefined"
)

type Config struct {
	libjvmFileName string
	nojvmrun       bool
	json           bool
	csv            bool
	skipfs         []string
	root           string
	Command        CommandType
	cookie         string
	wait           bool
	logdir         string
}

func (c *Config) OutputFilePath() string {
	return path.Join(c.logdir, "jdowser.out")
}

func (c *Config) ErrorFilePath() string {
	return path.Join(c.logdir, "jdowser.err")
}

func (c *Config) StatusFilePath() string {
	return path.Join(c.logdir, "jdowser.status")
}

type GivenFlags struct {
	OutJson  bool
	OutCsv   bool
	Root     string
	SkipFs   string
	NoJVMRun bool
	Wait     bool
	Version  bool
}

func InitConfig(givenFlags GivenFlags) *Config {
	config := Config{
		Command: CMD_UNDEFINED,
	}
	config.libjvmFileName = getLibJVMFileName()

	if givenFlags.Version {
		if givenFlags.OutJson {
			type Version struct {
				Version string `json:"version"`
			}
			txt, _ := json.Marshal(Version{VERSION})
			fmt.Println(string(txt))
		} else if givenFlags.OutCsv {
			w := csv.NewWriter(os.Stdout)
			w.Write([]string{"version", VERSION})
			w.Flush()
		} else {
			fmt.Println(filepath.Base(os.Args[0]), "version:", VERSION)
		}
		os.Exit(0)
	}

	if flag.NArg() != 1 {
		flag.Usage()
		return nil
	}

	if flag.Arg(0) != "" {
		config.Command = CommandType(flag.Arg(0))
	}

	allowedChars := regexp.MustCompile(`^[a-z,]+$`).MatchString
	if givenFlags.SkipFs != "" && !allowedChars(givenFlags.SkipFs) {
		fmt.Println("Error: bad -skipfs parameter:", givenFlags.SkipFs)
		os.Exit(1)
	}

	for _, fs := range strings.Split(givenFlags.SkipFs, ",") {
		if fs != "" {
			config.skipfs = append(config.skipfs, fs)
		}
	}

	config.nojvmrun = givenFlags.NoJVMRun
	config.json = givenFlags.OutJson
	config.csv = givenFlags.OutCsv
	config.root = givenFlags.Root
	config.wait = givenFlags.Wait

	u, err := user.Current()
	checkError(err)

	h, err := os.Hostname()
	checkError(err)

	config.cookie = fmt.Sprintf("SCANJVM_COOKIE=%s", u.Username)

	cacheDir, err := os.UserCacheDir()
	checkError(err)

	config.logdir = path.Join(cacheDir, "jdowser", h, u.Username)
	err = os.MkdirAll(config.logdir, 0700)
	checkError(err)

	return &config
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		os.Exit(1)
	}
}

func getLibJVMFileName() string {
	switch runtime.GOOS {
	case "darwin":
		return "libjvm.dylib"
	case "linux":
		return "libjvm.so"
	default:
		return "libjvm.so"
	}
}
