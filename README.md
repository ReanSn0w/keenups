# KeenUPS

KeenUPS turns a Keenetic router with Entware/OPKG and a USB-connected UPS into a small network UPS appliance. The first supported device is the Powercom WOW-500U (`0d9f:0004`).

The Go service deliberately delegates USB/HID decoding to NUT's proven `usbhid-ups` driver. It supervises that driver, reconnects to its local state socket, suppresses the WOW-500U's bogus startup readings, and publishes the normalized state over:

- NUT protocol 1.3 on TCP port `3493`;
- a read-only HTTP page and JSON API on TCP port `8080`.

The Go binary itself has no runtime dependencies and can be cross-compiled with `CGO_ENABLED=0`. The external NUT driver still uses the `libusb` package supplied by Entware.

## Build

```sh
make test
make build-keenetic
```

The target router observed during development is `linux/arm64` (`aarch64`). Copy `dist/keenups-linux-arm64` to `/opt/bin/keenups`.

## Router prerequisites

Install the USB driver:

```sh
opkg install nut-driver-usbhid-ups
```

Copy [`configs/ups.conf.example`](configs/ups.conf.example) to `/opt/etc/nut/ups.conf`. Only one `usbhid-ups` instance may control the UPS, so stop any manually suspended debug job before starting KeenUPS.

For a foreground smoke test:

```sh
/opt/bin/keenups
```

The defaults are tailored to Entware on Keenetic:

| Setting | Default |
|---|---|
| Driver | `/opt/lib/nut/usbhid-ups` |
| Driver state socket | `/opt/var/run/usbhid-ups-keenups` |
| NUT listen address | `:3493` |
| HTTP listen address | `:8080` |
| Network UPS name | `ups` |
| Monitor credentials | `monuser` / `secret` |
| Startup reading suppression | 8 seconds |

Options are parsed with [go-flags](https://github.com/jessevdk/go-flags). Every option has a GNU-style command-line flag (`keenups --help`) and a matching `KEENUPS_*` environment variable. Prefer the environment for credentials so they do not appear in the process list.

## Test the services

From another machine on the LAN:

```sh
upsc ups@KEENETIC_IP
curl http://KEENETIC_IP:8080/api/v1/status
```

The health endpoint returns HTTP 503 until the USB driver has supplied valid data and the eight-second startup suppression period has passed:

```sh
curl http://KEENETIC_IP:8080/healthz
```

## Synology

DSM is unusually strict when used as a secondary NUT client. Compatibility defaults are therefore:

- UPS name: `ups`;
- username: `monuser`;
- password: `secret`;
- server port: `3493`.

The password is only for the NUT monitor session and is unrelated to the router's SSH password. KeenUPS exposes no UPS power-control commands. Restrict ports `3493` and `8080` to trusted LAN interfaces and never forward them from the internet.

## HTTP endpoints

- `/` — auto-refreshing status table;
- `/api/v1/status` — complete JSON snapshot;
- `/healthz` — readiness status suitable for supervision.

## Entware startup

Copy the binary and init script, then make them executable:

```sh
chmod 755 /opt/bin/keenups
cp deploy/entware/S99keenups /opt/etc/init.d/S99keenups
chmod 755 /opt/etc/init.d/S99keenups
/opt/etc/init.d/S99keenups start
```

The example init script contains the DSM-compatible NUT password. Keep it readable only by root if the router has other shell users.
