import type hu from '../hu/audit.js';
import type { Shape } from '../../types.js';

export default {
	pageTitle: 'Audit log',
	subtitle: 'Security events are kept for {days} days.',
	filters: {
		label: 'Filter the audit log',
		event: 'Event',
		all: 'All',
		groupsHeading: 'Event groups',
		eventsHeading: 'Single events',
		groups: {
			auth: 'Sign-in',
			session: 'Sessions',
			user: 'Users',
			settings: 'Settings',
			privacy: 'Privacy',
			link: 'Sharing'
		},
		result: 'Result',
		actor: 'Actor',
		actorSearch: 'Search users…',
		actorEmpty: 'No such user.',
		actorUnavailable: 'The user list cannot be loaded.',
		actorUnknown: 'User #{id}',
		from: 'From',
		to: 'To',
		clear: 'Clear filters'
	},
	table: {
		caption: 'Audit log entries',
		at: 'Time',
		event: 'Event',
		result: 'Result',
		actor: 'Actor',
		target: 'Target',
		ip: 'IP',
		details: 'Details',
		showDetails: 'Show details'
	},
	result: {
		success: 'Succeeded',
		failure: 'Failed'
	},
	empty: {
		title: 'No event matches the filters.',
		text: 'Change or clear the filters.'
	},
	events: {
		bootstrap: {
			admin_created: 'Initial administrator created'
		},
		auth: {
			login: {
				success: 'Signed in',
				failure: 'Sign-in failed',
				locked: 'Sign-in locked'
			},
			logout: 'Signed out',
			recovery_code: {
				used: 'Recovery code used'
			},
			password: {
				changed: 'Password changed'
			},
			'2fa': {
				enabled: 'Two-factor sign-in turned on',
				disabled: 'Two-factor sign-in turned off',
				recovery_regenerated: 'New recovery codes'
			}
		},
		session: {
			revoked: 'Session revoked',
			revoked_others: 'Other sessions revoked'
		},
		user: {
			created: 'User created',
			updated: 'User updated',
			deactivated: 'User deactivated',
			activated: 'User activated',
			deleted: 'User deleted',
			password_reset: 'Password reset',
			'2fa_reset': 'Two-factor sign-in reset',
			unlocked: 'Unlocked',
			sessions_revoked: 'Sessions revoked',
			self_deleted: 'Own account deleted'
		},
		settings: {
			updated: 'Settings updated'
		},
		privacy: {
			notice_bumped: 'New privacy notice version',
			consent_accepted: 'Privacy notice accepted'
		},
		gdpr: {
			export: 'Data export'
		},
		link: {
			shared: 'Link shared',
			unshared: 'Sharing revoked'
		}
	},
	targets: {
		user: 'User',
		link: 'Link',
		session: 'Session',
		settings: 'Settings',
		tag: 'Tag'
	},
	details: {
		keys: {
			reason: 'Reason',
			method: 'Method',
			scope: 'Scope',
			changed: 'Changed fields',
			role: 'Role',
			active: 'Active',
			remaining: 'Remaining codes',
			sessionId: 'Session',
			count: 'Count',
			version: 'Version',
			targetType: 'Target type',
			targetRole: 'Target role',
			targetUserId: 'Target user',
			generatedPassword: 'Generated password',
			via: 'Source'
		},
		values: {
			reason: {
				credentials: 'Wrong password',
				unknown_user: 'Unknown username',
				disabled: 'Inactive account',
				totp: 'Wrong code'
			},
			method: {
				password: 'Password',
				totp: 'Authentication code',
				recovery_code: 'Recovery code'
			},
			scope: {
				user: 'User',
				ip: 'IP address'
			},
			targetType: {
				all: 'Everyone',
				role: 'Role',
				user: 'User'
			},
			via: {
				cli: 'Command line'
			}
		},
		yes: 'Yes',
		no: 'No',
		change: '{from} → {to}'
	}
} satisfies Shape<typeof hu>;
