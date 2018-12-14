NAME := "docker-console"

all: clean
	packr build -o $(NAME) .	

run: clean
	go run .

install:
	go get -u github.com/gobuffalo/packr/...

clean:
	rm -rf $(NAME)
	rm -rf *.bleve
