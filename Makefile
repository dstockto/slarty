.PHONY:
	echo "Make slarty!"

format:
	go fmt go fmt github.com/dstockto/slarty/...

build-all: mac-binary linux-amd-binary linux-arm-binary windows-binary

mac-binary:
	GOOS=darwin GOARCH=amd64 go build -o build/mac/slarty
	chmod +x build/mac/slarty

linux-amd-binary:
	GOOS=linux GOARCH=amd64 go build -o build/linux-amd64/slarty

linux-arm-binary:
	GOOS=linux GOARCH=arm64 go build -o build/linux-arm64/slarty

windows-binary:
	GOOS=windows GOARCH=amd64 go build -o build/windows-amd64/slarty.exe