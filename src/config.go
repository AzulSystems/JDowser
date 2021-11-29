// Copyright 2020 Azul Systems, Inc. All rights reserved.
// Use of this source code is governed by the 3-Clause BSD
// license that can be found in the LICENSE file.

package main

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

type CommandType string

const (
	CMD_START  CommandType = "start"
	CMD_STOP   CommandType = "stop"
	CMD_STATUS CommandType = "status"
	CMD_REPORT CommandType = "report"
)

type Config struct {
	libjvmFileName string
	nojvmrun       bool
	json           bool
	csv            bool
	skipfs         []string
	root           string
	command        CommandType
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

func InitConfig() *Config {
	config := Config{}
	config.libjvmFileName = getLibJVMFileName()

	outjson := flag.Bool("json", false, "dump output in JSON format")
	outcsv := flag.Bool("csv", false, "dump output in CSV format")
	root := flag.String("root", "/", "root scan directory")
	skipfs := flag.String("skipfs", "nfs,tmp,proc", "list of filesystem types to skip.")
	nojvmrun := flag.Bool("nojvmrun", false, "do not run java -version to detect version")
	wait := flag.Bool("wait", false, "wait completion of scan process")
	version := flag.Bool("version", false, "show version and exit")

	flag.Usage = func() {
		name := filepath.Base(os.Args[0])
		fmt.Println(name, "- Utility to find JVMs/JDKs and report their versions")
		fmt.Println("Version:", VERSION)
		fmt.Println()
		fmt.Printf("Usage: %s [-json|-csv] [-skipfs=fstype[,fstype..]] [-nojvmrun] [-wait] [-root=<scanroot>] %s\n", name, CMD_START)
		fmt.Printf("       %s [-json|-csv] [-wait] %s\n", name, CMD_STATUS)
		fmt.Printf("       %s [-json|-csv] [-wait] %s\n", name, CMD_REPORT)
		fmt.Printf("       %s [-json|-csv] %s\n", name, CMD_STOP)
		fmt.Printf("       %s [-json|-csv] -version\n", name)
		flag.PrintDefaults()
	}

	flag.Parse()

	if *version {
		if *outjson {
			type Version struct {
				Version string `json:"version"`
			}
			txt, _ := json.Marshal(Version{VERSION})
			fmt.Println(string(txt))
		} else if *outcsv {
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

	config.command = CommandType(flag.Arg(0))

	allowedChars := regexp.MustCompile(`^[a-z,]+$`).MatchString
	if *skipfs != "" && !allowedChars(*skipfs) {
		fmt.Println("Error: bad -skipfs parameter:", *skipfs)
		os.Exit(1)
	}

	for _, fs := range strings.Split(*skipfs, ",") {
		if fs != "" {
			config.skipfs = append(config.skipfs, fs)
		}
	}

	config.nojvmrun = *nojvmrun
	config.json = *outjson
	config.csv = *outcsv
	config.root = *root
	config.wait = *wait

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

