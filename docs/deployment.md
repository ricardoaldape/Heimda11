# Self-hosted deployment

## Docker Compose

1. Clone the repository.
2. Create `.env`.
3. Generate secrets:

```sh
openssl rand -hex 32
openssl rand -hex 32
```

Use one value for `HEIMDA11_MASTER_KEY` and use the other (or another long random value) for `HEIMDA11_ADMIN_TOKEN`.

4. Start:

```sh
docker compose up -d --build
```

5. Open `http://localhost:8080` on a trusted network.

For Internet exposure, terminate TLS at Caddy, Nginx, Traefik, Cloudflare Tunnel or your organization's ingress.

## Backup

The authoritative state lives at `/data/heimda11.json` in the container. Back up the Docker volume **and** store the Vault master key in a separate secure location.

A state backup without the master key cannot decrypt Vault data.

## Upgrade

```sh
git pull
docker compose up -d --build
```

Back up the data volume before an upgrade.

## Native binary

Heimda11 has no runtime dependencies besides trusted CA certificates for HTTPS Relay calls.

```sh
go build -o heimda11 ./cmd/heimda11
HEIMDA11_ADMIN_TOKEN=... \
HEIMDA11_MASTER_KEY=... \
./heimda11
```

## Community and Pro

Community is self-hosted and supports up to 10 registered AI workers. Pro is intended to remove the worker limit and add commercial support/enterprise distribution. The core runtime does not require customer data to leave the installation.
