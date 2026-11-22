// DOM Testing Library and jest-dom collapse every whitespace run of the rendered text, U+00A0
// included, but compare against the expected string as given. Texts that keep a value and its
// unit together with U+00A0 (plan 8.5.4) are compared in the same collapsed form.
export function normalizeSpace(value: string): string {
	return value.replace(/\s+/g, ' ').trim();
}
