# Copyright 2020-2021 Azul Systems, Inc. All rights reserved.
# Use of this source code is governed by the 3-Clause BSD
# license that can be found in the LICENSE file.
MYDIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

APP    := jdowser
SCRIPT := ansible-jdowser

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

FILES := \
  classfile.go \
  classfilereader.go \
  config.go \
  jvminstallation.go \
  main.go \
  scanlock.go \
  status.go \
  utils.go \

all: $(APP) $(SCRIPT)

$(APP): $(FILES:%=$(MYDIR)/src/%) $(VERSION)
	V=$$($(MYDIR)/src/version.sh) && \
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w -X \"main.VERSION=$${V}\"" -o $@ $^

$(SCRIPT): $(MYDIR)/src/$(SCRIPT)
	V=$$($(MYDIR)/src/version.sh) && \
	sed "s#%VERSION%#$${V}#" $^ > $@ && chmod +x $@

resolve:
	go get -u golang.org/x/sys/unix

clean:
	rm -f $(APP) $(SCRIPT)

.PHONY: clean resolve all
