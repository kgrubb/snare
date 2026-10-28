# goreleaser copies the built binary next to this Dockerfile.
FROM gcr.io/distroless/static-debian12:nonroot
COPY snare /snare
COPY config.example.yaml /config.example.yaml
USER nonroot:nonroot
EXPOSE 161/udp
ENTRYPOINT ["/snare"]
CMD ["--config", "/config/config.yaml"]
