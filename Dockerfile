# syntax=docker/dockerfile:1.4

FROM alpine:3.19
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Copy the multi-arch pre-built binary
COPY bin/combine-mcp-${TARGETOS}-${TARGETARCH} /usr/local/bin/combine-mcp
RUN chmod +x /usr/local/bin/combine-mcp

USER 1000:1000

ENTRYPOINT ["/usr/local/bin/combine-mcp"] 