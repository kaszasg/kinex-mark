// Saves a Blob through a temporary download link (plan 8.3).

/** Starts a browser download of the blob under the given file name. */
export function saveBlob(blob: Blob, filename: string): void {
	const url = URL.createObjectURL(blob);
	const anchor = document.createElement('a');
	anchor.href = url;
	anchor.download = filename;
	anchor.rel = 'noopener';
	anchor.style.display = 'none';
	document.body.appendChild(anchor);
	try {
		anchor.click();
	} finally {
		anchor.remove();
		// Revoke after the click has been handled by the browser.
		setTimeout(() => URL.revokeObjectURL(url), 0);
	}
}

/** Saves a text file (for example the recovery codes) as UTF-8. */
export function saveText(text: string, filename: string): void {
	saveBlob(new Blob([text], { type: 'text/plain;charset=utf-8' }), filename);
}
