// Public surface of the API layer: the client, the error type, the shapes and one namespace per module.
export {
	request,
	download,
	onAuthFailure,
	buildUrl,
	filenameFromDisposition,
	type Query,
	type RequestOptions,
	type Method
} from './client.js';
export {
	ApiError,
	codeFromStatus,
	isApiError,
	type ErrorCode,
	type FieldCode,
	type AuthFailureCode
} from './errors.js';
export type * from './types.js';
export * as publicApi from './public.js';
export * as authApi from './auth.js';
export * as meApi from './me.js';
export * as linksApi from './links.js';
export * as tagsApi from './tags.js';
export * as directoryApi from './directory.js';
export * as adminApi from './admin.js';
