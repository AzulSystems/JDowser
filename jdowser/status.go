// Copyright 2020 Azul Systems, Inc. All rights reserved.
// Use of this source code is governed by the 3-Clause BSD
// license that can be found in the LICENSE file.

package jdowser

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type StateType string

type Status struct {
	Hostname  string    `json:"host"`
	State     StateType `json:"state"`
	StartTime int64     `json:"start_time"`
	EndTime   int64     `json:"end_time"`
	Args      []string  `json:"args"`
	Error     []string  `json:"error,omitempty"`
	Config    *Config   `json:"-"`
}

const (
	Running    StateType = "Running"
	Finished   StateType = "Finished"
	Terminated StateType = "Terminated"
	Unknown    StateType = "Unknown"
	Error      StateType = "Error"
)

func NewStatus(config *Config) *Status {
	hostname, _ := os.Hostname()
	var args []string
	if config.Command == CMD_START {
		args = os.Args[1 : len(os.Args)-1]
	}

	s := &Status{
		Hostname:  hostname,
		State:     Unknown,
		StartTime: -1,
		EndTime:   -1,
		Args:      args,
		Config:    config,
	}
	return s
}

func ReadStatus(config *Config) *Status {
	data, e := os.ReadFile(config.StatusFilePath())
	if e != nil {
		return nil
	}
	status := &Status{}
	e = json.Unmarshal(data, status)
	if e != nil {
		return nil
	}
	status.Config = config

	file, err := os.Open(config.ErrorFilePath())
	if err == nil {
		defer closeFile(file)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			txt := strings.TrimSpace(scanner.Text())
			if txt != "" {
				status.Error = append(status.Error, txt)
			}
		}
	}

	return status
}

func (status *Status) SetState(state StateType) {
	switch state {
	case Running:
		status.StartTime = time.Now().Unix()
	case Finished, Terminated:
		status.EndTime = time.Now().Unix()
	case Unknown:
		status.EndTime = -2
	}

	status.State = state
	f, _ := os.Create(status.Config.StatusFilePath())
	txt, _ := json.Marshal(status)
	fmt.Fprintln(f, string(txt))
}

func endTime(status *Status) string {
	if status.EndTime < 0 {
		return strconv.FormatInt(int64(status.EndTime), 10)
	} else {
		return time.Unix(status.EndTime, 0).String()
	}
}

func (status *Status) Report() {
	if status.Config.Json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(status)
	} else if status.Config.CSV {
		w := csv.NewWriter(os.Stdout)
		w.Write([]string{"host", "state", "start_time", "end_time", "args"})
		w.Write([]string{
			status.Hostname,
			string(status.State),
			time.Unix(status.StartTime, 0).String(),
			endTime(status),
			strings.Trim(fmt.Sprint(status.Args), "][")})
		w.Flush()
	} else {
		fmt.Println("host:", status.Hostname)
		fmt.Println("state:", status.State)
		fmt.Println("start_time:", time.Unix(status.StartTime, 0).String())
		fmt.Println("end_time:", endTime(status))
		fmt.Println("args:", strings.Trim(fmt.Sprint(status.Args), "]["))
		if len(status.Error) > 0 {
			fmt.Println("error:", strings.Trim(fmt.Sprint(status.Error), "]["))
		}
	}
}
