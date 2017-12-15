# katsubushi
#
# VERSION 1.4.3

FROM alpine:3.6
MAINTAINER Fujiwara Shunichiro <fujiwara.shunichiro@gmail.com>

RUN apk --update add unzip curl && rm -rf /var/cache/apk/*
RUN curl -sL https://github.com/kayac/go-katsubushi/releases/download/v1.4.3/katsubushi-1.4.3-linux-amd64.zip > /tmp/katsubushi-1.4.3-linux-amd64.zip && cd /tmp && unzip katsubushi-1.4.3-linux-amd64.zip && install katsubushi /usr/local/bin && rm -f /tmp/katsubushi*

ENV worker_id 1
EXPOSE 11212
CMD ["sh", "-c", "exec /usr/local/bin/katsubushi -worker-id $worker_id"]
