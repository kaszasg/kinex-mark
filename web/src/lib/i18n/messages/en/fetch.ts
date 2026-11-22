import type hu from '../hu/fetch.js';
import type { Shape } from '../../types.js';

export default {
	status: {
		pending: 'Fetching',
		ok: 'Fetched',
		failed: 'Failed',
		blocked: 'Blocked address'
	},
	errors: {
		timeout: 'Timed out',
		dns: 'Domain not found',
		connection: 'Connection error',
		tls: 'Certificate error',
		http_status: 'The site returned an error',
		too_large: 'The page is too large',
		not_html: 'Not a web page (no title)',
		too_many_redirects: 'Too many redirects',
		blocked_address: 'Internal network address, fetching is blocked',
		blocked_port: 'Port not allowed'
	}
} satisfies Shape<typeof hu>;
