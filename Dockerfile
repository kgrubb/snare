# goreleaser dockers_v2 lays binaries under $TARGETPLATFORM/.
FROM gcr.io/distroless/static-debian12:nonroot
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/snare /snare
COPY config.example.yaml /config.example.yaml
USER nonroot:nonroot
EXPOSE 161/udp
ENTRYPOINT ["/snare"]
CMD ["--config", "/config/config.yaml"]
