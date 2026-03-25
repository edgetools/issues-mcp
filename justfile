set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

build:
    go build -o build/

install:
    go install
