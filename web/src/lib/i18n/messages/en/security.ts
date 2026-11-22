import type hu from '../hu/security.js';
import type { Shape } from '../../types.js';

// Settings > Security and Settings > Privacy (plan 8.7.6, owner F5).
export default {
	password: {
		title: 'Password',
		text: 'Changing your password signs you out on your other devices.'
	},
	twoFactor: {
		title: 'Two-factor sign-in',
		text: 'Signing in also takes the six-digit code from your authenticator app.',
		remaining: 'Unused recovery codes',
		lowCodes: {
			title: 'You have few recovery codes left',
			text: 'Generate new ones before they run out.'
		},
		enable: 'Turn on',
		regenerate: 'New recovery codes',
		disable: 'Turn off',
		enabled: 'Two-factor sign-in turned on',
		disabled: 'Two-factor sign-in turned off',
		regenerated: 'New recovery codes created'
	},
	enroll: {
		step: '{current} / {total}',
		stepLabel: 'Step {current} of {total}',
		password: {
			title: 'Confirm your password',
			text: 'Enter your current password to turn on two-factor sign-in.',
			label: 'Current password',
			submit: 'Confirm password'
		},
		scan: {
			title: 'Scan the QR code',
			text: 'Open your authenticator app and scan the code. Then enter the code the app shows.',
			qr: 'QR code for your authenticator app',
			manual: 'For manual entry:',
			copySecret: 'Copy secret key',
			secretCopied: 'Secret key copied',
			expires: 'This setup is valid until {time}',
			code: 'Authentication code',
			codeHelp: 'The six-digit code shown in your app.',
			submit: 'Verify code'
		},
		codes: {
			title: 'Save your recovery codes'
		}
	},
	recovery: {
		text: 'If you cannot reach your authenticator app, sign in with one of these. Each code works once.',
		listLabel: 'Recovery codes',
		printHeading: 'Kinex Mark recovery codes: {username}',
		copy: 'Copy codes',
		copied: 'Codes copied',
		download: 'Download as .txt',
		print: 'Print',
		saved: 'I have saved the codes and understand they are shown only now.',
		file: {
			title: 'Kinex Mark recovery codes',
			intro: 'Account: {username}. Each code can be used once instead of an authentication code.',
			generatedAt: 'Created: {time}',
			warning:
				'Keep them in a safe place. Anyone who knows your password and one code can sign in to your account.'
		}
	},
	confirm: {
		password: 'Current password',
		disable: {
			title: 'Turn off two-factor sign-in?',
			text: 'From then on, signing in takes only your password. Your recovery codes are deleted and you are signed out on your other devices.',
			code: 'Authentication or recovery code',
			codeHelp: 'The six-digit code from your app or an unused recovery code.',
			submit: 'Turn off two-factor sign-in'
		},
		regenerate: {
			title: 'Create new recovery codes?',
			text: 'Your old codes stop working. The new ones are shown only once.',
			code: 'Authentication code',
			codeHelp: 'The six-digit code shown in your app.',
			submit: 'Create new codes'
		}
	},
	sessions: {
		title: 'Sessions',
		text: 'The devices where you are signed in.',
		listLabel: 'Active sessions',
		current: 'This device',
		unknownDevice: 'Unknown device',
		ip: 'IP address',
		created: 'Created',
		lastSeen: 'Last activity',
		expires: 'Expires',
		revoke: 'Sign out: {device}',
		revoked: 'Session ended: {device}',
		revokeOthers: 'Sign out on other devices',
		revokedOthers: {
			zero: 'There were no other sessions',
			one: '{count} session ended',
			other: '{count} sessions ended'
		},
		revokeCurrent: {
			title: 'Sign out on this device?',
			text: 'This session ends and you need to sign in again.',
			submit: 'Sign out on this device'
		},
		empty: {
			title: 'No active sessions',
			text: 'Refresh the list or sign in again.'
		}
	},
	privacy: {
		notice: {
			title: 'Privacy notice',
			text: 'Accepting the notice confirms that you have read it.',
			current: 'Current version',
			updatedAt: 'Updated',
			version: 'Version {version}',
			history: 'Acceptances',
			acceptedAt: 'Accepted: {time}',
			currentBadge: 'Current',
			none: {
				title: 'No notice accepted yet',
				text: 'Open the notice and accept it.'
			},
			open: 'Open the notice',
			summary: 'Text of the notice'
		},
		export: {
			title: 'Download your data',
			text: 'The JSON file contains all data that belongs to your account:',
			items: {
				account: 'account details',
				preferences: 'preferences',
				sessions: 'sessions',
				tags: 'tags',
				links: 'your own links with their shares',
				shared: 'your personal state on links shared with you',
				audit: 'security log entries about your account, as actor and as target',
				lock: 'the state of the sign-in lock'
			},
			excluded: 'Your password, secret keys and recovery codes are not included.',
			submit: 'Download data (JSON)',
			done: 'Data downloaded'
		}
	},
	deleteAccount: {
		title: 'Delete account',
		text: 'Your account and all its data are deleted for good. If you want to keep your links, download your data first.',
		open: 'Delete account',
		dialogTitle: 'Delete your account?',
		dialogText:
			'All your links, tags, shares and preferences are deleted for good. This cannot be undone.',
		shared: {
			title: {
				one: '{count} shared link also disappears for its recipients.',
				other: '{count} shared links also disappear for their recipients.'
			},
			text: 'If they want to keep them, ask them to add the links to their own collection.'
		},
		password: 'Password',
		code: 'Authentication or recovery code',
		confirm: 'Type your username to confirm: {username}',
		submit: 'Delete account for good'
	}
} satisfies Shape<typeof hu>;
