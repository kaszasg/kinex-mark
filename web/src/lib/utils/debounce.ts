// Trailing-edge debounce with cancel and flush.

export type Debounced<A extends unknown[]> = ((...args: A) => void) & {
	cancel(): void;
	flush(): void;
	readonly pending: boolean;
};

/** Delays the call until `wait` ms passed without another call; the last arguments win. */
export function debounce<A extends unknown[]>(
	fn: (...args: A) => void,
	wait: number
): Debounced<A> {
	let timer: ReturnType<typeof setTimeout> | null = null;
	let lastArgs: A | null = null;

	const run = () => {
		timer = null;
		const args = lastArgs as A;
		lastArgs = null;
		fn(...args);
	};

	const debounced = ((...args: A) => {
		lastArgs = args;
		if (timer !== null) clearTimeout(timer);
		timer = setTimeout(run, wait);
	}) as Debounced<A>;

	debounced.cancel = () => {
		if (timer !== null) clearTimeout(timer);
		timer = null;
		lastArgs = null;
	};
	debounced.flush = () => {
		if (timer === null) return;
		clearTimeout(timer);
		run();
	};
	Object.defineProperty(debounced, 'pending', { get: () => timer !== null });
	return debounced;
}
