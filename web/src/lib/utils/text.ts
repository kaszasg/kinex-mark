// Free-text query values as the server accepts them (plan 5.1): trimmed, no control or bidi
// runes, and at most a given number of runes (code points).

/** C0 and C1 controls with DEL: a pasted tab or line break becomes a plain space. */
const CONTROL = /[\u0000-\u001f\u007f-\u009f]/g;
/**
 * The Unicode Bidi_Control set (ALM, LRM, RLM, embeddings, overrides and isolates): invisible,
 * so it is dropped, as on the server.
 */
const BIDI = /[\u061c\u200e\u200f\u202a-\u202e\u2066-\u2069]/g;

/**
 * Cleans a search text before it is sent: controls become spaces, bidi runes disappear, the
 * ends are trimmed and the text is cut to `max` runes, so the server never answers with
 * 400 invalid_query for something the user typed or pasted.
 */
export function cleanQueryText(text: string, max: number): string {
	const cleaned = text.replace(CONTROL, ' ').replace(BIDI, '').trim();
	const runes = [...cleaned];
	return runes.length > max ? runes.slice(0, max).join('').trim() : cleaned;
}
