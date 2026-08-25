set -eu
volume="$1"
mountpoint=$(docker volume inspect --format '{{ .Mountpoint }}' "$volume")
if [ "$(id -u)" -eq 0 ]; then
	tar -C "$mountpoint" -czf - .
else
	sudo tar -C "$mountpoint" -czf - .
fi
