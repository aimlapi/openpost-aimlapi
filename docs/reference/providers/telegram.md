---
description: Understand Telegram bot mode's operating boundary and requirements.
---

# Telegram bot mode

This page is for operators running Telegram bot mode.

Telegram bot mode is available in OpenPost. It connects channels and groups through an instance-owned bot and supports text and media publishing, observation from installation onward, and reaction counts.

## Operating requirements

Connect, publish, observation, and analytics are independent gates. Evidence for one operation never enables another.

The bot is instance-owned. Its token and webhook secret remain in encrypted operator configuration and must not be copied into workspace data, jobs, logs, connection links, or later API responses. A one-time `/connect` command is returned only through its authenticated issuance response and expires after 15 minutes.

## Implemented contract inventory

The connection paths can bind an eligible channel or supergroup, recheck destination identity and bot permissions, send text and media, preserve accepted message receipts, observe channel posts from installation onward, and record reaction counts.

Implemented publishing limits include 4,096 characters for a text message, 1,024 characters for a media caption, and up to 10 media items in one group. Caption overflow becomes a visible ordered follow-up. OpenPost does not invent historical coverage: observation begins at bot installation and does not backfill earlier messages.

## Operator boundary

Keep Telegram runtime controls enabled for production traffic. Reject webhook requests without the configured secret header, and never log raw update payloads or bot credentials.

Use the [provider application configuration guide](../configuration/provider-applications.md) for the private operator contract and the [launch matrix](../operations/provider-launch-matrix.md) for the evidence required before any public claim changes.
