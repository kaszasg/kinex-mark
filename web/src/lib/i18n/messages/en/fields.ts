import type hu from '../hu/fields.js';
import type { Shape } from '../../types.js';

export default {
	required: 'This field is required.',
	too_short: 'Too short.',
	too_long: 'Too long.',
	invalid: 'Invalid value.',
	invalid_format: 'Invalid format.',
	not_allowed: 'This value is not allowed.',
	out_of_range: 'The value is out of range.',
	unknown: 'Unknown item. Refresh the list.',
	duplicate: 'This already exists.',
	credentials_in_url: 'The URL must not contain a username or password.',
	incorrect: 'Incorrect value.',
	password_too_short: 'Use at least 12 characters.',
	password_too_long: 'Use at most 128 characters.',
	password_contains_username: 'The password must not contain the username.',
	password_too_simple: 'Use at least 6 different characters.',
	password_reused: 'The new password must differ from the current one.',
	range: 'Enter a value between {min} and {max}.',
	byField: {
		url: {
			invalid_format: 'Enter a full address, for example https://example.com.',
			not_allowed: 'Only http and https addresses can be saved.',
			too_long: 'The URL can be at most 2048 bytes long.'
		},
		username: {
			invalid_format: 'Letters, digits, dot, hyphen and underscore only.'
		},
		tagIds: {
			too_long: 'At most 20 tags.'
		},
		privacyControllerEmail: {
			invalid_format: 'Enter a valid e-mail address.'
		}
	}
} satisfies Shape<typeof hu>;
