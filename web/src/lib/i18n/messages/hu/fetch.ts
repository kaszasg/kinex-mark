export default {
	status: {
		pending: 'Letöltés folyamatban',
		ok: 'Letöltve',
		failed: 'Sikertelen',
		blocked: 'Tiltott hálózati cím'
	},
	errors: {
		timeout: 'Időtúllépés',
		dns: 'A domain nem található',
		connection: 'Kapcsolódási hiba',
		tls: 'Tanúsítványhiba',
		http_status: 'A webhely hibát adott vissza',
		too_large: 'Az oldal túl nagy',
		not_html: 'Nem weboldal (nincs cím)',
		too_many_redirects: 'Túl sok átirányítás',
		blocked_address: 'Belső hálózati cím, a letöltés tiltott',
		blocked_port: 'Tiltott port'
	}
};
