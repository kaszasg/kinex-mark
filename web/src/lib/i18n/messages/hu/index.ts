// Aggregated catalogue of one locale: one import per namespace (plan 8.5.1).
import common from './common.js';
import nav from './nav.js';
import errors from './errors.js';
import fields from './fields.js';
import roles from './roles.js';
import status from './status.js';
import auth from './auth.js';
import landing from './landing.js';
import privacy from './privacy.js';
import links from './links.js';
import capture from './capture.js';
import fetchMessages from './fetch.js';
import tags from './tags.js';
import tagColors from './tagColors.js';
import settings from './settings.js';
import admin from './admin.js';
import audit from './audit.js';
import security from './security.js';

export default {
	common,
	nav,
	errors,
	fields,
	roles,
	status,
	auth,
	landing,
	privacy,
	links,
	capture,
	fetch: fetchMessages,
	tags,
	tagColors,
	settings,
	admin,
	audit,
	security
};
