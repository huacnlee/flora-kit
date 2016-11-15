install:
	@go get
build:
	@go build main.go
run:
	@go run main.go
test:
	@go test ./flora