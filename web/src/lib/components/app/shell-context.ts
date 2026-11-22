// Context shared by the AppShell and the components inside it.
import { getContext, setContext } from 'svelte';

export type ShellContext = {
	/** Opens or closes the off-canvas sidebar below 1024 px. */
	toggleSidebar(): void;
	/** Closes the off-canvas sidebar. */
	closeSidebar(): void;
	/** Opens the keyboard shortcuts dialog (links display menu, "?" key). */
	openShortcuts(): void;
};

const KEY = Symbol.for('kxm-shell');

export function setShellContext(context: ShellContext): ShellContext {
	return setContext(KEY, context);
}

/** The shell context, or a no-op fallback outside the AppShell (tests, public pages). */
export function getShellContext(): ShellContext {
	return (
		getContext<ShellContext | undefined>(KEY) ?? {
			toggleSidebar: () => {},
			closeSidebar: () => {},
			openShortcuts: () => {}
		}
	);
}
