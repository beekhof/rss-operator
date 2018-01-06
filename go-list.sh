#!/bin/bash

: ${GOPATH=$HOME/go}

function listPkgs() {
	go list ./cmd/... ./pkg/... ./test/... 
}

function listFiles() {
	# pipeline is much faster than for loop
	listPkgs | xargs -I {} find "${GOPATH}/src/{}" -name '*.go' | grep -v generated
}

listFiles
