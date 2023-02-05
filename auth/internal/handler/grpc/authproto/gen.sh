#!/bin/sh

mkdir -p ../../../../pkg
protoc  ./auth.proto --go-grpc_out=../../../../pkg --go_out=../../../../pkg