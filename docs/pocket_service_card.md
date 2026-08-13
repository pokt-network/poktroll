# Pocket Service Card (v1)

**Moved.** Service cards and gateway cards are documented together in
**[`docs/pocket_cards.md`](./pocket_cards.md)**.

This file remains as a pointer because the path is referenced from proto comments and the
generated code and OpenAPI artifacts built from them (`x/shared/types/service.pb.go`,
`x/gateway/types/types.pb.go`, `docs/static/openapi.{json,yml}`). Those references move on the
next `make proto_regen`.

Direct links:

- [Service cards](./pocket_cards.md#service-cards) — fields, `rpc_types`, `serving`, worked examples
- [Gateway cards](./pocket_cards.md#gateway-cards) — fields, worked example, gateway-specific notes
- [Commands](./pocket_cards.md#commands) — validate, publish, read back, and the clearing caveat
- [What the chain does and does not do](./pocket_cards.md#what-the-chain-does-and-does-not-do)

Canonical schemas: `pkg/cards/service_card.schema.json`, `pkg/cards/gateway_card.schema.json`.
