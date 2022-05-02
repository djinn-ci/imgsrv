#!/bin/sh

set -e

_version() {
	git log --decorate=full --format=format:%d |
		head -1 |
		tr ',' '\n' |
		grep tag: |
		cut -d / -f 3 |
		tr -d ',)'
}

module="$(head -1 go.mod | awk '{ print $2 }')"
version="$(_version)"

if [ "$version" = "" ]; then
	version="devel $(git log -n 1 --format='format:%h %cd' HEAD)"
fi

ldflags=$(printf -- "-X 'main.Build=%s'" "$version")

[ ! -d bin ] && mkdir bin

cat static/main.css | tr -d '\n' | tr -d '\t' > static/main.min.css
qtc

set -x
go generate ./...
go build -ldflags "$ldflags" -tags netgo -o bin/djinn-imgsrv
