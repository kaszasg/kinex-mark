// Short device description for the sessions list (plan 8.7.6): "Chrome, Windows".

const BROWSERS: [RegExp, string][] = [
	[/\bEdgA?\//, 'Edge'],
	[/\bOPR\/|\bOpera\b/, 'Opera'],
	[/\bSamsungBrowser\//, 'Samsung Internet'],
	[/\bFirefox\/|\bFxiOS\//, 'Firefox'],
	[/\bChrome\/|\bCriOS\/|\bChromium\//, 'Chrome'],
	[/\bVersion\/[\d.]+.*\bSafari\//, 'Safari']
];

const SYSTEMS: [RegExp, string][] = [
	[/\bWindows\b/, 'Windows'],
	[/\biPhone\b|\biPad\b|\biPod\b/, 'iOS'],
	[/\bAndroid\b/, 'Android'],
	[/\bCrOS\b/, 'ChromeOS'],
	[/\bMac OS X\b|\bMacintosh\b/, 'macOS'],
	[/\bLinux\b/, 'Linux']
];

function firstMatch(ua: string, table: [RegExp, string][]): string | null {
	for (const [pattern, name] of table) if (pattern.test(ua)) return name;
	return null;
}

/** "Browser, OS" (or one of them) for a known user agent; null when neither is recognised. */
export function describeUserAgent(ua: string): string | null {
	const browser = firstMatch(ua, BROWSERS);
	const system = firstMatch(ua, SYSTEMS);
	if (browser && system) return `${browser}, ${system}`;
	return browser ?? system;
}

/** The raw fallback shown in mono: the first 40 characters of the user agent. */
export function rawUserAgent(ua: string): string {
	return ua.slice(0, 40);
}
