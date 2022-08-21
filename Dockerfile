FROM ubuntu
MAINTAINER borepstein@gmail.com
#
# Non-interactive initialization
#
ENV DEBIAN_FRONTEND noninteractive
ENV DEBCONF_NONINTERACTIVE_SEEN true
RUN apt-get update -y --fix-missing
RUN apt-get install -y git wget build-essential iputils-ping iproute2 telnet
#
# Installing Golang and hyprspace
#
WORKDIR /app
RUN wget https://go.dev/dl/go1.17.11.linux-amd64.tar.gz
RUN tar xvf go1.17.11.linux-amd64.tar.gz
RUN mv go /usr/local
RUN ln -s /usr/local/go/bin/go /usr/local/bin
COPY . ./hyprspace/
RUN find /app/hyprspace -exec ls -ld {} \;
RUN cd /app/hyprspace;go build
RUN mv /app/hyprspace /usr/local
RUN ln -s /usr/local/hyprspace/hyprspace /usr/local/bin/hyprspace
RUN mkdir /etc/hyprscale
# Create TUN device
RUN [ -d /dev/net ] || mkdir /dev/net
RUN cd /dev/net; mknod tun c 10 100
RUN hyprspace init hs0
CMD hyprspace up hs0; while true; do sleep 60; done
