## Cipher worker-rs

A Rust implementation, based on an OpenAPI-generated client.

### Running

This requires nightly Rust (the official native ES client, plus the usage of async traits).

```shell script
RUST_LOG=info cargo +nightly run -- --worker-id rusty-worker-1
```

The `RUST_LOG` param for setting logging level is optional.