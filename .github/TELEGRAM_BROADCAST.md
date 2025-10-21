# Telegram Broadcast Bot

This document describes how to trigger the Telegram broadcast bot for sending announcements to Pocket Network stakeholders.

## Overview

The Telegram bot is a notification system used to broadcast important announcements (e.g., network upgrades, releases) to multiple Telegram groups simultaneously, including exchange partners and community channels.

## Bot Modes

The bot operates in two distinct modes:

### 1. Test Mode
Sends messages **only** to the testing group for validation before production broadcasts.

- **Target**: Pocket Network Bot Testing Group (single group)
- **Purpose**: Verify message formatting and content before full broadcast
- **Safety**: No risk of incorrect announcements to external partners

### 2. Full Mode (Production)
Sends messages to **all configured groups** including exchange partners and community channels.

- **Targets**: 18 Telegram groups including:
  - Exchange partner channels (BitGet, ByBit, CoinEx, Gate.io, Kraken, KuCoin, etc.)
  - Community channels (The Poktopus Den)
  - Internal testing group
- **Purpose**: Official announcements and critical updates
- **Caution**: Messages reach external stakeholders immediately

## Triggering the Bot

### Method 1: Using Makefile Commands (Recommended)

#### Test Broadcast
```bash
make telegram_test_broadcast_msg MSG_FILE=path/to/message.txt
```

**Example:**
```bash
make telegram_test_broadcast_msg MSG_FILE=announcements/upgrade_notice.txt
```

#### Production Broadcast
```bash
make telegram_broadcast_msg MSG_FILE=path/to/message.txt
```

**Example:**
```bash
make telegram_broadcast_msg MSG_FILE=announcements/mainnet_release.html
```

### Method 2: Using GitHub Actions Workflow

#### Via GitHub Web UI
1. Navigate to **Actions** tab in the repository
2. Select **"Telegram Broadcast"** workflow
3. Click **"Run workflow"** button
4. Fill in the form:
   - **Message**: Enter your announcement text (supports HTML formatting)
   - **Test mode**: Check the box to send only to testing group
5. Click **"Run workflow"** to trigger

#### Via GitHub CLI
```bash
# Test broadcast
gh workflow run telegram-broadcast.yml \
  -f message="<b>Test Announcement</b><br>This is a test message." \
  -f test_mode=true

# Production broadcast
gh workflow run telegram-broadcast.yml \
  -f message="<b>Official Announcement</b><br>Network upgrade scheduled for..." \
  -f test_mode=false
```

## Message Formatting

Messages support **HTML formatting** for rich text:

### Supported HTML Tags
- `<b>text</b>` - **Bold text**
- `<i>text</i>` - *Italic text*
- `<u>text</u>` - Underlined text
- `<code>text</code>` - Monospace code
- `<pre>text</pre>` - Preformatted code block
- `<a href="url">text</a>` - Hyperlinks
- `<br>` - Line break

## Sample Messages

Below are sample messages for different broadcast scenarios. You can save these to files and use them as templates.

### 1. Test Message (For Testing Bot Functionality)

Save this as `test-message.html`:

```html
<b>🧪 TEST MESSAGE - Please Ignore</b>

This is a <b>test broadcast</b> from the Pocket Network Telegram bot.

<b>Purpose:</b> Verifying message formatting and delivery

<b>Formatting Test:</b>
• <b>Bold text</b> ✓
• <i>Italic text</i> ✓
• <code>Monospace code</code> ✓
• <a href="https://pokt.network">Hyperlink</a> ✓

<b>Status:</b> Testing in progress
<b>Action Required:</b> None - this is a test message

If you received this message, the bot is functioning correctly.

---
<i>Sent via Telegram Broadcast Bot (Test Mode)</i>
```

**How to use:**
```bash
make telegram_test_broadcast_msg MSG_FILE=test-message.html
```

---

### 2. MainNet Upgrade - Pre-Announcement (Production)

Save this as `mainnet-upgrade-pre.html`:

```html
<b>🚀 Poktroll MainNet Upgrade Announcement</b>

Dear Pocket Network Partners,

A <b>mandatory network upgrade</b> is scheduled for the Pocket Network blockchain.

<b>📋 Upgrade Details:</b>
<b>• Version:</b> <code>v0.1.30</code>
<b>• Upgrade Height:</b> <code>123,456</code>
<b>• Estimated Time:</b> December 15, 2024 at 14:00 UTC
<b>• Expected Duration:</b> ~30 minutes

<b>⚠️ Action Required:</b>

For <b>Exchange Partners</b> and <b>Node Operators</b>:
1. <b>Pause deposits/withdrawals</b> 1 hour before upgrade time
2. <b>Stop your node</b> before the upgrade height
3. <b>Update to new binary</b> (<code>v0.1.30</code>)
4. <b>Restart node</b> after upgrade completion
5. <b>Resume operations</b> once chain is stable

<b>📚 Resources:</b>
• Upgrade Guide: <a href="https://docs.pokt.network/protocol/upgrades">docs.pokt.network/protocol/upgrades</a>
• Binary Release: <a href="https://github.com/pokt-network/poktroll/releases/tag/v0.1.30">GitHub Releases</a>
• Support Channel: <a href="https://discord.gg/pokt">Discord</a>

<b>🔔 Timeline:</b>
• <b>Now:</b> Announcement and preparation period
• <b>Dec 15, 13:00 UTC:</b> Pause deposits/withdrawals
• <b>Dec 15, 14:00 UTC:</b> Upgrade executes
• <b>Dec 15, 14:30 UTC:</b> Expected completion

<b>❓ Questions or Issues?</b>
Contact the Pocket Network team immediately via Discord or email: support@pokt.network

We will send a follow-up message once the upgrade is successfully completed.

Thank you for your continued partnership!

---
<i>Pocket Network Team</i>
```

**How to use:**
```bash
# Test first
make telegram_test_broadcast_msg MSG_FILE=mainnet-upgrade-pre.html

# After verification, broadcast to production
make telegram_broadcast_msg MSG_FILE=mainnet-upgrade-pre.html
```

---

### 3. MainNet Upgrade - Post-Announcement (Production)

Save this as `mainnet-upgrade-post.html`:

```html
<b>✅ Poktroll MainNet Upgrade Complete</b>

Dear Pocket Network Partners,

The <b>MainNet upgrade has been successfully completed!</b>

<b>📊 Upgrade Summary:</b>
<b>• Version:</b> <code>v0.1.30</code>
<b>• Upgrade Height:</b> <code>123,456</code>
<b>• Completion Time:</b> December 15, 2024 at 14:25 UTC
<b>• Network Status:</b> <b>OPERATIONAL</b> ✓

<b>✅ Next Steps:</b>

For <b>Exchange Partners</b>:
1. <b>Verify your node</b> is running the new version (<code>v0.1.30</code>)
2. <b>Confirm synchronization</b> with the network
3. <b>Resume deposits/withdrawals</b> when ready
4. <b>Monitor operations</b> for the next 24 hours

For <b>Node Operators</b>:
1. <b>Ensure successful upgrade</b> by checking node logs
2. <b>Verify block production</b> is continuing normally
3. <b>Report any issues</b> immediately to the team

<b>🔍 Verification Commands:</b>
<pre>
# Check version
pocketd version

# Check sync status
pocketd status | jq .SyncInfo
</pre>

<b>📈 What's New in v0.1.30:</b>
• Enhanced relay mining efficiency
• Improved tokenomics calculations
• Security patches and stability improvements
• Performance optimizations

Full release notes: <a href="https://github.com/pokt-network/poktroll/releases/tag/v0.1.30">GitHub Releases</a>

<b>📞 Support:</b>
If you experience any issues or have questions:
• Discord: <a href="https://discord.gg/pokt">discord.gg/pokt</a>
• Email: support@pokt.network
• Documentation: <a href="https://docs.pokt.network">docs.pokt.network</a>

Thank you for your cooperation during this upgrade!

---
<i>Pocket Network Team</i>
```

**How to use:**
```bash
# Test first
make telegram_test_broadcast_msg MSG_FILE=mainnet-upgrade-post.html

# After verification, broadcast to production
make telegram_broadcast_msg MSG_FILE=mainnet-upgrade-post.html
```

---

### 4. Emergency Maintenance (Production - Use with Caution)

Save this as `emergency-maintenance.html`:

```html
<b>🚨 URGENT: Emergency Maintenance Notice</b>

Dear Pocket Network Partners,

We have detected a <b>critical issue</b> that requires immediate network maintenance.

<b>⚠️ IMMEDIATE ACTION REQUIRED:</b>

<b>All Node Operators:</b>
1. <b>STOP your nodes immediately</b>
2. <b>Do NOT restart</b> until further notice
3. <b>Await instructions</b> in this channel

<b>Exchange Partners:</b>
1. <b>PAUSE all deposits and withdrawals</b>
2. <b>Do NOT process any POKT transactions</b>
3. <b>Wait for all-clear notification</b>

<b>📋 Issue Details:</b>
<b>• Severity:</b> Critical
<b>• Impact:</b> Network-wide
<b>• Status:</b> Under investigation
<b>• ETA for Resolution:</b> TBD (updates every 30 minutes)

<b>🔄 Next Steps:</b>
• Our team is actively working on a fix
• A patched binary will be released ASAP
• Detailed instructions will follow shortly
• We will provide regular updates in this channel

<b>📞 Emergency Contact:</b>
• Discord: <a href="https://discord.gg/pokt">discord.gg/pokt</a> (fastest response)
• Email: emergency@pokt.network
• Status Page: <a href="https://status.pokt.network">status.pokt.network</a>

<b>Please acknowledge receipt of this message</b> in your respective channels.

We apologize for the inconvenience and appreciate your immediate cooperation.

---
<i>Pocket Network Team</i>
<i>Sent: December 15, 2024 - 10:00 UTC</i>
```

**How to use:**
```bash
# In true emergencies, you may skip testing, but verify the message carefully first
make telegram_broadcast_msg MSG_FILE=emergency-maintenance.html
```

**⚠️ Warning:** Only use for genuine emergencies. Overuse will reduce partner responsiveness.

---

### 5. Routine Update (Production)

Save this as `routine-update.html`:

```html
<b>📢 Pocket Network - Routine Update</b>

Hello Pocket Network Partners,

This is a <b>routine update</b> on the Pocket Network ecosystem.

<b>🌐 Network Status:</b>
<b>• Current Version:</b> <code>v0.1.29</code>
<b>• Block Height:</b> ~120,000
<b>• Network Health:</b> Excellent ✓
<b>• Active Suppliers:</b> 150+
<b>• Daily Relays:</b> 50M+

<b>📅 Upcoming Events:</b>
• <b>Next Upgrade:</b> Q1 2025 (details TBA)
• <b>Community Call:</b> January 10, 2025
• <b>Documentation Updates:</b> Ongoing

<b>💡 Recent Improvements:</b>
• Enhanced monitoring dashboards
• Updated API documentation
• Improved relay efficiency
• Better error handling and logging

<b>🔗 Useful Links:</b>
• Documentation: <a href="https://docs.pokt.network">docs.pokt.network</a>
• Block Explorer: <a href="https://explorer.pokt.network">explorer.pokt.network</a>
• GitHub: <a href="https://github.com/pokt-network/poktroll">github.com/pokt-network/poktroll</a>
• Status Page: <a href="https://status.pokt.network">status.pokt.network</a>

<b>📞 Stay Connected:</b>
• Discord: <a href="https://discord.gg/pokt">discord.gg/pokt</a>
• Twitter: <a href="https://twitter.com/poktnetwork">@POKTnetwork</a>
• Forum: <a href="https://forum.pokt.network">forum.pokt.network</a>

<b>Action Required:</b> None - informational only

If you have any questions or feedback, please don't hesitate to reach out.

Thank you for being part of the Pocket Network ecosystem!

---
<i>Pocket Network Team</i>
```

**How to use:**
```bash
# Test first
make telegram_test_broadcast_msg MSG_FILE=routine-update.html

# After verification, broadcast to production
make telegram_broadcast_msg MSG_FILE=routine-update.html
```

## Best Practices

### Before Broadcasting

1. **Always test first**: Run test mode to verify formatting and content
   ```bash
   make telegram_test_broadcast_msg MSG_FILE=announcement.html
   ```

2. **Review in testing group**: Check the message appearance in the test channel

3. **Verify links and formatting**: Ensure all URLs work and HTML renders correctly

4. **Get team approval**: Have announcements reviewed before production broadcast

### Production Broadcasting

1. **Double-check message file**: Ensure you're using the correct, approved message file

2. **Verify mode**: Make sure you're NOT in test mode for production broadcasts

3. **Send once**: Avoid duplicate broadcasts - messages cannot be recalled

4. **Monitor delivery**: Check a few target groups to confirm successful delivery

### Safety Guidelines

- **Test mode is mandatory** for all new message formats
- **Never broadcast untested content** to production groups
- **Coordinate with team** before sending to exchange partners
- **Time broadcasts appropriately** considering global time zones
- **Keep messages professional** - they represent Pocket Network to partners

## Workflow Configuration

### Configured Groups (18 Total)

| Group ID | Name | Type |
|----------|------|------|
| CHAT_1 | Pocket Network Bot Testing Group | Testing |
| CHAT_2 | BitGet <> Pocket Network (POKT) | Exchange |
| CHAT_3 | ByBit <> POKT | Exchange |
| CHAT_4 | CoinEx <> POKT | Exchange |
| CHAT_5 | Mr. Yang <> Grove / POKT | Partner |
| CHAT_6 | MEXC <> POKT | Exchange |
| CHAT_7 | OKX <> Pocket Network | Exchange |
| CHAT_8 | OrangeX <> Grove (POKT) | Exchange |
| CHAT_9 | POKT & Gate.io | Exchange |
| CHAT_10 | POKT Network \| AscendEX | Exchange |
| CHAT_11 | Uphold <> Grove (POKT) | Exchange |
| CHAT_12 | Pokt <> Kraken | Exchange |
| CHAT_13 | Bitrue & Pocket Network | Exchange |
| CHAT_14 | HTX <> POKT | Exchange |
| CHAT_15 | POKT <> Upbit | Exchange |
| CHAT_16 | KuCoin <> POKT | Exchange |
| CHAT_17 | Korbit <> POKT | Exchange |
| CHAT_18 | The Poktopus Den | Community |

### Implementation Files

- **Workflow**: `.github/workflows/telegram-broadcast.yml` - Manual trigger interface
- **Core Logic**: `.github/workflows/telegram-send-message.yml` - Message sending implementation
- **Makefile**: `makefiles/telegram.mk` - CLI commands for broadcasting

## Troubleshooting

### Message not delivered
- **Check workflow logs**: View GitHub Actions logs for error messages
- **Verify message format**: Ensure HTML is valid (unclosed tags cause failures)
- **File path**: Confirm `MSG_FILE` path is correct and file exists

### HTML not rendering
- **Use supported tags only**: Refer to the formatting section above
- **Escape special characters**: Use HTML entities for `<`, `>`, `&` in content
- **Test mode first**: Always verify rendering in test group before production

### Wrong groups received message
- **Check test mode flag**: Verify `test_mode` parameter value
- **Review workflow run**: Check GitHub Actions run to see which workflow executed

## Additional Resources

- **MainNet Release Procedure**: `docusaurus/docs/4_develop/upgrades/3_mainnet_release_procedure.md`
- **Workflow Source**: `.github/workflows/telegram-broadcast.yml`
- **Makefile Commands**: `makefiles/telegram.mk`

## Support

For issues or questions about the Telegram bot:
1. Review this documentation and troubleshooting section
2. Check GitHub Actions workflow logs for error details
3. Contact the Pocket Network development team
4. Report persistent issues in the repository issue tracker
