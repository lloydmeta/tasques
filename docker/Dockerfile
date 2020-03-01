############################
# STEP 1 build executable binary
############################
FROM golang@sha256:e484434a085a28801e81089cc8bcec65bc990dd25a070e3dd6e04b19ceafaced AS builder

# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

# Create unprivileged user
ENV USER=tasques
ENV UID=10001

# See https://stackoverflow.com/a/55757473/12429735RUN
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

RUN mkdir /build
ADD . /build/
WORKDIR /build

# Fetch dependencies.
# Using go get.
# Using go mod with go 1.11
RUN go mod download
RUN go mod verify

# Build the binary.
RUN GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -ldflags="-w -s" -o /app/bin/tasques ./app/main.go

# Setup dirs
RUN mkdir /app/config && chown tasques:tasques /app/config
RUN mkdir /app/logs && chown tasques:tasques /app/logs

############################
# STEP 2 build a small image
############################
FROM scratch

ARG VCS_REF
ARG BUILD_DATE

LABEL org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url="https://github.com/lloydmeta/tasques" \
      org.label-schema.build-date=$BUILD_DATE

# Import from builder.
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy our static executable.
COPY --from=builder /app/bin/tasques /app/bin/tasques
COPY --from=builder /app/config /app/config
COPY --from=builder /app/logs /app/logs

# Volumes
VOLUME /app/config /app/logs

# Use our unprivileged user.
USER tasques:tasques

# Port on which the service will be exposed.
EXPOSE 8080

ENTRYPOINT ["/app/bin/tasques"]