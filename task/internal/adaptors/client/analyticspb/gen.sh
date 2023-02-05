#!/bin/sh

mockgen -source ./analytics_grpc.pb.go > ./mock/mock_analytics.go
