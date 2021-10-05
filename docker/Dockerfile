# katsubushi

FROM alpine:3.11.3
ARG VERSION
ARG TARGETARCH
ADD dist/go-katsubushi_${VERSION}_linux_${TARGETARCH}/katsubushi /usr/local/bin/katsubushi
EXPOSE 11212
ENTRYPOINT ["/usr/local/bin/katsubushi"]
