.PHONY:
	echo "Make slarty!"

format:
	go fmt github.com/dstockto/slarty/...

build-all: clean mac-amd64-binary mac-arm64-binary linux-amd-binary linux-arm-binary windows-binary

clean:
	rm -rf build/*

mac-amd64-binary:
	GOOS=darwin GOARCH=amd64 go build -o build/mac-amd64/slarty
	chmod +x build/mac-amd64/slarty

mac-arm64-binary:
	GOOS=darwin GOARCH=arm64 go build -o build/mac-arm64/slarty
	chmod +x build/mac-arm64/slarty

linux-amd-binary:
	GOOS=linux GOARCH=amd64 go build -o build/linux-amd64/slarty

linux-arm-binary:
	GOOS=linux GOARCH=arm64 go build -o build/linux-arm64/slarty

windows-binary:
	GOOS=windows GOARCH=amd64 go build -o build/windows-amd64/slarty.exe