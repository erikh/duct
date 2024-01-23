FROM debian:latest

RUN apt-get update && apt-get install netcat-openbsd inetutils-ping -y
