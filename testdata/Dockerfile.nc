FROM debian

RUN apt-get update && apt-get install netcat-openbsd -y
