FROM golang:1.20

ARG AIR_VERSION=1.40.4

RUN curl -sSL https://github.com/air-verse/air/releases/download/v${AIR_VERSION}/air_${AIR_VERSION}_linux_amd64.tar.gz \
    | tar -xz -C /usr/local/bin air

WORKDIR /src

CMD ["air", "-c", "cmd/processor/.air.toml"]

