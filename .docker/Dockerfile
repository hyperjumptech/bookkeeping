FROM golang:alpine as builder
LABEL maintainer="com.idnmedia.awards"

# install openssl
RUN apk add --update openssl

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 

WORKDIR /build

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

COPY . .

RUN go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o awards-go-img cmd/Main.go

# Bind host from any ip
ENV SERVER_HOST=0.0.0.0

EXPOSE 7000

CMD ["/build/awards-go-img"]
