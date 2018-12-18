NAME="docker-console"
VERSION="0.1.0-alpha"
LDFLAGS=-ldflags "-X main.Version=${VERSION}"

all: clean
	packr build $(LDFLAGS) -o $(NAME) .

run: clean
	CompileDaemon -command="./docker-console -open=false" -graceful-kill=true -color=true -include="*.html" -include="*.css" -include="*.js"



install:
	go get -u github.com/gobuffalo/packr/...
	go get -u github.com/githubnemo/CompileDaemon

release: clean
	mkdir release
	env GOOS=darwin GOARCH=amd64 packr build $(LDFLAGS) -o release/$(NAME)_$(VERSION)_darwin_amd64 .
	env GOOS=linux GOARCH=amd64 packr build $(LDFLAGS) -o release/$(NAME)_$(VERSION)_linux_amd64 .
	env GOOS=windows GOARCH=amd64 packr build $(LDFLAGS) -o release/$(NAME)_$(VERSION)_windows_amd64.exe .

clean:
	rm -rf $(NAME)
	rm -rf release
