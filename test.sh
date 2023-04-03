#!/bin/sh

nohup dockerd -s vfs &

curl -sSL https://storage.googleapis.com/golang/go${GOLANG_VERSION:-1.20.1}.linux-amd64.tar.gz | tar -xz -C /usr/local

PATH="${PATH}:/usr/local/go/bin" go test -v ./... -count 1
