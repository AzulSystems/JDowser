# Copyright 2020-2021 Azul Systems, Inc. All rights reserved.
# Use of this source code is governed by the 3-Clause BSD
# license that can be found in the LICENSE file.

build-dist:
	rm -rf dist 
	GOOS=linux GOARCH=amd64 go build -o dist/jdowser_linux main.go
	GOOS=darwin GOARCH=amd64 go build -o dist/jdowser_mac main.go
	GOOS=windows GOARCH=amd64 go build -o dist/jdowser_windows.exe main.go

