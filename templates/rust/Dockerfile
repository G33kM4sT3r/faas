FROM {{.BaseImage}} AS builder
WORKDIR /app
COPY Cargo.toml ./
RUN mkdir -p src
COPY main.rs src/main.rs
RUN cargo build --release

FROM {{.RuntimeImage}}
COPY --from=builder /app/target/release/server /server
EXPOSE {{.Port}}
CMD ["/server"]
