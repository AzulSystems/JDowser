# Copyright 2020-2021 Azul Systems, Inc. All rights reserved.
# Use of this source code is governed by the 3-Clause BSD
# license that can be found in the LICENSE file.

# ./jdowser start
# ./jdowser status
# ./jdowser -json report > output_file.json
# ./jdowser -csv report > output_file.csv


build-dist:
	rm -rf dist 
	CGO_ENABLED=0 go build -o dist/jdowser main.go
	cp ansible-jdowser dist/ansible-jdowser.sh

