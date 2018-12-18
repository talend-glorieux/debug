NAME="docker-console"
VERSION="0.1.0-alpha"
LDFLAGS=-ldflags "-X main.Version=${VERSION}"

all: clean
	packr build $(LDFLAGS) -o $(NAME) .

run: clean
	go run .

install:
	go get -u github.com/gobuffalo/packr/...

release: clean
	mkdir release
	env GOOS=darwin GOARCH=amd64 packr build $(LDFLAGS) -o release/$(NAME)_$(VERSION)_darwin-amd64 .
	env GOOS=linux GOARCH=amd64 packr build $(LDFLAGS) -o release/$(NAME)_$(VERSION)_linux-amd64 .
	env GOOS=windows GOARCH=amd64 packr build $(LDFLAGS) -o release/$(NAME)_$(VERSION)_windows-amd64.exe .

clean:
	rm -rf $(NAME)
	rm -rf release
