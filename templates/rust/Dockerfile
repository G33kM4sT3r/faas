FROM {{.BaseImage}} AS builder
WORKDIR /app
COPY Cargo.toml Cargo.lock ./
COPY src/ src/
RUN cargo build --release

FROM {{.RuntimeImage}}
COPY --from=builder /app/target/release/server /server
EXPOSE {{.Port}}
CMD ["/server"]
