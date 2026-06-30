# Task 4 Report

## What was done

I checked the repository from the point of view of a clean local setup and updated the project so the startup path is easier to repeat on a new machine.

The main repository-side changes in this branch are:

- removed the custom npm registry override from `frontend/Dockerfile`
- added a root `README.md` with exact local run and test steps
- added `docs/SETUP.md` with a clean-clone startup guide
- added a basic GitHub Actions workflow for `go build ./...` and `go test ./...`
- added a real Go unit test so CI does not run against an empty test suite

## Why the frontend Dockerfile was changed

The previous Dockerfile forced `npm` to use `https://registry.npmmirror.com`.
That can work, but it also adds a dependency that is not required for a normal clean machine setup.
For a repo that should build predictably on student laptops and CI runners, the default npm registry is the safer default.

## What was verified

I verified the repository structure and Compose configuration, and I checked the command compatibility on the current machine.

One important environment detail showed up immediately:

- on this machine, `docker compose` is not available
- `docker-compose` is the available command form

Because of that, the setup instructions now call out both command variants and tell the reader to use the one their machine actually supports.

I ran a real startup with:

```bash
docker-compose up --build -d
```

The first attempt failed during image pull because the machine was still behind the corporate proxy and Docker hit TLS certificate validation errors.
After that network constraint was removed, `docker-compose up --build -d` completed and the stack came up successfully.

## Commands used

Repository inspection:

```bash
git status --short --branch
ls -la
docker compose config
docker-compose version
```

Go verification:

```bash
go build ./...
go test ./...
```

## Result

The repository now contains a repeatable local setup path and the missing CI files needed for the next tasks.
Verified result on the target machine:

- `docker-compose config` passed
- `docker-compose up --build -d` passed after disabling the corporate proxy constraint
- all five services reached `Up`, with PostgreSQL and Redis marked `healthy`
- `GET /v1/analytics/stats` returned valid JSON
- `POST /v1/click` returned a success JSON response
- `go build ./...` passed
- `go test ./...` passed, including the real rate limiter test in `internal/engine`

The remaining part outside the repository is the public deployment from Task 6, which still depends on access to a real hosting target.
