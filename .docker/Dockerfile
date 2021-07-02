#############################
# BUILDing the docker image
#############################
FROM golang:1.14.13-buster as builder
LABEL maintainer="https://github.com/IDN-Media"

ENV GOOS=linux \
    GOARCH=amd64 

WORKDIR /build

# Set timezone
RUN echo Asia/Jakarta > /etc/timezone && \
    rm /etc/localtime && \
    ln -snf /usr/share/zoneinfo/Asia/Jakarta /etc/localtime && \
    dpkg-reconfigure -f noninteractive tzdata

# Create appuser.
ENV USER=appuser
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

RUN apt-get update && apt-get -y dist-upgrade

RUN apt-get -y install \
    build-essential \
    libssl-dev \
    ca-certificates

RUN apt-get clean && apt-get -y autoremove && \
    rm -rf /tmp/* /var/tmp/* 

COPY . .

#RUN go build
RUN go build -ldflags "-s -w" -o /build/awards.idn.media.app cmd/Main.go

#############################
# CREATE the runtime 
#############################
FROM debian:buster-slim

# Set working directory
WORKDIR /app

# Set timezone
RUN echo Asia/Jakarta > /etc/timezone && \
    rm /etc/localtime && \
    ln -snf /usr/share/zoneinfo/Asia/Jakarta /etc/localtime && \
    dpkg-reconfigure -f noninteractive tzdata

# Copy user
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Install dependencies
RUN apt-get update && apt-get -y install \
    libssl-dev \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Cleanup
RUN apt-get clean && apt-get -y autoremove && \
    rm -rf /tmp/* /var/tmp/* 

# Copy executable file
COPY --from=builder --chown=appuser:appuser /build/awards.idn.media.app /app/

# Change directory permission

RUN chown -R appuser:appuser /app/

# Use an unprivileged user.
USER appuser:appuser


# Bind host from any ip
ENV SERVER_HOST=0.0.0.0
# Expose port
EXPOSE 50051
CMD ["/app/awards.idn.media.app"]