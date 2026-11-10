APP      := qemu-cli
VERSION  := 1.0.0
BIN_DIR  := bin
LDFLAGS  := -ldflags "-X github.com/nycon/qemu-cli/cmd.Version=$(VERSION)"

.PHONY: build install clean test smoke tidy release-binaries

build:
	mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(APP) .

install: build
	mkdir -p $(HOME)/go/bin $(HOME)/.local/bin
	install -m 755 $(BIN_DIR)/$(APP) $(HOME)/go/bin/$(APP)
	install -m 755 $(BIN_DIR)/$(APP) $(HOME)/.local/bin/$(APP)
	@echo "Installed to $(HOME)/go/bin/$(APP) and $(HOME)/.local/bin/$(APP)"

release-binaries:
	bash scripts/build-release.sh $(VERSION)

tidy:
	go mod tidy

test:
	go test ./...

smoke: build
	./$(BIN_DIR)/$(APP) --help >/dev/null
	./$(BIN_DIR)/$(APP) context --help >/dev/null
	./$(BIN_DIR)/$(APP) vm --help >/dev/null
	./$(BIN_DIR)/$(APP) version
	@echo "smoke ok"

clean:
	rm -rf $(BIN_DIR) dist
