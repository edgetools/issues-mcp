set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

build:
    go build -o build/

test:
    go test ./...

install:
    go install
