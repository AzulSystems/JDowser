// Copyright 2020 Azul Systems, Inc. All rights reserved.
// Use of this source code is governed by the 3-Clause BSD
// license that can be found in the LICENSE file.

package jdowser

import (
	"golang.org/x/sys/unix"
	"os"
	"path"
)

type FLock struct {
	lockFilePath string
	lockFile     *os.File
}

func ScanLock(config *Config) (*FLock, error) {
	return &FLock{
		path.Join(config.logdir, ".lck"),
		nil,
	}, nil
}

func (l *FLock) TryLock() error {
	fh, err := os.OpenFile(l.lockFilePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	if err = unix.Flock(int(fh.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return err
	}
	l.lockFile = fh
	return nil
}

func (l *FLock) Lock() error {
	fh, err := os.OpenFile(l.lockFilePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	if err = unix.Flock(int(fh.Fd()), unix.LOCK_EX); err != nil {
		return err
	}
	l.lockFile = fh
	return nil
}

func (l *FLock) Unlock() {
	if l.lockFile != nil {
		_ = l.lockFile.Close()
		l.lockFile = nil
	}
}
