// RFC 6238 TOTP for the E2E suite: SHA-1, 6 digits, 30 second steps (plan 9.4.1).
import { createHmac } from 'node:crypto';
import { setTimeout as sleep } from 'node:timers/promises';

const BASE32_ALPHABET = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
export const STEP_SECONDS = 30;
const DIGITS = 6;

export interface TotpCode {
	code: string;
	step: number;
}

/** Decodes an RFC 4648 base32 string; padding, spaces and letter case are ignored. */
export function base32Decode(input: string): Buffer {
	const clean = input.replace(/[\s=]/g, '').toUpperCase();
	const bytes: number[] = [];
	let buffer = 0;
	let bits = 0;
	for (const char of clean) {
		const value = BASE32_ALPHABET.indexOf(char);
		if (value < 0) {
			throw new Error(`invalid base32 character: ${char}`);
		}
		buffer = (buffer << 5) | value;
		bits += 5;
		if (bits >= 8) {
			bits -= 8;
			bytes.push((buffer >> bits) & 0xff);
		}
	}
	return Buffer.from(bytes);
}

/** Returns the time step that contains the given Unix time in milliseconds. */
export function stepAt(timeMs: number): number {
	return Math.floor(timeMs / 1000 / STEP_SECONDS);
}

/** Computes the code of one time step for a raw key. */
export function hotp(key: Buffer, step: number): string {
	const counter = Buffer.alloc(8);
	counter.writeBigUInt64BE(BigInt(step));
	const digest = createHmac('sha1', key).update(counter).digest();
	const offset = digest[digest.length - 1] & 0x0f;
	const binary = digest.readUInt32BE(offset) & 0x7fffffff;
	return String(binary % 10 ** DIGITS).padStart(DIGITS, '0');
}

/** Computes the code for a base32 secret at the given Unix time in milliseconds. */
export function totpAt(secret: string, timeMs: number): TotpCode {
	const step = stepAt(timeMs);
	return { code: hotp(base32Decode(secret), step), step };
}

/**
 * Returns a code the server has not accepted yet. When the current step equals
 * usedStep (a code of that step was already consumed), it waits for the next step.
 */
export async function freshCode(secret: string, usedStep?: number): Promise<TotpCode> {
	const now = Date.now();
	if (usedStep !== undefined && stepAt(now) <= usedStep) {
		const nextStepStart = (usedStep + 1) * STEP_SECONDS * 1000;
		await sleep(nextStepStart - now + 50);
	}
	return totpAt(secret, Date.now());
}
