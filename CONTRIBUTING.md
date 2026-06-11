# Contributing

Thanks for helping improve Unishare.

## Local Development

```sh
cp .env.example .env
make run
```

Open `http://127.0.0.1:8080` and sign in with one of the tokens from `.env`.

## Checks

```sh
make check
docker compose --env-file .env.example config
docker compose -f docker-compose.yml -f docker-compose.build.yml --env-file .env.example build
```

Keep changes small, preserve the no-build frontend setup, and avoid adding runtime dependencies unless they clearly improve the project.
