# Runtime image for the opm CLI, built by GoReleaser (dockers_v2).
#
# GoReleaser compiles the binary and stages it per platform in the build
# context, so nothing is built here: $TARGETPLATFORM is linux/amd64 or
# linux/arm64 and holds the matching opm binary. The binary is pure Go
# (CGO_ENABLED=0), which is why a static distroless base is enough. The base
# ships a CA bundle for HTTPS registries and a nonroot user.
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot@sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3
ARG TARGETPLATFORM
WORKDIR /
COPY ${TARGETPLATFORM}/opm /opm

USER 65532:65532

ENTRYPOINT ["/opm"]
