buildAll: buildLinux buildWindows buildMac

buildLinux:
	env GOOS=linux GOARCH=amd64 go build -o ./bin/yamlParser-linux-amd64 ./

buildWindows:
	env GOOS=windows GOARCH=amd64 go build -o ./bin/yamlParser-x64.exe ./

buildMac:
	env GOOS=darwin GOARCH=amd64 go build -o ./bin/yamlParser-darwin-amd64 ./
	env GOOS=darwin GOARCH=arm64 go build -o ./bin/yamlParser-darwin-arm64 ./
