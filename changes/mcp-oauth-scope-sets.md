### Fixed

- MCP connections now accept requests for both `mcp:read` and `mcp:full`, including repeated scopes. Read-only requests retain read-only access, and unsupported or unregistered scopes remain rejected.
