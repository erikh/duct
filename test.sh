#!/bin/sh

killdocker() {
  pkill dockerd
}

trap killdocker INT HUP TERM

nohup dockerd -s vfs &

curl -sSL https://storage.googleapis.com/golang/go${GOLANG_VERSION:-1.15.7}.linux-amd64.tar.gz | tar -xz -C /usr/local

PATH="${PATH}:/usr/local/go/bin" go test -v ./... -count 1
