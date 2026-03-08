FROM {{.BaseImage}} AS builder
WORKDIR /app
COPY go.mod main.go handler.go ./
{{- if .HasDependencies}}
RUN go get{{range .Dependencies}} {{.}}{{end}} && go mod tidy
{{- end}}
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server .

FROM {{.RuntimeImage}}
COPY --from=builder /server /server
EXPOSE {{.Port}}
CMD ["/server"]
