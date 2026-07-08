# GitLab CI/CD

This repository can use GitLab for both CI and CD even if GitHub stays the source of truth.

The intended flow is simple:

1. GitHub stays the main development remote
2. GitLab mirrors the repository
3. GitLab runs CI on the mirrored branch
4. GitLab deploys `main` to the university VM over SSH

## What the pipeline does

The current `.gitlab-ci.yml` has four stages and five jobs:

1. `backend`
   Runs `go build ./...` and `go test ./... -race -count=1`
2. `frontend`
   Runs `npm ci`, `npm run lint`, and `npm run build`
   Uploads `frontend/dist/` as a GitLab artifact
3. `govulncheck`
   Runs `govulncheck ./...`
4. `integration`
   Builds the real Docker images with `docker compose up --build -d`
   Runs the smoke suite from `scripts/ci/compose_smoke.sh`
5. `deploy_vm`
   Connects to the VM by SSH
   Pulls the latest `main`
   Rebuilds the stack
   Reloads `nginx_engine`
   Runs remote smoke checks

The local Compose stack is now self-contained for CI. It no longer relies on bind-mounting `init-db.sql`, nginx config files, or GeoIP databases from the runner workspace into Docker-in-Docker.

## Required GitLab variables

Add these project variables in GitLab:

- `SSH_PRIVATE_KEY`
- `VM_HOST`
- `VM_PORT`
- `VM_USER`
- `DEPLOY_PATH`

Optional:

- `DEPLOY_BRANCH`

Recommended values:

- `VM_PORT=22`
- `VM_USER=root` or your deploy user
- `DEPLOY_PATH=/root/apps/anti-fraud-platform`
- `DEPLOY_BRANCH=main`

The repository on the VM should use the GitLab mirror as `origin`, because the deploy job runs `git fetch origin` and `git pull origin main` on the server side.

## Important note about SSH_PRIVATE_KEY

If GitLab refuses to save a masked multiline private key, use one of these options:

1. add `SSH_PRIVATE_KEY` as a `File` variable
2. add it as a regular hidden variable without `Masked`

The deploy script supports both forms:

- raw private key text
- path to a temporary file created by GitLab for a `File` variable

## What the remote deploy runs

After SSH login, GitLab runs:

```bash
cd /root/apps/anti-fraud-platform
git fetch origin
git checkout main
git pull --ff-only origin main
bash scripts/deploy/vm_refresh_stack.sh
bash scripts/deploy/vm_smoke.sh
```

`vm_refresh_stack.sh` rebuilds the stack with Docker Compose and reloads `nginx_engine` so a changed mounted config is applied without a manual SSH session.

`vm_smoke.sh` verifies:

- frontend responds on `localhost:3001`
- analytics responds on `localhost:8082`
- nginx-engine responds on `localhost:9090`
- `/v1/challenge` returns a real challenge payload
- a click without a solved challenge comes back as `flagged`

## Local dry run

If you want to verify the same logic locally before pushing:

```bash
make ci-compose-up
make ci-compose-smoke
make ci-compose-down
```
