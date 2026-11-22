// Accent- and case-insensitive text key (plan 4.4): the browser twin of the server's foldText in
// internal/store/driver.go, used for the name order and the search after an optimistic change.

const COMBINING_MARK = /\p{M}/gu;

/**
 * Lower case, canonically decomposed (NFD), without combining marks, recomposed (NFC): an a acute
 * becomes a, an o double acute becomes o; letters without a decomposition stay as they are.
 */
export function foldText(value: string): string {
	return value.toLowerCase().normalize('NFD').replace(COMBINING_MARK, '').normalize('NFC');
}
