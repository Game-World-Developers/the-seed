.PHONY: all run build tests clean

BUILD_DIR := build
PROJECT_NAME := seed

all: run

run:
	go run main.go

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(PROJECT_NAME) main.go

tests: 
	go test -v ./...

clean:
	@rm -rf $(BUILD_DIR)
