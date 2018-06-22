# katsubushi
#
# VERSION 1.5.3

FROM alpine:3.7 AS build-env
MAINTAINER Fujiwara Shunichiro <fujiwara.shunichiro@gmail.com>

RUN apk --no-cache add unzip curl
RUN curl -sL https://github.com/kayac/go-katsubushi/releases/download/v1.5.3/katsubushi-1.5.3-linux-amd64.zip > /tmp/katsubushi-1.5.3-linux-amd64.zip && cd /tmp && unzip katsubushi-1.5.3-linux-amd64.zip

FROM alpine:3.7
COPY --from=build-env /tmp/katsubushi /usr/local/bin/katsubushi
EXPOSE 11212
ENTRYPOINT ["/usr/local/bin/katsubushi"]
