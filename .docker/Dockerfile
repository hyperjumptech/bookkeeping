FROM golang:1.17-alpine as builder
LABEL maintainer="https://github.com/hyperjumptech/bookkeeping"

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
RUN go mod tidy
RUN go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o bookkeeping-go-img cmd/Main.go


#############################
# CREATE the runtime 
#############################
FROM scratch
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

COPY --from=builder /build/bookkeeping-go-img /app/
WORKDIR /app

# Use an unprivileged user.
USER appuser:appuser
EXPOSE 7000

CMD ["/app/bookkeeping-go-img"]