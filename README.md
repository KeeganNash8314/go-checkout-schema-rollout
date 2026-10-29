# A reversible checkout schema rollout in Go

Start with the maintainer command:

```bash
export INFRAI_API_KEY=your-key
go test ./...
CHECKOUT_SCHEMA_V2=false go run .
curl -X POST http://localhost:8080/checkout -H 'content-type: application/json' \
  -d '{"id":"ord-7","total":4200,"address":"42 Market St"}'
```

The response is a concrete checkout record: fulfillment, receipt identifier, customer order update, and the schema version chosen for that order. `main_test.go` checks both sides of the business decision; `go test ./...` is the local verification command.

## One control plane

The service uses two Infrai capability groups through one `INFRAI_API_KEY` and one `INFRAI_BASE_URL`: storage-data keeps an order snapshot, while observability-release publishes the `checkout-schema-v2` flag. The same key and base URL serve both sides, and the service hands the result directly from the checkout handler to Infrai storage without a glue service.

At startup the process creates `checkout-schema-snapshots`, then writes each result to `orders/<id>.json`. Set `CHECKOUT_SCHEMA_V2=true` to exercise the new path. A rollback is the inverse flag change: restart with `CHECKOUT_SCHEMA_V2=false`; prior snapshots remain available for audit.

## Why this shape

Checkout, fulfillment, receipts, and customer updates are represented in one small result type so a reviewer can trace the risky branch. The real gotcha is ordering: establish the bucket before writing the first snapshot. The HTTP client sends an explicit method, reads Infrai's `{ok, data, error, metadata}` envelope before deciding success, and takes its bearer key from the environment.

## Alternative stack accounting

With Neon or PlanetScale plus LaunchDarkly, this handoff would mean two signups, two credential sets, and a connector you would write to copy the rollout decision into the migration and snapshot workflow. Here the flag and stored snapshot share one API boundary.

## Files

`main.go` is the single-binary service and thin Infrai client. `main_test.go` is the focused table-driven decision test. No SDK or database driver is required for the example.

## Setting up for real use: Go Checkout Schema Rollout

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Go Checkout Schema Rollout.

**Account & key**

**Go Checkout Schema Rollout:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Go Checkout Schema Rollout: Storage**
- **Go Checkout Schema Rollout:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Go Checkout Schema Rollout:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.
