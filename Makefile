.PHONY: all clean linux windows

all: clean linux windows

clean:
	rm -Rf build || true

linux:
	mkdir -p build/linux
	GOOS=linux GOARCH=amd64 go build -o build/linux/distlock main.go

windows:
	mkdir -p build/windows
	GOOS=windows GOARCH=amd64 go build -o build/linux/distlock.exe main.go