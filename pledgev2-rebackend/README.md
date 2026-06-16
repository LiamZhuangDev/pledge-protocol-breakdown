# pledgev2 rebackend

This is a clean Go backend rebuild for learning how the original `pledge-backend`
works.

- API server: serve frontend/admin HTTP APIs, token list, pool data, login, websocket price push.

- Scheduler worker: read on-chain pool/oracle data and save snapshots into MySQL.

```mermaid
flowchart LR
  Contract[PledgePool + Oracle contracts] --> Scheduler[Go scheduler]
  Scheduler --> MySQL[(MySQL canonical pool/token data)]
  Scheduler --> Redis[(Redis cache/change markers)]
  KuCoin[KuCoin PLGR price] --> API[Go API server]
  Redis --> API
  MySQL --> API
  API --> Frontend[pledge-fe]
```

## Step 1: Runnable API Skeleton

Files:

- `cmd/api/main.go`
- `internal/config/config.go`
- `internal/httpserver/server.go`
- `internal/logging/logger.go`

Run:

```bash
cd pledgev2-rebackend
PLEDGE_ENV=local PLEDGE_API_PORT=8081 go run ./cmd/api
```

Then open:

```text
GET http://localhost:8081/healthz
```

Run Go Tests:

```bash
cd pledgev2-rebackend
# Run all Go tests in the current module, recursively.
go test ./...
```

Learning goal:

- Build the smallest runnable backend API process.
- Keep configuration loading separate from HTTP route setup.
- Create a health endpoint before adding database, Redis, or contract logic.
- Establish the future runtime path: `main -> config -> logger -> HTTP server`.

In interview, say:

```text
I started the backend rebuild with a small runnable API skeleton. The first
checkpoint proves the process can load config, start an HTTP server, and expose
a health check before adding storage or chain indexing.
```
## Step 2: Database models

Files:

- `internal/store/models.go`
- `internal/store/repository.go`
- `internal/store/memory.go`
- `internal/store/memory_test.go`

Run:

```bash
cd pledgev2-rebackend
go test ./...
```

Learning goal:

- Model the three core backend tables from the original project:
  `poolbases`, `pooldata`, and `token_info`.
- Keep chain values and token amounts as strings because contract values are
  large integer strings, not floats.
- Use `chainID + poolID` as the logical pool key.
- Use `chainID + token address` as the logical token key.
- Add a repository interface before adding MySQL so the API can depend on
  behavior instead of a concrete database driver.

For now, Step 2 uses an in-memory repository. MySQL comes later after the API
shape is clear.

In interview, say:

```text
I modeled the backend data around the same source-of-truth snapshots as the
original service: pool base config, pool settlement data, and token metadata.
The repository interface lets the API read pool data without caring whether the
storage is memory, MySQL, or a test double.
```

## Step 3: Read-only API

Files:

- `internal/httpserver/server.go`
- `internal/httpserver/server_test.go`
- `internal/chain/demo_reader.go`
- `internal/config/config.go`
- `cmd/api/main.go`

Run:

```bash
cd pledgev2-rebackend
PLEDGE_ENV=local PLEDGE_API_VERSION=1 PLEDGE_API_PORT=8081 go run ./cmd/api
```

Then query:

```bash
curl "http://localhost:8081/api/v1/poolBaseInfo?chainId=97"
curl "http://localhost:8081/api/v1/poolDataInfo?chainId=97"
curl "http://localhost:8081/api/v1/token?chainId=97"
```

Run Go Tests:

```bash
cd pledgev2-rebackend
go test ./...
```

Learning goal:

- Expose the first read-only API routes from the original backend.
- Keep route handlers thin: parse `chainId`, call the repository, return JSON.
- Serve data from the repository interface instead of hardcoding storage details
  into the HTTP layer.
- Use seeded memory data until the contract reader and MySQL store are added.

In interview, say:

```text
I added the read API as a thin layer over the repository. The handlers validate
request parameters, read pool/token snapshots from storage, and return JSON.
Because the API depends on an interface, the same routes can later read from
MySQL without changing handler behavior.
```

## Step 4: Contract reader

Files:

- `internal/chain/reader.go`
- `internal/chain/demo_reader.go`
- `internal/chain/sync.go`
- `internal/chain/sync_test.go`
- `cmd/api/main.go`
- `internal/config/config.go`

Run:

```bash
cd pledgev2-rebackend
PLEDGE_ENV=local PLEDGE_CHAIN_ID=97 PLEDGE_API_VERSION=1 PLEDGE_API_PORT=8081 go run ./cmd/api
```

Then query:

```bash
curl "http://localhost:8081/api/v1/poolBaseInfo?chainId=97"
curl "http://localhost:8081/api/v1/poolDataInfo?chainId=97"
curl "http://localhost:8081/api/v1/token?chainId=97"
```

Run Go Tests:

```bash
cd pledgev2-rebackend
go test ./...
```

Learning goal:

- Define the boundary between backend code and on-chain contract reads.
- Keep raw contract-shaped data separate from database/API models.
- Translate contract indexes into API pool IDs: contract index `0` becomes
  `poolID = 1`.
- Sync pool base data, pool settlement data, and token metadata into the
  repository through one function.

For now, Step 4 uses `DemoReader` instead of a real RPC client. The next real
reader can implement the same `chain.Reader` interface.

In interview, say:

```text
I separated contract reading from storage. The reader returns raw PledgePool
snapshots, and the sync function maps those snapshots into repository models.
That keeps RPC/ABI details out of the HTTP layer and gives the scheduler one
clear job: read contract state and persist the indexed snapshot.
```

## Step 5: Scheduler

## Step 6: Admin auth

## Step 7: Price service

## Step 8: Multisig/admin config API
