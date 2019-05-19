.PHONY: test test-race test-cover cover
.DEFAULT_GOAL: test

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -cover -coverprofile cover.out  ./...

cover: test-cover
	go tool cover -func cover.out
	go tool cover -o cover.html -html cover.out
