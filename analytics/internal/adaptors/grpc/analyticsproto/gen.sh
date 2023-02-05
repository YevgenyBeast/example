#!/bin/sh

mkdir -p ../../../../pkg
protoc  ./analytics.proto --go-grpc_out=../../../../pkg --go_out=../../../../pkg