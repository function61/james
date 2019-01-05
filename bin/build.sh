#!/bin/bash -eu

source /build-common.sh

COMPILE_IN_DIRECTORY="cmd/james"
BINARY_NAME="james"
BINTRAY_PROJECT="function61/james"

# go-yaml contains non-gofmt'd code
GOFMT_TARGETS="cmd/"

standardBuildProcess
