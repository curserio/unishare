# Unishare

Unishare is a lightweight self-hosted PWA for moving links, text, and files between devices, browsers, Android profiles, and isolated spaces.

It works like a personal dropbox: share something into Unishare, open Unishare somewhere else, then copy it or share it onward to another app. It is useful for Private Space, work profiles, a second phone, a home server, or a simple "send to myself" workflow.

## Features

- Accepts links, text, and files from the Android share sheet.
- Shows a shared list of recent items.
- Lets you copy or share an item onward to any app.
- Runs as one small Go process with no external database.
- Stores per-user metadata and files under `/data/users`.
- Supports English and Russian, with system language detection by default.
- Supports light, dark, and system themes.

## Quick Start With Docker

Create `.env`:

```sh
cp .env.example .env
```

Edit at least these values:

```dotenv
UNISHARE_USERS=main:a-long-secret-code,mom:another-long-secret-code
UNISHARE_PUBLIC_BASE_URL=https://share.example.com
GHCR_OWNER=OWNER
```

Start the app from GHCR:

```sh
docker compose --env-file .env up -d
```

The default image is:

```text
ghcr.io/${GHCR_OWNER}/unishare:${UNISHARE_IMAGE_TAG:-latest}
```

Put an HTTPS reverse proxy in front of the container. HTTPS is required for reliable PWA installation and Android share target support. Make sure the proxy forwards `X-Forwarded-Proto: https`.

If you need to serve Unishare under a path such as `https://sub.domain.com/unishare`, set:

```dotenv
UNISHARE_PUBLIC_BASE_URL=https://sub.domain.com
UNISHARE_BASE_PATH=/unishare
```

## Local Development

Run without Docker:

```sh
cp .env.example .env
make run
```

Open `http://127.0.0.1:8080` and sign in with one of the tokens from `.env`.

Run checks:

```sh
make check
```

Build a local binary:

```sh
make build
```

Build and run the local Docker image:

```sh
docker compose -f docker-compose.yml -f docker-compose.build.yml --env-file .env.example build
docker compose -f docker-compose.yml -f docker-compose.build.yml --env-file .env up -d
```

## Usage

1. Open the site on every device, browser, Android profile, or isolated space where you want to send or receive data.
2. Enter one of the tokens from `UNISHARE_USERS`.
3. Install the PWA to the home screen if your browser supports it.
4. To send content: `Share` -> `Unishare`.
5. To receive content elsewhere: open `Unishare` -> choose an item -> `Share` or `Copy`.

Language and theme are local settings. Each browser, profile, or Private Space can use its own language and appearance.

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `UNISHARE_USERS` | required | Comma-separated named tokens, for example `main:token1,mom:token2`. |
| `UNISHARE_PUBLIC_BASE_URL` | empty | Public HTTPS origin, used to build file links. Do not include `UNISHARE_BASE_PATH` here. |
| `UNISHARE_BASE_PATH` | empty | Optional path prefix, for example `/unishare`. |
| `UNISHARE_MAX_UPLOAD_MB` | `50` | Per-file upload limit in MB. |
| `UNISHARE_ADDR` | `:8080` | HTTP listen address. |
| `UNISHARE_DATA_DIR` | `/data` | Data directory. |
| `UNISHARE_STATIC_DIR` | `static` locally, `/app/static` in Docker | Static asset directory. |
| `UNISHARE_COOKIE_SECURE` | `auto` | `auto`, `true`, or `false`. Use `true` if your proxy setup hides HTTPS from the app. |
| `UNISHARE_HTTP_PORT` | `8080` | Host port used by Docker Compose. |
| `UNISHARE_IMAGE_TAG` | `latest` | Docker image tag used by Compose. |
| `GHCR_OWNER` | `OWNER` | GitHub owner or organization for `ghcr.io/<owner>/unishare`. |

## Users And Isolation

Unishare does not have registration or user management. Instead, the server owner defines named tokens:

```dotenv
UNISHARE_USERS=main:long-random-token-1,mom:long-random-token-2
```

Each name gets a separate private buffer. A session created with one token cannot list, delete, download, or share files from another token. Data is stored under:

```text
/data/users/<name>/items.json
/data/users/<name>/files
```

## Data And Backups

The Docker setup stores all data in the `unishare-data` volume:

- `/data/users/<name>/items.json` - metadata.
- `/data/users/<name>/files` - uploaded files.

Back up this volume if the buffer contains files you care about.

## Nginx Subpath Example

This example serves Unishare at `https://sub.domain.com/unishare` while the container listens on `127.0.0.1:8080`:

```nginx
location /unishare/ {
    proxy_pass http://127.0.0.1:8080/unishare/;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}

location = /unishare {
    return 301 /unishare/;
}
```

Use these env vars with that nginx config:

```dotenv
UNISHARE_PUBLIC_BASE_URL=https://sub.domain.com
UNISHARE_BASE_PATH=/unishare
```

## Architecture

- `internal/config` - environment configuration.
- `internal/model` - JSON models for shared items.
- `internal/store` - file-backed per-user storage under `/data/users`.
- `internal/httpapp` - HTTP routes, auth cookie, API handlers, health check, and static file serving.
- `static/js` - no-build ESM frontend: API client, i18n, theme manager, and UI rendering.

## Publishing Images

GitHub Actions builds and publishes images to GHCR:

- `main` pushes `ghcr.io/<owner>/<repo>:latest`.
- tags like `v1.0.0` push matching tag images.
- manual workflow dispatch is supported.

For this repository, the expected image name is:

```text
ghcr.io/<owner>/unishare:latest
```

## Limitations

- The browser must support Web Share Target. Chrome is the safest choice on Android.
- Unishare does not bypass Android profile or Private Space isolation directly. All clients communicate through the same server-side buffer.
- File links require an authenticated browser cookie, so you need to sign in once in each browser, profile, or space.

## Русский

Unishare - легковесная self-hosted PWA для обмена ссылками, текстом и файлами между устройствами, браузерами, Android-профилями и изолированными пространствами.

Быстрый запуск:

```sh
cp .env.example .env
# отредактируйте UNISHARE_USERS, UNISHARE_PUBLIC_BASE_URL, GHCR_OWNER
docker compose --env-file .env up -d
```

Сценарий простой: отправьте данные через системное меню `Поделиться` в Unishare, откройте Unishare в другом месте и передайте элемент дальше в нужное приложение. Это удобно для Private Space, рабочего профиля, второго телефона или сценария "отправить себе".

Интерфейс поддерживает русский и английский языки. По умолчанию используется язык системы, но его можно поменять в настройках приложения.
