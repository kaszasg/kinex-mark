import type hu from '../hu/privacy.js';
import type { Shape } from '../../types.js';

export default {
	title: 'Privacy notice',
	version: 'Version {version}',
	updatedAt: 'Updated: {date}',
	backToApp: 'Back to the app',
	controller: {
		name: 'Name',
		address: 'Address',
		email: 'Email',
		dpo: 'Data protection contact'
	},
	sections: {
		controller: {
			title: 'Controller',
			body: 'The controller is the organisation that operates Kinex Mark. You can reach it with data protection questions at the contact details below.'
		},
		data: {
			title: 'Data processed',
			body: 'Username, display name, password hash, encrypted two-factor secret, sessions with IP address and browser identifier, links, tags, shares, usage statistics, settings, and a security log with IP addresses. The system does not process email addresses, phone numbers or tracking data.'
		},
		purposes: {
			title: 'Purposes',
			body: 'Providing the link collection service, and keeping the account and the system secure (sign-in throttling, security log).'
		},
		legalBasis: {
			title: 'Legal bases',
			body: 'Processing the account, links, tags, shares and settings is necessary to provide the service (GDPR Article 6(1)(b)). The security log, sign-in throttling and the IP address of sessions rest on the operator’s legitimate interest in a secure system (Article 6(1)(f)). Accepting this notice confirms that you have read it; it is not consent.'
		},
		recipients: {
			title: 'Recipients',
			body: 'Users you share a link with see the link, your username and your display name. Developers and administrators can find every active user by username, display name and role when they choose a share recipient. The operator’s administrators see the data needed for user management and the security log. The operator’s hosting provider accesses the data as a processor.'
		},
		transfers: {
			title: 'Transfers to third countries',
			body: 'No data is transferred to a third country or an international organisation.'
		},
		automatedDecisions: {
			title: 'Automated decision-making',
			body: 'There is no automated decision-making or profiling. Automatic archiving only affects how the link list is ordered.'
		},
		retention: {
			title: 'Retention',
			body: 'Sessions end after {hours} at most, or after {minutes} of inactivity. Security log entries are deleted after {days}. All other data is kept until the account is deleted.'
		},
		storage: {
			title: 'Cookies and local storage',
			body: 'The system uses a single cookie that sign-in requires (kxm_session or __Host-kxm_session). Your browser’s local storage keeps the chosen theme (mode-watcher-mode, mode-watcher-theme) and language (kxm-locale). There is no tracking, no analytics and no content loaded from third parties.'
		},
		fetching: {
			title: 'Metadata fetching',
			body: 'The server fetches the title and icon of your links. Your browser therefore does not contact a third party until you open a link.'
		},
		adminAccess: {
			title: 'Administrator access',
			body: 'An administrator can reset your password and thereby sign in to your account. This is recorded in the security log, appears in your own data export, and you have to set a new password at your next sign-in.'
		},
		rights: {
			title: 'Your rights',
			body: 'Under Articles 15 to 21 of the GDPR you have the following rights:'
		}
	},
	rights: {
		access:
			'Access: on the Settings › Privacy tab, the Download data button gives you all your data.',
		portability:
			'Portability: the same export is a machine-readable JSON file, and you can export your links as a browser bookmark file on the Links page.',
		rectification:
			'Rectification: edit your display name on the Settings › Profile tab, and your links and tags directly.',
		erasure:
			'Erasure: delete your account on the Settings › Privacy tab; all your links, tags and shares are deleted with it.',
		restriction:
			'Restriction of processing: ask the controller at the contact details above. An administrator can deactivate your account; your data is then kept, but its processing pauses.',
		objection:
			'Objection: you can object to processing based on legitimate interest at the contact details above.',
		complaint: 'Complaint: you can lodge a complaint with the supervisory authority: {authority}.',
		defaultAuthority: 'the data protection authority where you live'
	},
	consent: {
		title: 'Privacy notice',
		text: 'Read how Kinex Mark handles your data. To continue, confirm that you have read it.',
		openFull: 'Open full page',
		newTab: 'opens in a new tab',
		checkbox: 'I have read the privacy notice and take note of it.',
		submit: 'Confirm and continue'
	}
} satisfies Shape<typeof hu>;
