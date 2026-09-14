### Improved

- Release preparation checks generated files, types, lint, tests, and browser flows before pushing. Local commits can be verified together and sent in one push.
- Frontend tests run alongside the canonical build in CI, and Android builds can reuse cached Gradle task outputs.
