# KeenUPS

KeenUPS turns a Keenetic router with Entware and a USB-connected UPS into a network UPS server. It supervises NUT's `usbhid-ups` driver and publishes the UPS state over NUT protocol 1.3 and a read-only HTTP API.

The initial supported configuration is Powercom WOW-500U (`0d9f:0004`) on Keenetic ARM64 (`aarch64-3.10`). The defaults are compatible with Synology DSM and other NUT clients on the local network.

## Setup guides

- [Инструкция по настройке на русском](docs/setup.ru.md)
- [Setup guide in English](docs/setup.en.md)

## Development

```sh
make test
make build-keenetic
make package-keenetic VERSION=v1.0.0
```

Tags matching `v*.*.*` trigger tests, an ARM64 IPK build, and a GitHub Release.

## License

[MIT](LICENSE)
