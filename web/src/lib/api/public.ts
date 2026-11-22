// Public endpoints (plan 5.6, 5.14).
import { request } from './client.js';
import type { PublicConfig } from './types.js';

/** GET /api/public/config */
export function getPublicConfig(): Promise<PublicConfig> {
	return request<PublicConfig>('GET', '/public/config', { skipAuthRedirect: true });
}

/** GET /healthz */
export function getHealth(): Promise<{ status: string }> {
	return request<{ status: string }>('GET', '/healthz', { skipAuthRedirect: true });
}
