.PHONY: buildAll buildLinux buildLinux386 buildLinuxAmd64 buildMac buildMacAmd64 buildMacArm64 buildWindows installLint fmt lint lintFix test exampleStdout example

all: tidy fmt lint test

buildAll: buildLinux buildWindows buildMac

buildLinux386:
	env GOOS=linux GOARCH=386 go build -ldflags "-s -w" -o ./bin/zparser-linux-i386 ./cmd/main.go

buildLinuxAmd64:
	env GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o ./bin/zparser-linux-amd64 ./cmd/main.go

buildMacAmd64:
	env GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o ./bin/zparser-darwin-amd64 ./cmd/main.go

buildMacArm64:
	env GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o ./bin/zparser-darwin-arm64 ./cmd/main.go

buildWindows:
	env GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o ./bin/zparser-win-x64.exe ./cmd/main.go

buildLinux: buildLinuxAmd64 buildLinux386
buildMac: buildMacAmd64 buildMacArm64

installLint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.61.0

tidy:
	go mod tidy

fmt:
	gofmt -s -w ./src/. ./cmd/.

lint:
	golangci-lint run ./src/... ./cmd/... --verbose

lintFix:
	golangci-lint run ./src/... ./cmd/... --verbose --fix

test:
	go test -v ./src/... ./cmd/...

exampleStdout:
	go run ./cmd/main.go ./example.yml

example:
	go run ./cmd/main.go ./example.yml -f example.parsed.yml
