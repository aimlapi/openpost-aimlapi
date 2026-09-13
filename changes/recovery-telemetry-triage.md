### Fixed

- Stale import recovery no longer reloads blindly. The app and marketing site share one bounded controller (`@openpost/telemetry` chunk recovery): one decision per failure across preload, boundary, and rejection events, a persisted per-asset and overall attempt budget with no time-window reset, and automatic reloads only with evidence (a missing first-party asset means deployment skew, a served asset means one transient retry). Offline, same-build, unclassified, and unpersistable-budget failures keep the error boundary's explicit retry. Rejected imports stay rejected for SvelteKit.
- Editors never lose unsaved work to an automatic import reload. While the layout reports unsaved changes, chunk failures fall back to the error boundary's explicit retry action.
- Safari below 16.4 no longer hits an unclassified theme startup failure on `/video-editor`. The pinned `@asamuzakjp/css-color` no longer uses regex lookbehind in its calc/var/relative-color detectors (Bun-pinned patch), and unsupported-engine failures are reported under `theme_unsupported_browser` with the CSS fallback instead of entering reload recovery.
- Telemetry keeps deployment context across identity resets and attaches surface, environment, edition, version, and revision to every exception capture, including buffered ones. Import failures retain the first-party asset pathname; blocked, offline, and recovery outcomes are classified instead of redacted away.
- Console-logged failures now reach error tracking. `console.error` output (failed workspace, draft, media, schedule, and account loads) is bridged into PostHog with a `console_error` boundary while preserving console output.
- Official production builds fail loudly when `POSTHOG_UI_HOST` is missing while source-map upload is enabled, instead of silently uploading symbols to the wrong region and leaving production errors unsymbolicated.
- The Video Editor animated-image subscription no longer feeds its own effect: cache callbacks run untracked so a warm-cache mount cannot invalidate the subscribing effect.

### Changed

- Documented the browser baseline (Safari 16.4 and equivalents) in `docs/development/frontend.md`.
