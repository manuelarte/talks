# Run the application
run:
    cargo run

fmt:
    cargo fmt --all -- --check

# Run all the linters (clippy + rustfmt check)
lint: fmt
    cargo clippy --all-targets --all-features -- -D warnings

nightly:
    rustup override set nightly

ast: nightly
    rustc -Z unpretty=ast-tree src/main.rs

hir: nightly
    rustc -Z unpretty=hir-tree src/main.rs
