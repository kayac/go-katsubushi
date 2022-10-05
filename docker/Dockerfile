# katsubushi

FROM alpine:3.16.2
ARG VERSION
ARG TARGETARCH
ADD dist/go-katsubushi_linux_${TARGETARCH}/katsubushi /usr/local/bin/katsubushi
EXPOSE 11212
ENTRYPOINT ["/usr/local/bin/katsubushi"]
