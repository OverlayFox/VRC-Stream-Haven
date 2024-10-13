FROM golang:1.22.5 AS builder

RUN apt-get update && \

WORKDIR /server

COPY server/go.mod server/go.sum ./

RUN go mod download

COPY server/ .

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
    apt-get install -y --no-install-recommends \
        wget supervisor ffmpeg \
    #
    # install curl
    apt-get install -y curl; \
    #
    # create a non-root user
    groupadd --gid $USER_GID $USERNAME; \
    useradd --uid $USER_UID --gid $USERNAME --shell /bin/bash --create-home $USERNAME; \
    #
    # make directories for hermes
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
ENV API_URL="http://localhost:8080"
ENV IS_NODE="False"
ENV PASSPHRASE="helloworldhowareyou"
ENV MAX_VIEWERS=0

#
# copy files
COPY --from=builder /server/myapp /etc/haven/server_binary
COPY supervisord.conf /etc/haven/supervisord.conf
#COPY mediamtx.yml $MEDIA_MTX_CONFIG_PATH
COPY --from=bluenviron/mediamtx:latest /mediamtx /usr/bin/mediamtx
COPY bashScripts /etc/haven/bashScripts

#
# expose ports
EXPOSE 8080
EXPOSE 1935
EXPOSE 8890
EXPOSE 9997

#
# run hermes
CMD ["/usr/bin/supervisord", "-s", "-c", "/etc/haven/supervisord.conf"]
