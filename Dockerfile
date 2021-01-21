FROM quay.io/dockerlibrary/debian:latest

RUN apt-get update -qq && apt-get install curl build-essential -y && curl -sSL get.docker.com | bash
RUN curl -sSL https://github.com/containers/fuse-overlayfs/releases/download/v1.3.0/fuse-overlayfs-x86_64 >/usr/bin/fuse-overlayfs && chmod +x /usr/bin/fuse-overlayfs

CMD ["/bin/bash", "test.sh"]
