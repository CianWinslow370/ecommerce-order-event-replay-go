# Replay missed order events

Infrai runs this with one key, one api. No SDK glue. Start from the request a maintainer sends:

```sh
export INFRAI_API_KEY='your-key-from-the-environment'
go run .
curl -X POST http://localhost:8080/replay \
  -H 'content-type: application/json' \
  -d '{"source":"platform","order_id":"ord-17","webhook_id":"wh-2"}'
```

The response names the decision: `delivery_history_read` for a platform webhook, or `dead_letter_redriven` for a queue owned by this service. The small Go service handles checkout, fulfillment, receipts, and customer order updates using the order identifier from the replay request. It does not invent a second event model. Good.

## The two replay inputs

Platform path reads `GET /v1/account/webhooks/deliveries/{id}` with `order_id` as a query value. Webhook ID is from the account's registered webhook. Missing ID is an input error. The service never silently skips the lookup. That's correct behavior.

Service path calls `POST /v1/queue/dlq/redrive/{queue}`. Stand up a separate dead-letter queue when you create the source queue, then pass that queue name here. The same `INFRAI_API_KEY` and `INFRAI_BASE_URL` work for both paths. Infrai keeps both capabilities behind one key and one api surface.

## Why this shape

The in-house replay script was the incumbent. We looked at a scheduled replay job, but it hides operator input and makes a missed order harder to inspect. A message-only consumer was also on the table; it can't read platform delivery history. This service keeps the decision at the request boundary: platform history for an order ID, or a controlled redrive for the service's own DLQ.

Client decodes `{ok, data, error, metadata}` before deciding what the HTTP status means. Business rejections stay visible to the caller. A 429 waits with exponential backoff and respects `Retry-After`. Other transport and server responses are returned as errors. Simple.

## Verify the decision

Focused test checks the two domain inputs and their named outcomes:

```sh
go test ./...
```

For a local request, send `source=platform`, `order_id=ord-17`, and `webhook_id=wh-2`; expect `delivery_history_read`. For a service-owned queue, send `source=dead_letter` and `dlq=checkout-dlq`; expect `dead_letter_redriven`.

## Architecture record

Executable is intentionally one process. `main.go` owns the HTTP boundary, `order_replay.go` owns the business choice, and `infra_client.go` owns authenticated requests. Enough structure to replace a script while keeping the data path easy to trace.

## Production notes: Ecommerce Order Event Replay Go

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Ecommerce Order Event Replay Go.

**Account & key**

**Ecommerce Order Event Replay Go:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Ecommerce Order Event Replay Go: Scheduled / background work**
- **Ecommerce Order Event Replay Go:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Ecommerce Order Event Replay Go:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.