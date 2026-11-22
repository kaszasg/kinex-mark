import type hu from '../hu/admin.js';
import type { Shape } from '../../types.js';

export default {
	users: {
		pageTitle: 'Users',
		summary: { one: '{count} user', other: '{count} users' },
		add: 'Add user',
		filters: {
			label: 'Filter users',
			search: 'Search',
			searchPlaceholder: 'Username or name',
			role: 'Role',
			allRoles: 'All',
			status: 'Status',
			statusActive: 'Active',
			statusInactive: 'Inactive',
			statusAll: 'All',
			clear: 'Clear filters'
		},
		table: {
			caption: 'List of users',
			username: 'Username',
			displayName: 'Display name',
			role: 'Role',
			status: 'Status',
			twoFactor: '2FA',
			twoFactorOn: 'On',
			links: 'Links',
			sessions: 'Sessions',
			lastLogin: 'Last sign-in',
			actions: 'Actions'
		},
		actions: {
			menu: 'Actions for {username}',
			edit: 'Edit',
			resetPassword: 'Reset password',
			resetTwoFactor: 'Reset 2FA',
			unlock: 'Unlock',
			revokeSessions: 'Revoke sessions',
			deactivate: 'Deactivate',
			activate: 'Activate',
			delete: 'Delete',
			selfHint: 'You manage your own account under Settings.'
		},
		confirm: {
			resetPassword: {
				title: 'Reset the password of {username}?',
				text: 'All of their sessions end and they get a new temporary password. The action is recorded in the security log, and they must change the password at their next sign-in.',
				action: 'Reset password'
			},
			resetTwoFactor: {
				title: 'Reset two-factor sign-in for {username}?',
				text: 'All of their sessions end, and they have to turn on two-factor sign-in again.',
				action: 'Reset 2FA'
			},
			deactivate: {
				title: 'Deactivate the account of {username}?',
				text: 'All of their sessions end. Their shares stay in effect.',
				action: 'Deactivate account'
			},
			activate: {
				title: 'Activate the account of {username}?',
				text: 'They can sign in again with their existing password.',
				action: 'Activate account'
			},
			delete: {
				title: 'Delete the account of {username}?',
				text: 'Their links, tags and shares are deleted for good; in the audit log their name is replaced by a pseudonym. This cannot be undone.',
				sharedTitle: {
					one: '{count} shared link also disappears for its recipients.',
					other: '{count} shared links also disappear for their recipients.'
				},
				sharedText: 'If the recipients still need them, they should save them first.',
				action: 'Delete user'
			}
		},
		toast: {
			created: '{username} created',
			updated: '{username} updated',
			passwordReset: 'Password of {username} reset',
			twoFactorReset: 'Two-factor sign-in of {username} reset',
			unlocked: '{username} unlocked',
			sessionsRevoked: {
				zero: 'There were no sessions to revoke',
				one: '{count} session revoked',
				other: '{count} sessions revoked'
			},
			deactivated: '{username} deactivated',
			activated: '{username} activated',
			deleted: '{username} deleted'
		},
		empty: {
			title: 'No user matches the filters.',
			text: 'Change or clear the filters.'
		}
	},
	pager: {
		label: 'Pagination',
		previous: 'Previous page',
		next: 'Next page',
		page: 'Page {page}',
		range: '{from}–{to} / {total}'
	},
	userDialog: {
		createTitle: 'Add user',
		createText: 'The new user sets their own password at the first sign-in.',
		editTitle: 'Edit {username}',
		editText: 'Changes to the role and the active state are recorded in the security log.',
		fields: {
			username: 'Username',
			displayName: 'Display name',
			role: 'Role',
			locale: 'Language',
			password: 'Password',
			active: 'Active'
		},
		usernameHelp: '3–64 characters: letters, digits, dot, hyphen, underscore.',
		displayNameHelp: 'Others see this name on shared links. At most 100 characters.',
		localeHelp: 'The interface language at the first sign-in. They can change it later.',
		passwordMode: {
			label: 'Password choice',
			generated: 'Generated password',
			manual: 'Set it myself'
		},
		generatedHelp: 'The server creates a 16-character password and shows it only once.',
		passwordHelp:
			'At least {min} characters, at least {distinct} different characters, without the username.',
		activeHelp: 'An inactive account cannot sign in.',
		selfHint: 'You manage your own account under Settings.',
		create: 'Create user',
		save: 'Save changes'
	},
	tempPassword: {
		title: 'Temporary password',
		text: 'Temporary password of {username}:',
		copy: 'Copy password',
		copied: 'Password copied',
		warningTitle: 'The password is shown only now',
		warningText:
			'Hand it over through a secure channel. The user must replace it at the first sign-in.',
		done: 'Done'
	},
	settings: {
		pageTitle: 'System settings',
		subtitle: 'Settings that apply to every user.',
		cards: {
			archive: {
				title: 'Archiving',
				text: 'Links that are not opened are archived automatically after this period, unless the user has set their own.'
			},
			sessions: {
				title: 'Sessions',
				text: 'A sign-in expires after the idle time or after the maximum duration.'
			},
			audit: {
				title: 'Audit log',
				text: 'Security events are kept for this long and deleted afterwards.'
			},
			fetch: {
				title: 'Metadata fetching',
				text: 'The server fetches the title and icon of links.'
			},
			privacy: {
				title: 'Privacy notice',
				text: 'These details appear in the privacy notice. Publish a new version after a significant change.'
			}
		},
		fields: {
			defaultArchiveDays: 'Archiving period',
			sessionIdleMinutes: 'Idle time',
			sessionAbsoluteHours: 'Maximum session length',
			auditRetentionDays: 'Retention period',
			allowPrivateFetch: 'Allow internal network addresses',
			privacyControllerName: 'Controller name',
			privacyControllerAddress: 'Controller address',
			privacyControllerEmail: 'Controller e-mail address',
			privacyDpoContact: 'Data protection contact',
			privacySupervisoryAuthority: 'Competent supervisory authority'
		},
		help: {
			archiveDays: '{min}–{max} days. 0: never. Default: {default}.',
			days: '{min}–{max} days. Default: {default}.',
			minutes: '{min}–{max} minutes. Default: {default}.',
			hours: '{min}–{max} hours. Default: {default}.',
			text: 'At most {max} characters.',
			email: 'May be left empty. At most {max} characters.',
			supervisoryAuthority:
				'At most {max} characters. For example: Information Commissioner’s Office, ico.org.uk',
			allowPrivateFetch:
				'For intranet links. When on, the server may also fetch addresses on the local network; cloud metadata addresses stay blocked.'
		},
		units: {
			days: 'days',
			minutes: 'minutes',
			hours: 'hours'
		},
		runArchive: 'Run archiving now',
		archived: {
			zero: 'There were no links to archive',
			one: '{count} link archived',
			other: '{count} links archived'
		},
		locked: {
			title: 'Fixed by an environment variable',
			privateFetch: 'The value of APP_ALLOW_PRIVATE_FETCH decides.',
			generic: 'The server configuration decides.'
		},
		notice: {
			version: 'Current version',
			updatedAt: 'Published',
			bump: 'Publish new version',
			bumpTitle: 'Publish a new version of the notice?',
			bumpText: 'Every user has to accept it again at their next sign-in or request.',
			bumped: 'Version {version} of the notice published'
		},
		unsaved: 'Unsaved changes',
		noChanges: 'No unsaved changes',
		save: 'Save settings',
		discard: 'Discard',
		saved: 'Settings saved',
		leave: {
			title: 'Discard your changes?',
			text: 'The unsaved settings will be lost.',
			action: 'Discard changes'
		}
	}
} satisfies Shape<typeof hu>;
