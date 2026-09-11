#!/bin/sh

set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 3 ]; then
	echo "usage: $0 vMAJOR.MINOR.PATCH [binary] [output-directory]" >&2
	exit 2
fi

tag_version=$1
version=${tag_version#v}
binary=${2:-dist/keenups-linux-arm64}
output_dir=${3:-dist}

if ! printf '%s\n' "$tag_version" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
	echo "invalid version '$tag_version': expected vMAJOR.MINOR.PATCH" >&2
	exit 2
fi

if [ ! -f "$binary" ]; then
	echo "binary not found: $binary" >&2
	exit 1
fi

if ! tar --version 2>/dev/null | grep -q 'GNU tar'; then
	echo "GNU tar is required to build a reproducible IPK package" >&2
	exit 1
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
project_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
build_dir=$(mktemp -d "${TMPDIR:-/tmp}/keenups-ipk.XXXXXX")
trap 'rm -rf "$build_dir"' EXIT HUP INT TERM

data_dir=$build_dir/data
control_dir=$build_dir/control
package_dir=$build_dir/package
mkdir -p "$data_dir/opt/bin" "$data_dir/opt/etc/init.d" "$data_dir/opt/etc/nut" "$control_dir" "$package_dir" "$output_dir"

cp "$binary" "$data_dir/opt/bin/keenups"
cp "$project_dir/deploy/entware/S99keenups" "$data_dir/opt/etc/init.d/S99keenups"
cp "$project_dir/configs/ups.conf.example" "$data_dir/opt/etc/nut/ups.conf.keenups.example"
chmod 0755 "$data_dir/opt/bin/keenups" "$data_dir/opt/etc/init.d/S99keenups"
chmod 0644 "$data_dir/opt/etc/nut/ups.conf.keenups.example"

installed_size=$(tar -cf - -C "$data_dir" . | wc -c | tr -d ' ')
package_version=$version-1

cat >"$control_dir/control" <<EOF
Package: keenups
Version: $package_version
Depends: nut-driver-usbhid-ups
Architecture: aarch64-3.10
Maintainer: Dmitriy Papkov <papkovda@me.com>
Section: utils
Priority: optional
License: MIT
Installed-Size: $installed_size
Description: Network UPS bridge for Keenetic routers
 Supervises the NUT USB HID driver and publishes UPS state over NUT and HTTP.
EOF

printf '%s\n' '/opt/etc/init.d/S99keenups' >"$control_dir/conffiles"
printf '2.0\n' >"$package_dir/debian-binary"

archive_time=${SOURCE_DATE_EPOCH:-$(git -C "$project_dir" log -1 --format=%ct 2>/dev/null || printf '0')}
tar_options="--format=gnu --numeric-owner --owner=0 --group=0 --sort=name --mtime=@$archive_time"

# Word splitting is intentional: tar_options contains individual GNU tar arguments.
# shellcheck disable=SC2086
tar $tar_options -czf "$package_dir/data.tar.gz" -C "$data_dir" .
# shellcheck disable=SC2086
tar $tar_options -czf "$package_dir/control.tar.gz" -C "$control_dir" .

package_file=$output_dir/keenups_${package_version}_aarch64-3.10.ipk
# shellcheck disable=SC2086
tar $tar_options -czf "$package_file" -C "$package_dir" ./debian-binary ./data.tar.gz ./control.tar.gz

echo "$package_file"
