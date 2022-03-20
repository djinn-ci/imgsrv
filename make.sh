#!/bin/sh

[ ! -d bin ] && mkdir bin

cat static/main.css | tr -d '\n' | tr -d '\t' > static/main.min.css
qtc

set -x
go generate ./...
go build -tags netgo -o bin/djinn-imgsrv
