.PHONY: all run build tests clean install uninstall

BUILD_DIR := build
PROJECT_NAME := seed

all: tests build

run: tests
	go run main.go

build: tests
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(PROJECT_NAME) main.go

tests:
	gest ./...

install:
	mv $(BUILD_DIR)/$(PROJECT_NAME) /usr/local/bin/$(PROJECT_NAME)

uninstall:
	@rm -f /usr/local/bin/$(PROJECT_NAME)

clean:
	@rm -rf $(BUILD_DIR)
