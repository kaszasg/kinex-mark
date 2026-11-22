// Countdown of a sign-in lock (429 too_many_attempts with Retry-After, plan 8.7.2).
import { ApiError } from '$lib/api/errors.js';

/** Fallback when a 429 answer carries no retry time. */
export const DEFAULT_LOCK_SECONDS = 60;
/** The countdown ticks every second; the alert shows whole minutes. */
export const LOCK_TICK_MS = 1000;

/** Reads `details.retryAfterSeconds` of a too_many_attempts error. */
export function retryAfterSeconds(err: ApiError): number {
	const value = Number(err.details.retryAfterSeconds);
	return Number.isFinite(value) && value > 0 ? Math.ceil(value) : DEFAULT_LOCK_SECONDS;
}

export class Lockout {
	until = $state<number | null>(null);
	now = $state(Date.now());
	#timer: ReturnType<typeof setInterval> | null = null;
	#onExpire: (() => void) | undefined;

	constructor(onExpire?: () => void) {
		this.#onExpire = onExpire;
	}

	/** Whole seconds left; 0 when there is no lock. */
	get remainingSeconds(): number {
		if (this.until === null) return 0;
		return Math.max(0, Math.ceil((this.until - this.now) / 1000));
	}

	get active(): boolean {
		return this.remainingSeconds > 0;
	}

	/** The error to display, with the remaining time as its retry hint. */
	get error(): ApiError | null {
		if (!this.active) return null;
		return new ApiError(429, 'too_many_attempts', {}, { retryAfterSeconds: this.remainingSeconds });
	}

	/** Starts (or restarts) the lock for the given number of seconds. */
	start(seconds: number): void {
		this.#clearTimer();
		this.now = Date.now();
		this.until = this.now + Math.max(1, seconds) * 1000;
		this.#timer = setInterval(() => this.#tick(), LOCK_TICK_MS);
	}

	/** Ends the lock without calling the expiry callback (component teardown, new attempt). */
	stop(): void {
		this.#clearTimer();
		this.until = null;
	}

	#tick(): void {
		this.now = Date.now();
		if (this.active) return;
		this.stop();
		this.#onExpire?.();
	}

	#clearTimer(): void {
		if (this.#timer !== null) clearInterval(this.#timer);
		this.#timer = null;
	}
}
