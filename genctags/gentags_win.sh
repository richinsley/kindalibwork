#!/bin/bash

# generate pass one tags
pushd passone
go run main_pass_one.go windows
popd

pushd passtwo/pkg/platformpy39
go run main.go 39 windows
popd

pushd passtwo/pkg/platformpy310
go run main.go 310 windows
popd

pushd passtwo/pkg/platformpy311
go run main.go 311 windows
popd

pushd passtwo/pkg/platformpy312
go run main.go 312 windows
popd
