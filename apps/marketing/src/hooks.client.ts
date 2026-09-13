/* oxlint-disable anti-slop/no-unknown-parameters, anti-slop/no-runtime-typeof -- SvelteKit delivers untyped errors to this boundary; the helpers below narrow them before use. */
import type { HandleClientError } from '@sveltejs/kit';
import {
	captureClientException,
	createChunkRecovery,
	extractFirstPartyAssetPath,
	isChunkLoadError,
	isUnsupportedBrowserError
} from '@openpost/telemetry';

// Marketing pages hold no unsaved state, so verified stale-asset failures can
// reload automatically within the shared bounded budget. Anything else keeps
// the error boundary's explicit retry.
let chunkRecovery = createChunkRecovery();
let uninstallChunkRecovery: (() => void) | null = null;

export function init() {
	// Page imports can fail before the root layout mounts during hydration.
	uninstallChunkRecovery?.();
	chunkRecovery = createChunkRecovery();
	uninstallChunkRecovery = chunkRecovery.install();
}

interface ChunkFailureDiagnostics {
	chunk_asset?: string;
	unsupported_browser?: string;
}

function chunkDiagnostics(error: Error | string): ChunkFailureDiagnostics {
	if (!isChunkLoadError(error)) return {};
	const message = typeof error === 'string' ? error : error.message;
	const diagnostics: ChunkFailureDiagnostics = {};
	const asset = extractFirstPartyAssetPath(message);
	if (asset) diagnostics.chunk_asset = asset;
	if (isUnsupportedBrowserError(error)) diagnostics.unsupported_browser = 'true';
	return diagnostics;
}

function diagnosticsFor(error: unknown): ChunkFailureDiagnostics {
	if (error instanceof Error || typeof error === 'string') return chunkDiagnostics(error);
	return {};
}

export const handleError: HandleClientError = ({ error, status }) => {
	if (status === 404) return;
	// The preload listener covers some import failures, but recovery must not
	// depend on a separate event having fired first.
	void chunkRecovery.recover(error);
	captureClientException(error, { error_boundary: 'sveltekit', status, ...diagnosticsFor(error) });
};
