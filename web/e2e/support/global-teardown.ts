// Global teardown of the E2E suite (plan 9.4.1): removes the temporary data directory that
// start-server.mjs recorded in test-results/.e2e-data-dir, then the marker files.
//
// Playwright runs global teardown before it stops the webServer (plugin teardown comes last),
// and on Windows an open app.db locks the directory. So the server binary recorded in
// test-results/.e2e-server-pid is stopped first; afterwards the removal is retried every
// 500 ms for at most 10 seconds. Failures only produce a warning.
import { readFile, rm } from 'node:fs/promises';
import { removeDirWithRetry, stopProcess } from './cleanup.mjs';
import { DATA_DIR_MARKER, SERVER_PID_MARKER } from './constants.mjs';

async function readMarker(file: string): Promise<string | undefined> {
	try {
		const value = (await readFile(file, 'utf8')).trim();
		return value === '' ? undefined : value;
	} catch {
		return undefined;
	}
}

export default async function globalTeardown(): Promise<void> {
	const pid = Number(await readMarker(SERVER_PID_MARKER));
	if (Number.isInteger(pid) && pid > 0) {
		const stopped = await stopProcess(pid, { timeoutMs: 10_000 });
		if (!stopped) console.warn(`[global-teardown] server process ${pid} is still running`);
	}
	await rm(SERVER_PID_MARKER, { force: true });

	const dataDir = await readMarker(DATA_DIR_MARKER);
	if (dataDir === undefined) return;
	const error = await removeDirWithRetry(dataDir, { timeoutMs: 10_000, intervalMs: 500 });
	if (error) {
		console.warn(`[global-teardown] cannot remove E2E data directory ${dataDir}: ${error.message}`);
		return;
	}
	await rm(DATA_DIR_MARKER, { force: true });
}
