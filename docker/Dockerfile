# katsubushi

FROM alpine:3.11.3
ARG VERSION
ADD dist/go-katsubushi_${VERSION}_linux_amd64/katsubushi /usr/local/bin/katsubushi
EXPOSE 11212
ENTRYPOINT ["/usr/local/bin/katsubushi"]
