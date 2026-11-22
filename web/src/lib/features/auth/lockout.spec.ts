import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { ApiError } from '$lib/api/errors.js';
import {
	DEFAULT_LOCK_SECONDS,
	LOCK_TICK_MS,
	Lockout,
	retryAfterSeconds
} from './lockout.svelte.js';

beforeEach(() => {
	vi.useFakeTimers();
	vi.setSystemTime(new Date('2026-11-21T10:00:00Z'));
});

afterEach(() => {
	vi.useRealTimers();
});

describe('retryAfterSeconds', () => {
	test.each([
		[{ retryAfterSeconds: 120 }, 120],
		[{ retryAfterSeconds: 90.2 }, 91],
		[{ retryAfterSeconds: '30' }, 30],
		[{ retryAfterSeconds: 0 }, DEFAULT_LOCK_SECONDS],
		[{ retryAfterSeconds: 'soon' }, DEFAULT_LOCK_SECONDS],
		[{}, DEFAULT_LOCK_SECONDS]
	])('details %j give %i seconds', (details, seconds) => {
		expect(retryAfterSeconds(new ApiError(429, 'too_many_attempts', {}, details))).toBe(seconds);
	});
});

describe('Lockout', () => {
	test('is inactive without a lock', () => {
		const lockout = new Lockout();
		expect(lockout.active).toBe(false);
		expect(lockout.remainingSeconds).toBe(0);
		expect(lockout.error).toBeNull();
	});

	test('counts down every second and reports the remaining time in its error', () => {
		const lockout = new Lockout();
		lockout.start(125);
		expect(lockout.active).toBe(true);
		expect(lockout.error?.code).toBe('too_many_attempts');
		expect(lockout.error?.details.retryAfterSeconds).toBe(125);
		vi.advanceTimersByTime(65 * LOCK_TICK_MS);
		expect(lockout.remainingSeconds).toBe(60);
		lockout.stop();
	});

	test('calls the expiry callback once the lock is over and stops ticking', () => {
		const onExpire = vi.fn();
		const lockout = new Lockout(onExpire);
		lockout.start(2);
		vi.advanceTimersByTime(LOCK_TICK_MS);
		expect(onExpire).not.toHaveBeenCalled();
		vi.advanceTimersByTime(LOCK_TICK_MS);
		expect(onExpire).toHaveBeenCalledTimes(1);
		expect(lockout.active).toBe(false);
		expect(lockout.until).toBeNull();
		vi.advanceTimersByTime(10 * LOCK_TICK_MS);
		expect(onExpire).toHaveBeenCalledTimes(1);
	});

	test('a restart replaces the running lock, a lock lasts at least one second', () => {
		const lockout = new Lockout();
		lockout.start(300);
		lockout.start(0);
		expect(lockout.remainingSeconds).toBe(1);
		vi.advanceTimersByTime(LOCK_TICK_MS);
		expect(lockout.active).toBe(false);
	});

	test('stop ends the lock without the callback', () => {
		const onExpire = vi.fn();
		const lockout = new Lockout(onExpire);
		lockout.start(30);
		lockout.stop();
		vi.advanceTimersByTime(60 * LOCK_TICK_MS);
		expect(lockout.active).toBe(false);
		expect(onExpire).not.toHaveBeenCalled();
	});
});
