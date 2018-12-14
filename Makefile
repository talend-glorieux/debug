NAME := "docker-console"

all: clean
	packr build -o $(NAME) .	

run: clean
	go run .

clean:
	rm -rf $(NAME)

install:
	go get -u github.com/gobuffalo/packr/...
