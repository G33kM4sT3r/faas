FROM {{.BaseImage}} AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server .

FROM {{.RuntimeImage}}
COPY --from=builder /server /server
EXPOSE {{.Port}}
CMD ["/server"]
