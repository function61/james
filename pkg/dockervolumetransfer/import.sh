set -eu
volume="$1"
if docker volume inspect "$volume" >/dev/null 2>&1; then
	printf 'Docker volume already exists: %s\n' "$volume" >&2
	exit 1
fi

docker volume create "$volume" >/dev/null
cleanup() {
	docker volume rm "$volume" >/dev/null 2>&1 || true
}
trap cleanup EXIT

mountpoint=$(docker volume inspect --format '{{ .Mountpoint }}' "$volume")
if [ "$(id -u)" -eq 0 ]; then
	tar -C "$mountpoint" -xzf -
else
	sudo tar -C "$mountpoint" -xzf -
fi

trap - EXIT
