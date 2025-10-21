# Telegram Broadcast Bot

Send announcements to 18+ Telegram groups including exchanges and community channels.

## Quick Start

```bash
# Test first (single test group)
make telegram_test_broadcast_msg MSG_FILE=message.html

# Production (all 18 groups)
make telegram_broadcast_msg MSG_FILE=message.html
```

## Message Format

Use HTML for formatting:
- `<b>bold</b>`, `<i>italic</i>`, `<code>code</code>`
- `<a href="url">link</a>`, `<br>` for line breaks

**Example:**
```html
<b>🚀 Upgrade to v0.1.30</b>

<b>Height:</b> <code>123,456</code>
<b>Time:</b> Dec 15, 14:00 UTC

<b>Action Required:</b>
1. Pause deposits/withdrawals
2. Upgrade node
3. Resume operations

Docs: <a href="https://docs.pokt.network">docs.pokt.network</a>
```

## Targets

**Test mode:** Pocket Network Bot Testing Group only
**Production:** All 18 groups (BitGet, ByBit, CoinEx, Gate.io, Kraken, KuCoin, MEXC, OKX, etc.)

## Rules

1. **Always test first** - verify in test group before production
2. **No recalls** - messages cannot be undone
3. **Get approval** - coordinate with team for exchange broadcasts
