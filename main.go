// Copyright 2020 Azul Systems, Inc. All rights reserved.
// Use of this source code is governed by the 3-Clause BSD
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"jdowser/jdowser"
	"os"
	"path/filepath"
)

func main() {

	outjson := flag.Bool("json", false, "dump output in JSON format")
	outcsv := flag.Bool("csv", false, "dump output in CSV format")
	root := flag.String("root", "/", "root scan directory")
	skipfs := flag.String("skipfs", "nfs,tmp,proc", "list of filesystem types to skip.")
	nojvmrun := flag.Bool("nojvmrun", false, "do not run java -version to detect version")
	wait := flag.Bool("wait", false, "wait completion of scan process")
	version := flag.Bool("version", false, "show version and exit")

	givenFlags := jdowser.GivenFlags{
		OutJson:  *outjson,
		OutCsv:   *outcsv,
		Root:     *root,
		SkipFs:   *skipfs,
		NoJVMRun: *nojvmrun,
		Wait:     *wait,
		Version:  *version,
	}

	flag.Usage = func() {
		name := filepath.Base(os.Args[0])
		fmt.Println(name, "- Utility to find JVMs/JDKs and report their versions")
		fmt.Println("Version:", jdowser.VERSION)
		fmt.Println()
		fmt.Printf("Usage: %s [-json|-csv] [-skipfs=fstype[,fstype..]] [-nojvmrun] [-wait] [-root=<scanroot>] %s\n", name, jdowser.CMD_START)
		fmt.Printf("       %s [-json|-csv] [-wait] %s\n", name, jdowser.CMD_STATUS)
		fmt.Printf("       %s [-json|-csv] [-wait] %s\n", name, jdowser.CMD_REPORT)
		fmt.Printf("       %s [-json|-csv] %s\n", name, jdowser.CMD_STOP)
		fmt.Printf("       %s [-json|-csv] -version\n", name)
		flag.PrintDefaults()
	}

	flag.Parse()

	config := jdowser.InitConfig(givenFlags)

	if config != nil {
		switch config.Command {
		case jdowser.CMD_START:
			jdowser.CmdStart(config)
		case jdowser.CMD_STOP:
			jdowser.CmdStop(config)
		case jdowser.CMD_STATUS:
			jdowser.CmdStatus(config)
		case jdowser.CMD_REPORT:
			jdowser.CmdReport(config)

		default:
			fmt.Println("Unknown command:", config.Command)
			os.Exit(1)
		}
	}

}
