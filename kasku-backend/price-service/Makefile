.PHONY: build run test lint docker-build clean

SERVICE_NAME = price-service

build:
	cargo build --release

run:
	cargo run

test:
	cargo test

lint:
	cargo clippy -- -D warnings

docker-build:
	docker build -t kasku-$(SERVICE_NAME):latest .

clean:
	cargo clean
