### Changed

- Console error reports preserve original exceptions and stack traces. Marketing telemetry includes the Cloudflare deployment revision.
- Publication Builder job reports retain safe failure categories, including invalid director, rendition, or reviewer output, without exposing private model text.
- Application, marketing, and documentation builds receive the server-only source-map upload credential through Turbo without putting it in frontend bundles or build hashes.
