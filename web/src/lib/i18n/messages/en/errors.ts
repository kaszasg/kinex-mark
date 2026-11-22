import type hu from '../hu/errors.js';
import type { Shape } from '../../types.js';

export default {
	endpoint_not_found: { title: 'This function is not available.', text: 'Reload the page.' },
	invalid_json: { title: 'The request was malformed.', text: 'Reload the page and try again.' },
	invalid_id: { title: 'Invalid identifier.', text: 'Open the item again from the list.' },
	invalid_query: { title: 'Invalid filter.', text: 'Clear the filters and try again.' },
	validation_failed: { title: 'Some fields are invalid.', text: 'Correct the marked fields.' },
	payload_too_large: {
		title: 'The file is larger than 2 MB.',
		text: 'Choose a smaller file.'
	},
	not_found: {
		title: 'The item was not found.',
		text: 'It may have been deleted or is no longer visible to you. Refresh the list.'
	},
	unauthenticated: { title: 'Your session has expired.', text: 'Sign in again.' },
	forbidden: {
		title: 'You do not have permission for this.',
		text: 'Contact an administrator.'
	},
	csrf_rejected: {
		title: 'A security check rejected the request.',
		text: 'Reload the page and try again.'
	},
	host_mismatch: {
		title: 'The request did not reach the configured Kinex Mark address.',
		text: 'Open the app at its usual address or contact an administrator.'
	},
	password_change_required: {
		title: 'Password change required.',
		text: 'Set a new password to continue.'
	},
	consent_required: {
		title: 'The privacy notice must be accepted.',
		text: 'Accept it to continue.'
	},
	invalid_credentials: {
		title: 'Incorrect username or password.',
		text: 'Check them and try again.'
	},
	account_disabled: { title: 'This account is disabled.', text: 'Contact an administrator.' },
	too_many_attempts: {
		title: 'Too many failed attempts.',
		text: 'Try again in {minutes} min.'
	},
	challenge_invalid: { title: 'The sign-in request has expired.', text: 'Sign in again.' },
	invalid_totp: {
		title: 'The code is wrong or already used.',
		text: 'Enter the new code shown in your app.'
	},
	current_password_invalid: {
		title: 'The current password is incorrect.',
		text: 'Enter it again.'
	},
	totp_already_enabled: { title: 'Two-factor sign-in is already on.', text: 'Reload the page.' },
	totp_not_enabled: { title: 'Two-factor sign-in is off.', text: 'Reload the page.' },
	totp_setup_required: { title: 'The setup has expired.', text: 'Start the setup again.' },
	link_duplicate: {
		title: 'This link is already in your collection.',
		text: 'Open the existing one.'
	},
	link_limit_reached: {
		title: 'You have reached the limit of 10,000 links.',
		text: 'Delete old links before adding new ones.'
	},
	not_owner: {
		title: 'Only the owner can change this link.',
		text: 'You can still set your own tags and pin.'
	},
	share_not_allowed: {
		title: 'Your role cannot share links.',
		text: 'Ask a developer or an administrator.'
	},
	share_duplicate: {
		title: 'Already shared with this recipient.',
		text: 'Choose another recipient.'
	},
	no_url_found: {
		title: 'No URL found.',
		text: 'Drop or paste a link starting with http or https.'
	},
	import_invalid: {
		title: 'The file could not be read.',
		text: 'Choose a Kinex Mark JSON export or a browser HTML bookmark file.'
	},
	import_too_many_entries: {
		title: 'The file has more than 5,000 entries.',
		text: 'Split it into several files and import them one after another.'
	},
	tag_duplicate: { title: 'A tag with this name already exists.', text: 'Choose another name.' },
	tag_limit_reached: {
		title: 'You have reached the limit of 500 tags.',
		text: 'Delete unused tags.'
	},
	username_taken: { title: 'This username is taken.', text: 'Choose another one.' },
	last_admin: {
		title: 'This is the last active administrator.',
		text: 'Appoint another administrator first.'
	},
	self_modification: {
		title: 'You cannot change your own account here.',
		text: 'Another administrator can change your role; delete your account under Settings.'
	},
	setting_locked: {
		title: 'This setting is fixed by an environment variable.',
		text: 'Change the server configuration to modify it: {keys}.'
	},
	consent_version_mismatch: {
		title: 'A new notice has been published.',
		text: 'Read it and accept it again.'
	},
	confirmation_mismatch: {
		title: 'The confirmation does not match your username.',
		text: 'Type it exactly.'
	},
	not_implemented: {
		title: 'This function is not available yet.',
		text: 'Try again in a later version.'
	},
	internal_error: { title: 'An unexpected server error occurred.', text: 'Try again later.' },
	network_error: {
		title: 'The server cannot be reached.',
		text: 'Check your network connection.'
	}
} satisfies Shape<typeof hu>;
