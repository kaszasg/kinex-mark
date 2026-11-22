// Starts the real kinex-mark binary for the E2E suite (plan 9.4.1): Playwright's webServer
// command. Dependency-free Node script.
//
// 1. Binary: KINEX_MARK_BIN, default ../dist/kinex-mark(.exe).
// 2. Removes data directories of earlier, forcibly stopped runs (older than one hour).
// 3. Creates a fresh temporary data directory and records it in test-results/.e2e-data-dir.
// 4. Starts the metadata fixture origin on 127.0.0.1:18081.
// 5. Starts the binary on 127.0.0.1:18080 with the E2E environment.
// 6. On SIGTERM/SIGINT stops the binary, waits for its exit, then removes the data directory.
//    Playwright on Windows kills the process tree instead, so there global-teardown.ts and
//    step 2 of the next run clean up.
import { spawn } from 'node:child_process';
import { existsSync } from 'node:fs';
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { removeDirWithRetry, removeStaleDataDirs } from './cleanup.mjs';
import {
	ADMIN_PASSWORD,
	ADMIN_USERNAME,
	DATA_DIR_MARKER,
	DATA_DIR_PREFIX,
	SERVER_HOST,
	SERVER_PID_MARKER,
	SERVER_PORT,
	TEST_RESULTS_DIR
} from './constants.mjs';
import { startFixtureOrigin } from './fixture-origin.mjs';

const REPO_ROOT = fileURLToPath(new URL('../../../', import.meta.url));

/** @param {string} message */
function log(message) {
	console.error(`[start-server] ${message}`);
}

/** Path of the server binary: KINEX_MARK_BIN or the build output of build.ps1 / make build. */
export function resolveBinary() {
	const fromEnv = process.env.KINEX_MARK_BIN;
	if (fromEnv) return path.resolve(fromEnv);
	const name = process.platform === 'win32' ? 'kinex-mark.exe' : 'kinex-mark';
	return path.join(REPO_ROOT, 'dist', name);
}

/**
 * Environment of the server under test. Inherited APP_* variables are dropped, so a local
 * APP_BASE_URL or APP_SECRET_KEY cannot leak into the run.
 * @param {string} dataDir
 * @returns {NodeJS.ProcessEnv}
 */
export function serverEnv(dataDir) {
	/** @type {NodeJS.ProcessEnv} */
	const env = {};
	for (const [key, value] of Object.entries(process.env)) {
		if (!key.toUpperCase().startsWith('APP_')) env[key] = value;
	}
	return {
		...env,
		APP_ADDR: `${SERVER_HOST}:${SERVER_PORT}`,
		APP_DATA_DIR: dataDir,
		APP_ADMIN_USER: ADMIN_USERNAME,
		APP_ADMIN_PASSWORD: ADMIN_PASSWORD,
		// Also locks the allowPrivateFetch system setting (lockedByEnv).
		APP_ALLOW_PRIVATE_FETCH: 'true',
		// The fixture origin runs on 127.0.0.1 (plan 6.8).
		APP_ALLOW_LOOPBACK_FETCH: 'true',
		APP_SECURE_COOKIES: 'false',
		APP_FETCH_TIMEOUT: '5s',
		APP_LOG_LEVEL: 'warn'
	};
}

async function main() {
	const binary = resolveBinary();
	if (!existsSync(binary)) {
		log(`server binary not found: ${binary}`);
		log('Run ".\\build.ps1" or "make build" first.');
		process.exit(1);
	}

	const stale = await removeStaleDataDirs({ log });
	if (stale.removed.length > 0) {
		log(
			`removed ${stale.removed.length} stale data director${stale.removed.length === 1 ? 'y' : 'ies'}`
		);
	}

	const dataDir = await mkdtemp(path.join(os.tmpdir(), DATA_DIR_PREFIX));
	await mkdir(TEST_RESULTS_DIR, { recursive: true });
	await writeFile(DATA_DIR_MARKER, `${dataDir}\n`, 'utf8');

	let fixture;
	try {
		fixture = await startFixtureOrigin({ log });
	} catch (err) {
		log(`fixture origin failed to start: ${err instanceof Error ? err.message : String(err)}`);
		await rm(dataDir, { recursive: true, force: true });
		process.exit(1);
	}

	const child = spawn(binary, [], {
		env: serverEnv(dataDir),
		stdio: ['ignore', 'inherit', 'inherit'],
		windowsHide: true
	});
	if (child.pid !== undefined) {
		await writeFile(SERVER_PID_MARKER, `${child.pid}\n`, 'utf8');
	}

	/** @type {Promise<{ code: number | null, signal: NodeJS.Signals | null }>} */
	const exited = new Promise((resolve) => {
		child.once('exit', (code, signal) => resolve({ code, signal }));
		child.once('error', (err) => {
			log(`cannot start ${binary}: ${err.message}`);
			resolve({ code: 1, signal: null });
		});
	});

	let stopping = false;
	/** @param {NodeJS.Signals} signal */
	const stop = (signal) => {
		if (stopping) return;
		stopping = true;
		log(`${signal} received, stopping the server`);
		if (child.exitCode === null && child.signalCode === null) child.kill('SIGTERM');
		const force = setTimeout(() => {
			if (child.exitCode === null && child.signalCode === null) child.kill('SIGKILL');
		}, 5_000);
		force.unref();
	};
	process.once('SIGTERM', () => stop('SIGTERM'));
	process.once('SIGINT', () => stop('SIGINT'));

	const { code, signal } = await exited;
	if (!stopping) {
		log(`server exited (code ${code ?? 'none'}, signal ${signal ?? 'none'})`);
	}
	await rm(SERVER_PID_MARKER, { force: true });
	await fixture.close();
	// Only after the exit: on Windows the open app.db locks the directory.
	const error = await removeDirWithRetry(dataDir, { timeoutMs: 5_000 });
	if (error) log(`cannot remove data directory ${dataDir}: ${error.message}`);
	process.exit(stopping ? 0 : code || 1);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
	await main();
}
