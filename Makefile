RELEASE_PATH = release/flora/

install:
	@go get
build:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o release/flora-kit-darwin-amd64
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o release/flora-kit-linux-amd64
	GOOS=linux GOARCH=386 go build -ldflags "-s -w" -o release/flora-kit-linux-386
package:
	rm -Rf $(RELEASE_PATH)/*
	mkdir -p $(RELEASE_PATH)
	cp ./flora.default.conf $(RELEASE_PATH)
	cp ./geoip.mmdb $(RELEASE_PATH)
	cp ./LICENSE $(RELEASE_PATH)
	cp ./README.md $(RELEASE_PATH)
	# macOS
	cp ./release/flora-kit-darwin-amd64 $(RELEASE_PATH)flora
	cd ./release && zip flora-darwin-amd64.zip flora
	# Linux amd64
	cp ./release/flora-kit-linux-amd64 $(RELEASE_PATH)flora
	cd ./release && tar zcf flora-linux-amd64.tar.gz flora
	# Linux 386
	cp ./release/flora-kit-linux-386 $(RELEASE_PATH)flora
	cd ./release && tar zcf flora-linux-386.tar.gz flora
	# remove history
	rm $(RELEASE_PATH)flora
run:
	@go run main.go
test:
	@go test ./flora