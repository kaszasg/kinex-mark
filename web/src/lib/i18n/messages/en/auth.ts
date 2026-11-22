import type hu from '../hu/auth.js';
import type { Shape } from '../../types.js';

export default {
	login: {
		title: 'Sign in',
		subtitle: 'Kinex Mark, the Kinexium link collection.',
		username: 'Username',
		password: 'Password',
		submit: 'Sign in',
		noAccount: 'Accounts are created by an administrator.',
		version: 'Version {version}',
		footer: 'Language, theme and privacy notice'
	},
	totp: {
		title: 'Two-factor sign-in',
		text: 'Enter the 6-digit code from your authenticator app.',
		code: 'Authentication code',
		submit: 'Verify code',
		useRecovery: 'Use a recovery code',
		useCode: 'Use an authentication code',
		back: 'Back to sign-in'
	},
	recovery: {
		text: 'Enter one of the recovery codes you saved.',
		code: 'Recovery code',
		placeholder: 'xxxxx-xxxxx',
		help: 'Hyphens and spaces are optional.',
		used: {
			one: 'You signed in with a recovery code. {count} code is left.',
			other: 'You signed in with a recovery code. {count} codes are left.'
		},
		low: 'Generate new codes on the Settings › Security tab.'
	},
	logout: {
		title: 'You have signed out.',
		text: 'Sign in again to continue.'
	},
	expired: {
		title: 'Your session has expired.',
		text: 'Sign in again.'
	},
	accountDeleted: {
		title: 'Your account has been deleted.',
		text: 'Your links, tags, shares and settings are permanently gone.'
	},
	passwordChange: {
		pageTitle: 'Password change',
		title: 'Set a new password',
		text: 'Your account was created with a temporary password, or an administrator reset it. Choose a new one to continue.',
		current: 'Current password',
		new: 'New password',
		confirm: 'Repeat new password',
		mismatch: 'The two passwords do not match.',
		submit: 'Change password',
		saved: 'Password changed',
		savedOtherSessions: 'Password changed. You were signed out on your other devices.'
	},
	passwordPolicy:
		'At least {min} characters, at least {distinct} different characters, without your username.'
} satisfies Shape<typeof hu>;
