FROM golang:1.23 AS build
COPY . /build
WORKDIR /build
RUN make build

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /build/dist/purr /
WORKDIR /
USER 1000:1000
VOLUME /cache
CMD ["/purr"]