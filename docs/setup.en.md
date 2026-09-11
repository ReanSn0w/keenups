# KeenUPS setup

[Русская версия](setup.ru.md)

KeenUPS runs on a Keenetic router with Entware, reads a USB UPS through NUT's `usbhid-ups` driver, and publishes its state to the LAN: NUT protocol 1.3 on TCP `3493`, plus a web page and JSON API on TCP `8080`.

This configuration has been tested with the Powercom WOW-500U (`0d9f:0004`) and an ARM64 Keenetic router. Other HID UPS models may require a different `ups.conf` section.

## 1. Check the router and UPS

Entware/OPKG, SSH access, and the `aarch64` architecture are required for the prebuilt IPK. Check on the router:

```sh
uname -m
dmesg | grep -i -E 'usb|ups|hid'
```

For the WOW-500U, expect `idVendor=0d9f`, `idProduct=0004`, and `Product: HID UPS Battery`.

## 2. Installation

### From an IPK

Download `keenups_<version>-1_aarch64-3.10.ipk` from GitHub Releases. On your computer:

```sh
scp keenups_1.0.0-1_aarch64-3.10.ipk root@KEENETIC_IP:/tmp/keenups.ipk
```

On the Keenetic router:

```sh
opkg update
opkg install /tmp/keenups.ipk
```

OPKG also installs the `nut-driver-usbhid-ups` dependency. The package adds `/opt/bin/keenups`, the init script, and `/opt/etc/nut/ups.conf.keenups.example` without replacing an existing `ups.conf`.

### Manual installation

On the Keenetic router:

```sh
opkg update
opkg install nut-driver-usbhid-ups
```

On your computer, from the repository root:

```sh
make build-keenetic
scp dist/keenups-linux-arm64 root@KEENETIC_IP:/opt/bin/keenups
scp deploy/entware/S99keenups root@KEENETIC_IP:/opt/etc/init.d/S99keenups
scp configs/ups.conf.example root@KEENETIC_IP:/opt/etc/nut/ups.conf.keenups.example
```

On the Keenetic router:

```sh
chmod 755 /opt/bin/keenups /opt/etc/init.d/S99keenups
```

## 3. Configure the USB UPS

If `ups.conf` does not exist yet:

```sh
cp /opt/etc/nut/ups.conf.keenups.example /opt/etc/nut/ups.conf
chmod 600 /opt/etc/nut/ups.conf
```

WOW-500U configuration:

```ini
[keenups]
    driver = usbhid-ups
    port = auto
    vendorid = 0d9f
    productid = 0004
    pollonly
    pollinterval = 2
    pollfreq = 5
    desc = "Powercom WOW-500U"
```

`[keenups]` must match `KEENUPS_DRIVER_NAME`. `pollonly` is important for the WOW-500U because it makes the driver poll the device regularly. Only one `usbhid-ups` process can claim the UPS; stop any manually started debug process first.

## 4. Configure the service

Edit environment variables in `/opt/etc/init.d/S99keenups`:

| Variable | Default | Purpose |
|---|---|---|
| `KEENUPS_LISTEN` | `:3493` | NUT server |
| `KEENUPS_HTTP_LISTEN` | `:8080` | HTTP; an empty value disables it |
| `KEENUPS_DRIVER_BIN` | `/opt/lib/nut/usbhid-ups` | NUT driver |
| `KEENUPS_DRIVER_SOCKET` | `/opt/var/run/usbhid-ups-keenups` | Driver Unix socket |
| `KEENUPS_DRIVER_NAME` | `keenups` | `ups.conf` section |
| `KEENUPS_DRIVER_USER` | `root` | Driver user |
| `KEENUPS_UPS_NAME` | `ups` | Network-visible UPS name |
| `KEENUPS_UPS_DESCRIPTION` | `Powercom WOW-500U via Keenetic` | Description |
| `KEENUPS_USERNAME` | `monuser` | NUT username |
| `KEENUPS_PASSWORD` | `secret` | NUT password |
| `KEENUPS_WARMUP` | `8s` | Invalid startup reading suppression |

All options are also shown by `/opt/bin/keenups --help`. For Synology, keep `ups`, `monuser`, `secret`, and port `3493`. This is a NUT password, not the Keenetic SSH password. If clients support custom credentials, change it and run `chmod 700 /opt/etc/init.d/S99keenups`.

## 5. Start and verify

```sh
/opt/etc/init.d/S99keenups start
```

After configuration changes, run `stop` and then `start`. Verify from the LAN:

```sh
curl http://KEENETIC_IP:8080/healthz
curl http://KEENETIC_IP:8080/api/v1/status
upsc ups@KEENETIC_IP
```

The web page is at `http://KEENETIC_IP:8080/`. Without `upsc`, use:

```sh
printf 'GET VAR ups ups.status\n' | nc KEENETIC_IP 3493
```

For the first eight seconds, `/healthz` may return 503 while KeenUPS suppresses invalid WOW-500U startup readings.

## 6. Synology DSM

1. Open the UPS/power management section in DSM.
2. Enable network UPS server client mode.
3. Enter the Keenetic router IP address.
4. Configure the safe-shutdown delay for battery operation.

After setup, charge the battery and briefly disconnect UPS input power. DSM should report battery operation. Do not test a complete NAS shutdown until the battery condition has been verified. KeenUPS does not expose UPS load-control commands to network clients.

## 7. Security and troubleshooting

Allow `3493` and `8080` only from the trusted LAN and never forward them from the internet. The HTTP API has no authentication.

Driver diagnostics:

```sh
/opt/lib/nut/usbhid-ups -a keenups -u root -DD
ls -l /opt/var/run/usbhid-ups-keenups
```

Stop the debug driver before restarting KeenUPS. If the section name changes, update both `KEENUPS_DRIVER_NAME` and `KEENUPS_DRIVER_SOCKET`. If Synology cannot connect, check access to `KEENETIC_IP:3493`, VLAN/firewall rules, UPS name `ups`, and `monuser` / `secret`.

## 8. Update and removal

```sh
/opt/etc/init.d/S99keenups stop
opkg install /tmp/keenups.ipk
/opt/etc/init.d/S99keenups start
```

OPKG preserves local changes to the init script because it is marked as a configuration file. To remove KeenUPS:

```sh
/opt/etc/init.d/S99keenups stop
opkg remove keenups
```
