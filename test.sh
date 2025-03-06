#!/bin/sh

nohup dockerd -l fatal -s vfs &

machine="amd64"

case "$(uname -m)" in
  aarch64)
    machine="arm64"
    ;;
esac
  

url=https://go.dev/dl/go${GOLANG_VERSION:-1.24.0}.linux-${machine}.tar.gz 
echo "Fetching golang from $url"
curl -sSL "$url" | tar -xz -C /usr/local

PATH="${PATH}:/usr/local/go/bin" go test -v ./... -count 1
