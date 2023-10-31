package jdowser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
)

func CmdReport(config *Config) {
	if config.wait {
		lock, e := ScanLock(config)
		if e != nil {
			fmt.Println(e.Error())
			return
		}
		lock.Lock()
		defer lock.Unlock()
	}

	f, e := os.Open(config.OutputFilePath())
	if os.IsNotExist(e) {
		fmt.Printf("Report not found. Run '%s' to generate it first.\n", CMD_START)
		return
	}
	if e != nil {
		fmt.Printf("Cannot open report file: %s\n", e.Error())
		return
	}
	defer closeFile(f)

	if fileIsEmpty(f) {
		if config.json {
			fmt.Println("[]")
		} else {
			fmt.Println("No results found")
		}
		return
	}

	inUseLibJVM := findInUseLibJVM()

	scanner := bufio.NewScanner(f)

	if config.json {
		aw := NewJSONArrayWriter(os.Stdout)
		defer aw.Close()
		enc := json.NewEncoder(aw)
		enc.SetIndent("  ", "  ")
		for scanner.Scan() {
			if info, e := unmarshalInfo(scanner.Bytes(), inUseLibJVM); e == nil {
				_ = enc.Encode(info)
			}
		}
	} else if config.csv {
		DumpCSVHeader(os.Stdout)
		for scanner.Scan() {
			if info, e := unmarshalInfo(scanner.Bytes(), inUseLibJVM); e == nil {
				info.DumpCSV(os.Stdout)
			}
		}
	} else {
		for scanner.Scan() {
			if info, e := unmarshalInfo(scanner.Bytes(), inUseLibJVM); e == nil {
				info.Dump(os.Stdout)
			}
		}
	}
}

func unmarshalInfo(bytes []byte, inUseLibJVM map[string]int) (*JVMInstallation, error) {
	var info JVMInstallation
	e := json.Unmarshal(bytes, &info)
	if e != nil {
		return nil, e
	}
	for running := range inUseLibJVM {
		stat1, e1 := os.Stat(running)
		stat2, e2 := os.Stat(info.LibJVM)
		if e1 == nil && e2 == nil && os.SameFile(stat1, stat2) {
			info.RunningInstances += inUseLibJVM[running]
		}
	}
	return &info, nil
}

func CmdStart(config *Config) {
	cookie := os.Getenv("SCANJVM_COOKIE")

	_ = os.Setenv("LC_ALL", "C")
	_ = os.Setenv("SCANJVM_COOKIE", strings.Split(config.cookie, "=")[1])

	if !config.wait && cookie == "" {
		signals := make(chan os.Signal, 1)
		started := make(chan bool, 1)
		signal.Notify(signals, syscall.SIGUSR1)

		go func() {
			<-signals
			if status := ReadStatus(config); status != nil {
				status.Report()
			}
			started <- true
		}()

		env := func(name string) string {
			return fmt.Sprintf("%s=%s", name, os.Getenv(name))
		}

		// start a new detached process with SCANJVM_COOKIE and same args
		var attr = os.ProcAttr{
			Env: []string{
				"LC_ALL=C",
				fmt.Sprintf("SCANJVM_PID=%d", os.Getpid()),
				env("SCANJVM_COOKIE"),
				env("HOME"),
				env("USER"),
				env("PATH"),
			},
			Files: []*os.File{nil, nil, nil},
		}

		process, e := os.StartProcess(os.Args[0], os.Args, &attr)
		if e != nil {
			fmt.Println(e.Error())
			return
		}
		_ = process.Release()
		<-started
		return
	}

	// If cookie is set - this process is already detached and has no input/output

	reportStatus := func() {
		if pid, e := strconv.Atoi(os.Getenv("SCANJVM_PID")); e == nil {
			// Send signal to the parent
			_ = syscall.Kill(pid, syscall.SIGUSR1)
		} else {
			if status := ReadStatus(config); status != nil {
				status.Report()
			}
		}
	}

	lock, e := ScanLock(config)
	if e != nil {
		reportStatus()
		return
	}
	if e = lock.TryLock(); e != nil {
		// Another scan is in progress
		reportStatus()
		if config.wait {
			_ = lock.Lock()
			reportStatus()
		}
		return
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	defer lock.Unlock()
	status := NewStatus(config)

	go func() {
		<-signals
		status.SetState(Terminated)
		lock.Unlock()
		os.Exit(1)
	}()

	status.SetState(Running)
	reportStatus()

	outFile, _ := os.Create(config.OutputFilePath())
	errFile, _ := os.Create(config.ErrorFilePath())

	e = findFiles(config, func(libjvm string) {
		if info := InitJVMInstallation(libjvm, config); info != nil {
			if txt, _ := json.Marshal(info); txt != nil {
				fmt.Fprintln(outFile, string(txt))
			}
		}
	})

	if e != nil {
		fmt.Fprintln(errFile, e.Error())
		status.SetState(Error)
	} else {
		status.SetState(Finished)
	}

	if config.wait {
		status.Report()
	}
}

func CmdStatus(config *Config) {
	lock, e := ScanLock(config)
	if e != nil {
		fmt.Println(e.Error())
		return
	}

	if config.wait {
		e = lock.Lock()
	} else {
		e = lock.TryLock()
	}

	status := ReadStatus(config)

	if e == nil {
		// Process is not running, but end time is not set...
		// This is Unknown state
		if status != nil && status.EndTime == -1 {
			status.SetState(Unknown)
		}
	}

	lock.Unlock()

	if status == nil {
		// Cannot get status ... Will return an empty one
		status = NewStatus(config)
	}

	status.Report()
}

func CmdStop(config *Config) {
	procDir, e := os.ReadDir("/proc")
	if e != nil {
		fmt.Println(e.Error())
		return
	}

	for _, entry := range procDir {
		if entry.IsDir() {
			pidStr := entry.Name()
			pid, e := strconv.Atoi(pidStr)
			if e != nil {
				continue
			}
			envFile := path.Join("/proc", pidStr, "environ")
			processStringsFromFile(envFile, 0, math.MaxInt64, func(str string) bool {
				if strings.HasPrefix(str, config.cookie) {
					syscall.Kill(pid, syscall.SIGTERM)
					return false
				}
				return true
			})
		}
	}

	if status := ReadStatus(config); status != nil {
		status.Report()
	}
}
