install:
	@go get
build:
	@go build -ldflags "-s -w"
build_windows:
	GOOS=windows go build -ldflags "-s -w"
run:
	@go run main.go
test:
	@go test ./flora