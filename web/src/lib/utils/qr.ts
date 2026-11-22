// QR matrix and SVG path for the 2FA enrolment (plan 8.4 QrCode, D4).
import { encode } from 'uqr';

/** Quiet zone around the code, in modules. */
export const QUIET_ZONE = 4;

export type QrMatrix = { size: number; modules: boolean[][] };

/** The module matrix of the text with the quiet zone included (error correction M). */
export function qrMatrix(text: string): QrMatrix {
	const result = encode(text, { ecc: 'M', border: QUIET_ZONE });
	return { size: result.size, modules: result.data };
}

/** One SVG path that draws every dark module as a 1x1 square (viewBox 0 0 size size). */
export function qrPath(matrix: QrMatrix): string {
	const parts: string[] = [];
	matrix.modules.forEach((row, y) => {
		row.forEach((dark, x) => {
			if (dark) parts.push(`M${x} ${y}h1v1h-1z`);
		});
	});
	return parts.join('');
}
