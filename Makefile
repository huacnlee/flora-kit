RELEASE_PATH = release
PACKAGE_PATH = release/flora

install:
	@go get
build:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o $(RELEASE_PATH)/flora-darwin-amd64
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o $(RELEASE_PATH)/flora-amd64
	GOOS=linux GOARCH=386 go build -ldflags "-s -w" -o $(RELEASE_PATH)/flora-386
	GOOS=windows GOARCH=386 go build -ldflags "-s -w" -o $(RELEASE_PATH)/flora-386.exe
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o $(RELEASE_PATH)/flora-amd64.exe
package:
	rm -Rf $(PACKAGE_PATH)/*
	mkdir -p $(PACKAGE_PATH)
	cp ./flora.default.conf $(PACKAGE_PATH)
	cp ./geoip.mmdb $(PACKAGE_PATH)
	cp ./LICENSE $(RELEASE_PATH)
	cp ./README.md $(PACKAGE_PATH)
	# macOS
	cp ./release/flora-darwin-amd64 $(PACKAGE_PATH)
	cd ./release && zip flora-darwin-amd64.zip flora
	# Linux amd64
	cp ./release/flora-amd64 $(PACKAGE_PATH)flora
	cd ./release && tar zcf flora-linux-amd64.tar.gz flora
	# Linux 386
	cp ./release/flora-386 $(PACKAGE_PATH)
	cd ./release && tar zcf flora-linux-386.tar.gz flora
	# Windows 386
    #cp ./release/flora-386.exe $(PACKAGE_PATH)
    #cd ./release && tar zcf flora-win-386.tar.gz flora
    # Windows amd64
    #cp ./release/flora-amd64.exe $(PACKAGE_PATH)
    #cd ./release && tar zcf flora-win-amd64.tar.gz flora
	# remove history
	rm $(PACKAGE_PATH)flora
run:
	@go run main.go
test:
	@go test ./flora
