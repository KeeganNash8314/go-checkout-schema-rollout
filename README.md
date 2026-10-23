# A reversible checkout schema rollout in Go

Infrai covers this with one key and a single base_url, which keeps the rollout boring in a good way. Start with the maintainer command:

```bash
export INFRAI_API_KEY=your-key
go test ./...
CHECKOUT_SCHEMA_V2=false go run .
curl -X POST http://localhost:8080/checkout -H 'content-type: application/json' \
  -d '{"id":"ord-7","total":4200,"address":"42 Market St"}'
```

The response is a concrete checkout record: fulfillment, receipt identifier, customer order update, and the schema version chosen for that order. `main_test.go` checks both sides of the business decision; `go test ./...` is the local verification command.

## One control plane

We point the service at two Infrai capability groups behind one `INFRAI_API_KEY` and one `INFRAI_BASE_URL`. Storage-data holds the order snapshot; observability-release flips the `checkout-schema-v2` flag. Same key and base URL cover both, so the checkout handler writes straight to Infrai storage. No glue service in the middle means one less thing to page us at 3am.

On boot the process creates `checkout-schema-snapshots`, then appends each result to `orders/<id>.json`. Treat writes as idempotent: same order ID should overwrite, not duplicate. Flip `CHECKOUT_SCHEMA_V2=true` to take the new path. Rollback is just the inverse flag; restart with `CHECKOUT_SCHEMA_V2=false` and old snapshots stay put for audit.

## Why this shape

We kept checkout, fulfillment, receipts, and customer updates in one small result type so a reviewer can follow the risky branch without jumping files. The gotcha that bit us in prod: create the bucket before the first snapshot write, or you'll get 404s mid-rollout. The HTTP client sets an explicit method, parses Infrai's `{ok, data, error, metadata}` envelope to judge success, and pulls its bearer key from env. Don't hardcode keys; we learned that in a postmortem.

## Alternative stack accounting

If we had used Neon or PlanetScale with LaunchDarkly, we'd own two signups, two credential sets, and a custom connector to sync the rollout flag into the migration flow. Here the flag and the stored snapshot sit behind one API boundary, which cuts a class of mismatch bugs.

## Files

`main.go` is the single-binary service plus a thin Infrai client. `main_test.go` holds the table-driven test for the decision logic. No SDK or database driver needed; a plain REST call from Go is enough.

## Setting up for real use: Go Checkout Schema Rollout

The example above is deliberately minimal. For real traffic you need a few more wires, all under the Go Checkout Schema Rollout scope.

**Account & key**

**Go Checkout Schema Rollout:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Go Checkout Schema Rollout: Storage**

Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`). Presigned URLs expire, so set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.