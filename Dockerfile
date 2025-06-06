FROM golang:1.22.5 AS builder

RUN apt-get update

WORKDIR /server

COPY src/go.mod server/go.sum ./

RUN go mod download

COPY src/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o myapp

FROM ubuntu:22.04

#
# set custom user
ARG USERNAME=haven
ARG USER_UID=1000
ARG USER_GID=$USER_UID

#
# install dependencies
ENV DEBIAN_FRONTEND=noninteractive
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends wget; \
    apt-get install -y --no-install-recommends supervisor; \
    apt-get install -y --no-install-recommends ffmpeg; \
    apt-get install -y --no-install-recommends curl; \
    #
    # install curl
    apt-get install -y curl; \
    #
    # create a non-root user
    groupadd --gid $USER_GID $USERNAME; \
    useradd --uid $USER_UID --gid $USERNAME --shell /bin/bash --create-home $USERNAME; \
    #
    # make directories for haven
    mkdir -p /etc/haven /var/www; \
    chown -R $USERNAME:$USERNAME /home/$USERNAME /var/www; \
    #
    # clean up
    apt-get clean -y; \
    rm -rf /var/lib/apt/lists/* /var/cache/apt/*

#
# set default envs
ENV USER=$USERNAME
ENV MEDIA_MTX_CONFIG_PATH="/etc/haven/mediamtx.yml"

#
# mandatory envs, leave FLAGSHIP_IP as "" when running as flagship
ENV PASSPHRASE="helloworldhowareyou"
ENV RTSP_PORT=8554
ENV API_PORT=8080
ENV FLAGSHIP_IP="10.42.156.11"
ENV FLAGSHIP_API_PORT=8080
ENV BACKEND_IP="10.42.156.14"

#
# mandatory envs only used when src is running as the flagship
ENV SRT_PORT=8890

#
# mandatory envs only used when src is running as the escort
ENV MAX_VIEWERS=0

#
# copy files
COPY --from=builder /src/myapp /etc/haven/server_binary
COPY supervisord.conf /etc/haven/supervisord.conf
#COPY mediamtx.yml $MEDIA_MTX_CONFIG_PATH
COPY --from=bluenviron/mediamtx:latest /mediamtx /usr/bin/mediamtx
COPY src/shared/geoLocator/GeoLite2-City.mmdb /etc/haven/GeoLite2-City.mmdb

#
# expose ports
EXPOSE 8080
EXPOSE 1935
EXPOSE 8890
EXPOSE 8554

#
# run haven
CMD ["/usr/bin/supervisord", "-s", "-c", "/etc/haven/supervisord.conf"]
