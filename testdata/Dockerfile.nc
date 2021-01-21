FROM quay.io/dockerlibrary/debian

RUN apt-get update && apt-get install netcat-openbsd -y
