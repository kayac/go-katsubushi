# katsubushi
#
# VERSION 1.6.0

FROM alpine:3.11.3 AS build-env
MAINTAINER Fujiwara Shunichiro <fujiwara.shunichiro@gmail.com>

RUN apk --no-cache add unzip curl
WORKDIR /tmp
RUN curl -sL https://github.com/kayac/go-katsubushi/releases/download/v1.6.0/katsubushi-1.6.0-linux-amd64.zip \
    > /tmp/katsubushi-1.6.0-linux-amd64.zip && \
    unzip katsubushi-1.6.0-linux-amd64.zip

FROM alpine:3.11.3
COPY --from=build-env /tmp/katsubushi /usr/local/bin/katsubushi
EXPOSE 11212
ENTRYPOINT ["/usr/local/bin/katsubushi"]
