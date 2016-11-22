RELEASE_PATH = release/flora/

install:
	@go get
build:
	@go build -ldflags "-s -w"
release:
	@make build
	@make package
package:
	rm -Rf $(RELEASE_PATH)/*
	mkdir -p $(RELEASE_PATH)
	cp ./flora-kit $(RELEASE_PATH)
	cp ./flora.default.conf $(RELEASE_PATH)
	cp ./geoip.mmdb $(RELEASE_PATH)
	cp ./LICENSE $(RELEASE_PATH)
	cp ./README.md $(RELEASE_PATH)
	cd ./release && tar zcf flora.tar.gz flora
build_windows:
	GOOS=windows go build -ldflags "-s -w"
run:
	@go run main.go
test:
	@go test ./flora