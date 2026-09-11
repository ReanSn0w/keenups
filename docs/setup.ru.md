# Настройка KeenUPS

[English version](setup.en.md)

KeenUPS работает на Keenetic с Entware, читает USB-ИБП через NUT `usbhid-ups` и публикует состояние в LAN: NUT protocol 1.3 на TCP `3493`, веб-страница и JSON API на TCP `8080`.

Эта конфигурация проверена с Powercom WOW-500U (`0d9f:0004`) и Keenetic ARM64. Другим HID-ИБП может потребоваться другая секция `ups.conf`.

## 1. Проверка Keenetic и ИБП

Нужны Entware/OPKG, SSH-доступ и `aarch64` для готового IPK. Проверьте на роутере:

```sh
uname -m
dmesg | grep -i -E 'usb|ups|hid'
```

Для WOW-500U в логе ожидаются `idVendor=0d9f`, `idProduct=0004` и `Product: HID UPS Battery`.

## 2. Установка

### Из IPK

Скачайте `keenups_<version>-1_aarch64-3.10.ipk` из GitHub Releases. На компьюте:

```sh
scp keenups_1.0.0-1_aarch64-3.10.ipk root@KEENETIC_IP:/tmp/keenups.ipk
```

На Keenetic:

```sh
opkg update
opkg install /tmp/keenups.ipk
```

OPKG также установит зависимость `nut-driver-usbhid-ups`. Пакет добавит `/opt/bin/keenups`, init-скрипт и `/opt/etc/nut/ups.conf.keenups.example`, не заменяя существующий `ups.conf`.

### Вручную

На Keenetic:

```sh
opkg update
opkg install nut-driver-usbhid-ups
```

На компьюте из корня репозитория:

```sh
make build-keenetic
scp dist/keenups-linux-arm64 root@KEENETIC_IP:/opt/bin/keenups
scp deploy/entware/S99keenups root@KEENETIC_IP:/opt/etc/init.d/S99keenups
scp configs/ups.conf.example root@KEENETIC_IP:/opt/etc/nut/ups.conf.keenups.example
```

На Keenetic:

```sh
chmod 755 /opt/bin/keenups /opt/etc/init.d/S99keenups
```

## 3. Настройка USB-ИБП

Если `ups.conf` ещё нет:

```sh
cp /opt/etc/nut/ups.conf.keenups.example /opt/etc/nut/ups.conf
chmod 600 /opt/etc/nut/ups.conf
```

Конфигурация WOW-500U:

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

`[keenups]` должно совпадать с `KEENUPS_DRIVER_NAME`. `pollonly` важен для WOW-500U: драйвер регулярно опрашивает устройство. Одновременно ИБП может захватить только один `usbhid-ups`; остановите ручной отладочный процесс.

## 4. Настройка сервиса

Изменяйте переменные в `/opt/etc/init.d/S99keenups`:

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `KEENUPS_LISTEN` | `:3493` | NUT-сервер |
| `KEENUPS_HTTP_LISTEN` | `:8080` | HTTP; пустое значение отключает его |
| `KEENUPS_DRIVER_BIN` | `/opt/lib/nut/usbhid-ups` | Драйвер NUT |
| `KEENUPS_DRIVER_SOCKET` | `/opt/var/run/usbhid-ups-keenups` | Unix-сокет драйвера |
| `KEENUPS_DRIVER_NAME` | `keenups` | Секция `ups.conf` |
| `KEENUPS_DRIVER_USER` | `root` | Пользователь драйвера |
| `KEENUPS_UPS_NAME` | `ups` | Сетевое имя ИБП |
| `KEENUPS_UPS_DESCRIPTION` | `Powercom WOW-500U via Keenetic` | Описание |
| `KEENUPS_USERNAME` | `monuser` | NUT-пользователь |
| `KEENUPS_PASSWORD` | `secret` | NUT-пароль |
| `KEENUPS_WARMUP` | `8s` | Отсечение ошибочных стартовых данных |

Все опции также видны в `/opt/bin/keenups --help`. Для Synology оставьте `ups`, `monuser`, `secret` и порт `3493`. Это NUT-пароль, не SSH-пароль Keenetic. Если клиенты поддерживают свои учётные данные, измените пароль и выполните `chmod 700 /opt/etc/init.d/S99keenups`.

## 5. Запуск и проверка

```sh
/opt/etc/init.d/S99keenups start
```

После изменений выполните `stop`, затем `start`. Проверка из LAN:

```sh
curl http://KEENETIC_IP:8080/healthz
curl http://KEENETIC_IP:8080/api/v1/status
upsc ups@KEENETIC_IP
```

Веб-страница: `http://KEENETIC_IP:8080/`. Без `upsc` можно выполнить:

```sh
printf 'GET VAR ups ups.status\n' | nc KEENETIC_IP 3493
```

Первые восемь секунд `/healthz` может возвращать 503: KeenUPS отсекает ложные стартовые показания WOW-500U.

## 6. Synology DSM

1. Откройте в DSM раздел управления питанием и ИБП.
2. Включите режим клиента сетевого UPS-сервера.
3. Укажите IP-адрес Keenetic.
4. Задайте задержку безопасного выключения NAS при работе от батареи.

После настройки зарядите батарею и ненадолго отключите входное питание ИБП: DSM должен увидеть переход на батарею. Не проводите полный тест автоотключения NAS, пока не проверена батарея.

## 7. Безопасность и диагностика

Разрешайте `3493` и `8080` только из доверенной LAN и не пробрасывайте их в интернет. HTTP API не имеет аутентификации. KeenUPS не предоставляет сетевым клиентам команды отключения нагрузки ИБП.

Отладка драйвера:

```sh
/opt/lib/nut/usbhid-ups -a keenups -u root -DD
ls -l /opt/var/run/usbhid-ups-keenups
```

Если имя секции изменено, обновите `KEENUPS_DRIVER_NAME` и `KEENUPS_DRIVER_SOCKET`. Если Synology не видит ИБП, проверьте с NAS доступ к `KEENETIC_IP:3493`, VLAN/firewall, имя `ups` и `monuser` / `secret`.

## 8. Обновление и удаление

```sh
/opt/etc/init.d/S99keenups stop
opkg install /tmp/keenups.ipk
/opt/etc/init.d/S99keenups start
```

Init-скрипт отмечен как конфигурационный файл, поэтому OPKG сохраняет локальные изменения. Удаление:

```sh
/opt/etc/init.d/S99keenups stop
opkg remove keenups
```
