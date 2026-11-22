// Cleanup helpers shared by start-server.mjs and global-teardown.ts (plan 9.4.1, steps 5-6).
// On Windows an open app.db keeps its directory locked until the server process has exited,
// so every removal retries for a while and only logs a final failure.
import { readdir, rm, stat } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { setTimeout as sleep } from 'node:timers/promises';
import { DATA_DIR_PREFIX, STALE_DATA_DIR_MS } from './constants.mjs';

/**
 * Removes a directory tree, retrying while the files are still locked.
 * @param {string} dir
 * @param {{ timeoutMs?: number, intervalMs?: number }} [options]
 * @returns {Promise<Error | undefined>} the last error when the directory could not be removed
 */
export async function removeDirWithRetry(dir, options = {}) {
	const timeoutMs = options.timeoutMs ?? 10_000;
	const intervalMs = options.intervalMs ?? 500;
	const deadline = Date.now() + timeoutMs;
	for (;;) {
		try {
			await rm(dir, { recursive: true, force: true });
			return undefined;
		} catch (err) {
			if (Date.now() + intervalMs > deadline) {
				return err instanceof Error ? err : new Error(String(err));
			}
			await sleep(intervalMs);
		}
	}
}

/**
 * Removes the data directories of earlier runs (kinex-mark-e2e-*) that are older than the
 * limit: leftovers of runs whose server was killed before it could clean up.
 * @param {{ tmpDir?: string, now?: number, maxAgeMs?: number, log?: (message: string) => void }} [options]
 * @returns {Promise<{ removed: string[], failed: string[] }>}
 */
export async function removeStaleDataDirs(options = {}) {
	const tmpDir = options.tmpDir ?? os.tmpdir();
	const now = options.now ?? Date.now();
	const maxAgeMs = options.maxAgeMs ?? STALE_DATA_DIR_MS;
	const log = options.log ?? (() => {});
	/** @type {string[]} */
	const removed = [];
	/** @type {string[]} */
	const failed = [];
	let entries;
	try {
		entries = await readdir(tmpDir, { withFileTypes: true });
	} catch (err) {
		log(`cannot list ${tmpDir}: ${err instanceof Error ? err.message : String(err)}`);
		return { removed, failed };
	}
	for (const entry of entries) {
		if (!entry.isDirectory() || !entry.name.startsWith(DATA_DIR_PREFIX)) continue;
		const dir = path.join(tmpDir, entry.name);
		try {
			const info = await stat(dir);
			if (now - info.mtimeMs <= maxAgeMs) continue;
			await rm(dir, { recursive: true, force: true });
			removed.push(dir);
		} catch (err) {
			failed.push(dir);
			log(
				`cannot remove stale data directory ${dir}: ${err instanceof Error ? err.message : String(err)}`
			);
		}
	}
	return { removed, failed };
}

/**
 * Reports whether a process exists. EPERM means it exists but belongs to someone else.
 * @param {number} pid
 */
export function isProcessAlive(pid) {
	try {
		process.kill(pid, 0);
		return true;
	} catch (err) {
		return /** @type {NodeJS.ErrnoException} */ (err).code === 'EPERM';
	}
}

/**
 * Stops a process and waits until it is gone. SIGTERM lets the server shut down gracefully on
 * Linux and macOS; on Windows Node terminates the process immediately. After the timeout the
 * process is killed.
 * @param {number} pid
 * @param {{ timeoutMs?: number }} [options]
 * @returns {Promise<boolean>} true when the process no longer exists
 */
export async function stopProcess(pid, options = {}) {
	const timeoutMs = options.timeoutMs ?? 10_000;
	if (!isProcessAlive(pid)) return true;
	try {
		process.kill(pid, 'SIGTERM');
	} catch {
		// The process exited in the meantime.
	}
	const deadline = Date.now() + timeoutMs;
	while (Date.now() < deadline) {
		if (!isProcessAlive(pid)) return true;
		await sleep(100);
	}
	try {
		process.kill(pid, 'SIGKILL');
	} catch {
		// The process exited in the meantime.
	}
	await sleep(500);
	return !isProcessAlive(pid);
}
