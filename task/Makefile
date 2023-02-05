test.all:
	go test -tags=all ./... -cover

test.all.info:
	go test -tags=all ./... -v -cover

test.unit:
	go test -tags=unit ./internal/domain/task -cover

test.unit.info:
	go test -tags=unit ./internal/domain/task -v -cover

test.integration:
	go test -tags=integration ./itests

test.integration.info:
	go test -tags=integration ./itests -v

lint:
	golangci-lint run

mock-gen:
	go generate ./internal/adaptors/client/...
	