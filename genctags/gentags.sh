#!/bin/bash

# generate pass one tags
pushd ctags_parser
go run main.go 3.9 3.12
popd

