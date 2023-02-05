#!/bin/sh

mockgen -source ./auth_grpc.pb.go > ./mock/mock_auth.go
