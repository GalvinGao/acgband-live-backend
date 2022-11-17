FROM golang:1.19.3-alpine AS base
WORKDIR /app

# builder
FROM base AS builder
ENV GOOS linux
ENV GOARCH amd64

RUN apk --no-cache add bash git openssh

# modules: utilize build cache
COPY go.mod ./
COPY go.sum ./

# RUN go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod download
COPY . .

# inject versioning information & build the binary
RUN go build -o backend .

# runner
FROM base AS runner
RUN apk add --no-cache libc6-compat tini
# Tini is now available at /sbin/tini

COPY --from=builder /app/backend /app/backend
EXPOSE 8080

ENTRYPOINT ["/sbin/tini", "--"]
CMD [ "/app/backend" ]
