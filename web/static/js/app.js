/* ============================================
   CasaDrop - Frontend Application
   Premium SPA with i18n, auth, upload, shares,
   receive links, settings, and user management
   ============================================ */

(function () {
    'use strict';

    // ==========================================
    // Theme (apply before render to prevent flash)
    // ==========================================
    const THEME = localStorage.getItem('casadrop_theme') || 'light';
    document.documentElement.setAttribute('data-theme', THEME);

    // ==========================================
    // i18n
    // ==========================================
    const SUPPORTED_LANGS = ['en','de','fr','es','it','pt','nl','pl','ru','ja','zh','ko','tr','ar'];
    const LANG = (() => {
        const stored = localStorage.getItem('casadrop_lang');
        if (stored && SUPPORTED_LANGS.includes(stored)) return stored;
        const nav = (navigator.language || navigator.languages?.[0] || 'en').toLowerCase();
        const exact = SUPPORTED_LANGS.find(l => nav === l || nav.startsWith(l + '-'));
        return exact || 'en';
    })();

    const I18N = {
        en: {
            'common.breadcrumb': 'Breadcrumb',
            'common.copy': 'Copy',
            'common.copyLink': 'Copy link',
            'common.folder': 'Folder',
            'common.expiring': 'Expiring',
            'common.autoShare': 'Auto-share',
            'common.qr': 'QR code',
            'common.edit': 'Edit',
            'common.delete': 'Delete',
            'common.remove': 'Remove',
            'common.close': 'Close',
            'common.refresh': 'Refresh',
            'nav.upload': 'Upload',
            'nav.hostshare': 'Share from host',
            'nav.shares': 'Shares',
            'nav.receive': 'Receive',
            'nav.settings': 'Settings',
            'nav.logout': 'Logout',
            'hostshare.title': 'Share from host',
            'hostshare.hint': 'Share files and folders that already live on the server — no re-upload.',
            'hostshare.home': 'Home',
            'hostshare.empty': 'This folder is empty',
            'hostshare.error': 'Could not access this location',
            'hostshare.folder': 'Folder',
            'hostshare.share': 'Share',
            'hostshare.dialogTitle': 'Share this item',
            'hostshare.symlink': 'Link instead of copy (no extra disk space)',
            'hostshare.success': 'Share created',
            'hostshare.done': 'Done',
            'login.subtitle': 'Self-hosted file sharing',
            'login.password': 'Password',
            'login.submit': 'Sign In',
            'login.or': 'or',
            'login.sso': 'Login with SSO',
            'upload.title': 'Upload Files',
            'upload.dropText': 'Drag & drop files here or click to browse',
            'upload.hint': 'Multiple files supported',
            'upload.options': 'Upload Options',
            'upload.password': 'Password (optional)',
            'upload.expiry': 'Expires in',
            'upload.expiry.1h': '1 hour',
            'upload.expiry.6h': '6 hours',
            'upload.expiry.12h': '12 hours',
            'upload.expiry.1d': '1 day',
            'upload.expiry.3d': '3 days',
            'upload.expiry.7d': '7 days',
            'upload.expiry.14d': '14 days',
            'upload.expiry.30d': '30 days',
            'upload.expiry.never': 'Never (unlimited)',
            'upload.expiry.hours': 'hours',
            'upload.maxDownloads': 'Max downloads (0 = unlimited)',
            'upload.submit': 'Upload',
            'upload.uploading': 'Uploading...',
            'upload.complete': 'Upload complete',
            'upload.error': 'Upload failed',
            'upload.networkError': 'Server unreachable',
            'upload.another': 'Upload More',
            'shares.title': 'Active Shares',
            'shares.empty': 'No active shares yet',
            'shares.downloads': 'downloads',
            'shares.expired': 'expired',
            'shares.never': 'never',
            'shares.unlimited': 'unlimited',
            'shares.expiryRequired': 'Please enter an expiry in hours, or switch "unlimited" on.',
            'shares.deleteConfirm': 'Delete this share?',
            'shares.copied': 'Link copied!',
            'shares.size': 'size',
            'shares.deleted': 'Share deleted',
            'shares.edit': 'Edit Share',
            'shares.save': 'Save',
            'shares.updated': 'Share updated',
            'shares.changePassword': 'change or clear',
            'shares.addPassword': 'add',
            'receive.title': 'Receive Links',
            'receive.create': 'New Link',
            'receive.formTitle': 'Create Receive Link',
            'receive.name': 'Name',
            'receive.password': 'Password (optional)',
            'receive.maxUploads': 'Max uploads (0 = unlimited)',
            'receive.maxFileSize': 'Max file size (MB, 0 = unlimited)',
            'receive.expiry': 'Expires in (hours, 0 = never)',
            'receive.extensions': 'Allowed extensions (e.g. .pdf,.doc)',
            'receive.autoShare': 'Auto-share uploaded files',
            'receive.submit': 'Create',
            'receive.cancel': 'Cancel',
            'receive.empty': 'No receive links yet',
            'receive.uploads': 'uploads',
            'receive.deleteConfirm': 'Delete this receive link?',
            'receive.created': 'Receive link created',
            'receive.deleted': 'Receive link deleted',
            'settings.title': 'Settings',
            'settings.network': 'Network Configuration',
            'settings.webhook': 'Webhook',
            'settings.users': 'User Management',
            'settings.primary': 'Primary',
            'settings.detected': 'Detected',
            'settings.notDetected': 'Not detected',
            'settings.customHint': 'Enter your own fixed domain (e.g. reverse proxy or a Cloudflare named-tunnel domain).',
            'settings.enabled': 'Enabled',
            'settings.useDetected': 'Use detected',
            'settings.save': 'Save',
            'settings.saved': 'Settings saved',
            'settings.cfRotating': 'Generating a fresh Cloudflare share URL…',
            'settings.cfRotated': 'New Cloudflare URL ready:',
            'settings.cfRotateTimeout': 'No new Cloudflare URL appeared — is the tunnel container running?',
            'settings.testWebhook': 'Test Webhook',
            'settings.webhookUrl': 'Webhook URL',
            'settings.webhookSecret': 'Webhook Secret',
            'settings.webhookEnabled': 'Webhook enabled',
            'settings.webhookEvents': 'Notify on',
            'settings.webhookOnDownload': 'File downloaded',
            'settings.webhookOnLimit': 'Download limit reached',
            'settings.webhookOnExpire': 'Share expired',
            'settings.webhookSecretKeep': 'unchanged — leave blank to keep',
            'settings.webhookSecretNone': 'no secret set',
            'settings.webhookSecretClear': 'Remove the stored secret',
            'settings.webhookNeedsUrl': 'Enter a webhook URL or switch the webhook off',
            'settings.webhookSent': 'Webhook sent',
            'settings.createUser': 'Create User',
            'settings.quota': 'Storage / Quota',
            'settings.quotaGB': 'Quota (GB, 0 = unlimited)',
            'settings.unlimited': 'unlimited',
            'settings.quotaUpdated': 'Quota updated',
            'settings.email': 'Email',
            'settings.name': 'Name',
            'settings.role': 'Role',
            'settings.userPassword': 'Password',
            'settings.userPasswordHint': 'Optional — leave empty for SSO-only accounts (min. 8 characters if set)',
            'settings.passwordTooShort': 'Password must be at least 8 characters',
            'settings.deleteUserConfirm': 'Delete this user?',
            'settings.userCreated': 'User created',
            'settings.userDeleted': 'User deleted',
            'settings.fillAll': 'Fill in all fields',
            'settings.maxFileSize': 'Max file size',
            'settings.fileRestrictions': 'File Restrictions',
            'settings.fileRestrictionsHint': 'Control which file types can be uploaded. Blocked extensions are rejected, allowed extensions create a whitelist (empty = all allowed except blocked).',
            'settings.blockedExtensions': 'Blocked extensions',
            'settings.allowedExtensions': 'Allowed extensions (whitelist)',
            'settings.allowedExtensionsHint': 'Empty = all allowed (except blocked)',
            'toast.copied': 'Link copied to clipboard',
            'toast.error': 'Something went wrong',
            'stat.totalShares': 'Total Shares',
            'stat.totalDownloads': 'Downloads',
            'stat.totalSize': 'Total Size',
            'stat.protected': 'Protected',
            'shares.sendEmail': 'Send via Email',
            'shares.taildrop': 'Send to device (Taildrop)',
            'taildrop.title': 'Send to device',
            'taildrop.device': 'Target device',
            'taildrop.send': 'Send',
            'taildrop.sending': 'Sending…',
            'taildrop.sent': 'File sent via Taildrop',
            'taildrop.noDevices': 'No Tailscale devices available',
            'shares.selectAll': 'Select All',
            'shares.bulkDelete': 'Delete Selected',
            'shares.selectedCount': '{n} selected',
            'email.recipientEmail': 'Recipient Email',
            'email.recipientName': 'Recipient Name',
            'email.message': 'Message',
            'email.optional': 'Optional',
            'email.send': 'Send',
            'email.sent': 'Email sent successfully',
            'email.enterRecipient': 'Enter recipient email',
            'email.sendFailed': 'Email could not be sent:',
            'upload.folder': 'Upload Folder',
            'settings.apiKeys': 'API Keys',
            'settings.apiKeysHint': 'API keys allow programmatic access to CasaDrop. Use the X-API-Key header.',
            'settings.noApiKeys': 'No API keys yet',
            'settings.createApiKey': 'Create Key',
            'settings.apiKeyCreated': 'API Key Created — Copy it now!',
            'settings.apiKeyOnlyOnce': 'This key will only be shown once. Store it securely.',
            'settings.deleteApiKeyConfirm': 'Delete this API key?',
            'settings.apiKeyDeleted': 'API key deleted',
            'settings.smtp': 'Email / SMTP',
            'settings.smtpEnabled': 'Enable email sending',
            'settings.testSmtp': 'Test Connection',
            'settings.smtpTestOk': 'SMTP connection successful',
            'settings.twofa': 'Two-Factor Authentication',
            'settings.twofaHint': 'Add a time-based one-time password (TOTP) to your admin login. Use an authenticator app like Aegis, Google Authenticator or 1Password.',
            'settings.twofaStatus': 'Status',
            'settings.twofaEnabled': 'Enabled',
            'settings.twofaDisabled': 'Disabled',
            'settings.twofaEnable': 'Enable 2FA',
            'settings.twofaDisable': 'Disable 2FA',
            'settings.twofaVerifyEnable': 'Verify & Enable',
            'settings.twofaCancel': 'Cancel',
            'settings.twofaScan': 'Scan this QR code with your authenticator app:',
            'settings.twofaManual': 'Or enter this secret manually:',
            'settings.twofaCode': 'Enter the 6-digit code',
            'settings.twofaEnabledMsg': 'Two-factor authentication enabled',
            'settings.twofaDisabledMsg': 'Two-factor authentication disabled',
            'settings.twofaCodeRequired': 'Please enter the 6-digit code',
            'settings.twofaSetupFailed': 'Could not start 2FA setup',
            'settings.sessions': 'Active Sessions',
            'sessions.hint': 'Every device signed in to your account. End a session you no longer recognise.',
            'sessions.hintAdmin': 'Every active session on this server. End one you no longer recognise.',
            'sessions.current': 'This device',
            'sessions.since': 'Signed in',
            'sessions.expires': 'Expires',
            'sessions.revoke': 'End session',
            'sessions.revokeOthers': 'Sign out all other devices',
            'sessions.revokedOthers': 'Signed out {n} other device(s)',
            'sessions.revoked': 'Session ended',
            'sessions.empty': 'No other active sessions.',
            'sessions.confirmOthers': 'Sign out of every other device?',
            'settings.activity': 'Activity log',
            'settings.activityHint': 'Who downloaded, uploaded, signed in — with address and time. Entries older than the retention period are removed automatically.',
            'activity.when': 'When',
            'activity.kind': 'Event',
            'activity.who': 'Who',
            'activity.share': 'Share',
            'activity.from': 'From',
            'activity.detail': 'Detail',
            'activity.empty': 'Nothing recorded yet.',
            'activity.export': 'Export CSV',
            'activity.allKinds': 'All events',
            'activity.loadMore': 'Load more',
            'activity.anonymous': 'anonymous',
            'activity.system': 'system',
            'activity.forShare': 'Activity',
            'activity.loadFailed': 'Could not load the activity log',
            'activity.kind.share.created': 'Share created',
            'activity.kind.share.updated': 'Share updated',
            'activity.kind.share.deleted': 'Share deleted',
            'activity.kind.share.expired': 'Share expired',
            'activity.kind.share.downloaded': 'Downloaded',
            'activity.kind.share.streamed': 'Streamed',
            'activity.kind.receive.uploaded': 'File received',
            'activity.kind.auth.login': 'Signed in',
            'activity.kind.auth.login_failed': 'Sign-in failed',
            'activity.kind.auth.locked': 'Locked out',
            'activity.kind.auth.logout': 'Signed out',
            'activity.kind.auth.setup': 'Setup / 2FA change',
            'activity.kind.session.revoked': 'Session revoked',
            'activity.kind.security': 'Security',
            'theme.light': 'Light mode',
            'theme.dark': 'Dark mode',
            'load.networkFailed': 'Could not load network settings',
            'load.restrictionsFailed': 'Could not load file restrictions',
            'load.webhookFailed': 'Could not load webhook settings',
            'load.usersUnavailable': 'User management is not available',
            'load.twoFAFailed': 'Could not load 2FA settings',
            'load.apiKeysFailed': 'Could not load API keys',
            'load.smtpFailed': 'Could not load SMTP settings',
            'role.admin': 'Admin',
            'role.user': 'User',
            'role.viewer': 'Viewer',
            'settings.apiKeyNamePlaceholder': 'My API key',
            'common.copied': 'Copied!',
            'smtp.host': 'SMTP host',
            'smtp.port': 'Port',
            'smtp.encryption': 'Encryption',
            'smtp.encryptionNone': 'None',
            'smtp.username': 'Username',
            'smtp.password': 'Password',
            'smtp.fromEmail': 'Sender email',
            'smtp.fromName': 'Sender name',
            'auth.signInAgain': 'Please sign in again',
            'shares.expiresWhen': 'Expires {when}',
            'activity.d.loginOk': 'Signed in ({role})',
            'activity.d.loginOkApi': 'Signed in via API ({role})',
            'activity.d.loginFailed': 'Failed sign-in attempt {n}/{max}',
            'activity.d.loginFailedApi': 'Failed API sign-in attempt {n}/{max}',
            'activity.d.loginLocked': 'Locked after too many failed attempts',
            'activity.d.loginLockedApi': 'Locked after too many failed API attempts',
            'activity.d.twoFAInvalid': '2FA code missing or invalid',
            'activity.d.twoFAInvalidApi': '2FA code missing or invalid (API)',
            'activity.d.localAuthDisabled': 'Password sign-in disabled (SSO only)',
            'activity.d.csrfInvalid': 'Invalid security token on sign-in',
            'activity.d.rateLimited': 'Too many sign-in attempts',
            'activity.d.loggedOut': 'Signed out',
            'activity.d.setupTokenInvalid': 'Setup rejected: invalid setup token',
            'activity.d.setupDone': 'Initial setup completed',
            'activity.d.twoFAEnabled': '2FA enabled',
            'activity.d.twoFADisabled': '2FA disabled',
            'activity.d.revokedAll': 'All other sessions ended',
            'activity.d.revokedOne': 'Session {id} ended ({email})',
            'activity.d.bulk': 'bulk deletion',
        },
        de: {
            'common.breadcrumb': 'Navigationspfad',
            'common.copy': 'Kopieren',
            'common.copyLink': 'Link kopieren',
            'common.folder': 'Ordner',
            'common.expiring': 'Läuft bald ab',
            'common.autoShare': 'Auto-Freigabe',
            'common.qr': 'QR-Code',
            'common.edit': 'Bearbeiten',
            'common.delete': 'Löschen',
            'common.remove': 'Entfernen',
            'common.close': 'Schließen',
            'common.refresh': 'Aktualisieren',
            'nav.upload': 'Hochladen',
            'nav.hostshare': 'Vom Host teilen',
            'nav.shares': 'Freigaben',
            'nav.receive': 'Empfangen',
            'hostshare.title': 'Vom Host teilen',
            'hostshare.hint': 'Teile Dateien und Ordner, die schon auf dem Server liegen — ohne erneutes Hochladen.',
            'hostshare.home': 'Start',
            'hostshare.empty': 'Dieser Ordner ist leer',
            'hostshare.error': 'Zugriff auf diesen Ort nicht möglich',
            'hostshare.folder': 'Ordner',
            'hostshare.share': 'Teilen',
            'hostshare.dialogTitle': 'Diesen Eintrag teilen',
            'hostshare.symlink': 'Verknüpfen statt kopieren (kein zusätzlicher Speicher)',
            'hostshare.success': 'Freigabe erstellt',
            'hostshare.done': 'Fertig',
            'nav.settings': 'Einstellungen',
            'nav.logout': 'Abmelden',
            'login.subtitle': 'Selbst-gehostetes Filesharing',
            'login.password': 'Passwort',
            'login.submit': 'Anmelden',
            'login.or': 'oder',
            'login.sso': 'Mit SSO anmelden',
            'upload.title': 'Dateien hochladen',
            'upload.dropText': 'Dateien hierher ziehen oder klicken',
            'upload.hint': 'Mehrere Dateien möglich',
            'upload.options': 'Upload-Optionen',
            'upload.password': 'Passwort (optional)',
            'upload.expiry': 'Läuft ab in',
            'upload.expiry.1h': '1 Stunde',
            'upload.expiry.6h': '6 Stunden',
            'upload.expiry.12h': '12 Stunden',
            'upload.expiry.1d': '1 Tag',
            'upload.expiry.3d': '3 Tage',
            'upload.expiry.7d': '7 Tage',
            'upload.expiry.14d': '14 Tage',
            'upload.expiry.30d': '30 Tage',
            'upload.expiry.never': 'Unbegrenzt',
            'upload.expiry.hours': 'Stunden',
            'upload.maxDownloads': 'Max Downloads (0 = unbegrenzt)',
            'upload.submit': 'Hochladen',
            'upload.uploading': 'Wird hochgeladen...',
            'upload.complete': 'Upload abgeschlossen',
            'upload.error': 'Upload fehlgeschlagen',
            'upload.networkError': 'Server nicht erreichbar',
            'upload.another': 'Weitere hochladen',
            'shares.title': 'Aktive Freigaben',
            'shares.empty': 'Noch keine Freigaben',
            'shares.downloads': 'Downloads',
            'shares.expired': 'abgelaufen',
            'shares.never': 'nie',
            'shares.unlimited': 'unbegrenzt',
            'shares.expiryRequired': 'Bitte eine Ablaufzeit in Stunden eintragen oder "Unbegrenzt" einschalten.',
            'shares.deleteConfirm': 'Freigabe löschen?',
            'shares.copied': 'Link kopiert!',
            'shares.size': 'Größe',
            'shares.deleted': 'Freigabe gelöscht',
            'shares.edit': 'Freigabe bearbeiten',
            'shares.save': 'Speichern',
            'shares.updated': 'Freigabe aktualisiert',
            'shares.changePassword': 'ändern oder entfernen',
            'shares.addPassword': 'hinzufügen',
            'receive.title': 'Empfangslinks',
            'receive.create': 'Neuer Link',
            'receive.formTitle': 'Empfangslink erstellen',
            'receive.name': 'Name',
            'receive.password': 'Passwort (optional)',
            'receive.maxUploads': 'Max Uploads (0 = unbegrenzt)',
            'receive.maxFileSize': 'Max. Dateigröße (MB, 0 = unbegrenzt)',
            'receive.expiry': 'Ablauf in (Stunden, 0 = nie)',
            'receive.extensions': 'Erlaubte Endungen (z.B. .pdf,.doc)',
            'receive.autoShare': 'Hochgeladene Dateien automatisch teilen',
            'receive.submit': 'Erstellen',
            'receive.cancel': 'Abbrechen',
            'receive.empty': 'Noch keine Empfangslinks',
            'receive.uploads': 'Uploads',
            'receive.deleteConfirm': 'Empfangslink löschen?',
            'receive.created': 'Empfangslink erstellt',
            'receive.deleted': 'Empfangslink gelöscht',
            'settings.title': 'Einstellungen',
            'settings.network': 'Netzwerkkonfiguration',
            'settings.webhook': 'Webhook',
            'settings.users': 'Benutzerverwaltung',
            'settings.primary': 'Primär',
            'settings.detected': 'Erkannt',
            'settings.notDetected': 'Nicht erkannt',
            'settings.customHint': 'Eigene feste Domain eintragen (z. B. Reverse-Proxy oder feste Cloudflare-Tunnel-Domain).',
            'settings.enabled': 'Aktiviert',
            'settings.useDetected': 'Erkannten Wert verwenden',
            'settings.save': 'Speichern',
            'settings.saved': 'Einstellungen gespeichert',
            'settings.cfRotating': 'Neue Cloudflare-Freigabe-URL wird erzeugt…',
            'settings.cfRotated': 'Neue Cloudflare-URL bereit:',
            'settings.cfRotateTimeout': 'Keine neue Cloudflare-URL erschienen — läuft der Tunnel-Container?',
            'settings.testWebhook': 'Webhook testen',
            'settings.webhookUrl': 'Webhook URL',
            'settings.webhookSecret': 'Webhook Secret',
            'settings.webhookEnabled': 'Webhook aktiviert',
            'settings.webhookEvents': 'Benachrichtigen bei',
            'settings.webhookOnDownload': 'Datei heruntergeladen',
            'settings.webhookOnLimit': 'Download-Limit erreicht',
            'settings.webhookOnExpire': 'Freigabe abgelaufen',
            'settings.webhookSecretKeep': 'unverändert — leer lassen zum Beibehalten',
            'settings.webhookSecretNone': 'kein Secret gesetzt',
            'settings.webhookSecretClear': 'Gespeichertes Secret entfernen',
            'settings.webhookNeedsUrl': 'Webhook-URL eingeben oder Webhook deaktivieren',
            'settings.webhookSent': 'Webhook gesendet',
            'settings.createUser': 'Benutzer erstellen',
            'settings.quota': 'Speicher / Quota',
            'settings.quotaGB': 'Quota (GB, 0 = unbegrenzt)',
            'settings.unlimited': 'unbegrenzt',
            'settings.quotaUpdated': 'Quota aktualisiert',
            'settings.email': 'E-Mail',
            'settings.name': 'Name',
            'settings.role': 'Rolle',
            'settings.userPassword': 'Passwort',
            'settings.userPasswordHint': 'Optional — für reine SSO-Konten leer lassen (min. 8 Zeichen, falls gesetzt)',
            'settings.passwordTooShort': 'Passwort muss mindestens 8 Zeichen haben',
            'settings.deleteUserConfirm': 'Benutzer löschen?',
            'settings.userCreated': 'Benutzer erstellt',
            'settings.userDeleted': 'Benutzer gelöscht',
            'settings.fillAll': 'Alle Felder ausfüllen',
            'settings.maxFileSize': 'Max. Dateigröße',
            'settings.fileRestrictions': 'Dateibeschränkungen',
            'settings.fileRestrictionsHint': 'Steuere welche Dateitypen hochgeladen werden dürfen. Blockierte Endungen werden abgelehnt, erlaubte Endungen erstellen eine Whitelist (leer = alle erlaubt außer blockierte).',
            'settings.blockedExtensions': 'Blockierte Endungen',
            'settings.allowedExtensions': 'Erlaubte Endungen (Whitelist)',
            'settings.allowedExtensionsHint': 'Leer = alle erlaubt (außer blockierte)',
            'toast.copied': 'Link in Zwischenablage kopiert',
            'toast.error': 'Etwas ist schiefgelaufen',
            'stat.totalShares': 'Freigaben',
            'stat.totalDownloads': 'Downloads',
            'stat.totalSize': 'Gesamtgröße',
            'stat.protected': 'Geschützt',
            'shares.sendEmail': 'Per E-Mail senden',
            'shares.taildrop': 'An Gerät senden (Taildrop)',
            'taildrop.title': 'An Gerät senden',
            'taildrop.device': 'Zielgerät',
            'taildrop.send': 'Senden',
            'taildrop.sending': 'Senden…',
            'taildrop.sent': 'Datei via Taildrop gesendet',
            'taildrop.noDevices': 'Keine Tailscale-Geräte verfügbar',
            'shares.selectAll': 'Alle auswählen',
            'shares.bulkDelete': 'Ausgewählte löschen',
            'shares.selectedCount': '{n} ausgewählt',
            'email.recipientEmail': 'Empfänger E-Mail',
            'email.recipientName': 'Empfängername',
            'email.message': 'Nachricht',
            'email.optional': 'Optional',
            'email.send': 'Senden',
            'email.sent': 'E-Mail erfolgreich gesendet',
            'email.enterRecipient': 'Empfänger E-Mail eingeben',
            'email.sendFailed': 'E-Mail konnte nicht gesendet werden:',
            'upload.folder': 'Ordner hochladen',
            'settings.apiKeys': 'API-Schlüssel',
            'settings.apiKeysHint': 'API-Schlüssel ermöglichen programmatischen Zugriff. Verwende den X-API-Key Header.',
            'settings.noApiKeys': 'Noch keine API-Schlüssel',
            'settings.createApiKey': 'Schlüssel erstellen',
            'settings.apiKeyCreated': 'API-Schlüssel erstellt — Jetzt kopieren!',
            'settings.apiKeyOnlyOnce': 'Dieser Schlüssel wird nur einmal angezeigt. Sicher aufbewahren.',
            'settings.deleteApiKeyConfirm': 'API-Schlüssel löschen?',
            'settings.apiKeyDeleted': 'API-Schlüssel gelöscht',
            'settings.smtp': 'E-Mail / SMTP',
            'settings.smtpEnabled': 'E-Mail-Versand aktivieren',
            'settings.testSmtp': 'Verbindung testen',
            'settings.smtpTestOk': 'SMTP-Verbindung erfolgreich',
            'settings.twofa': 'Zwei-Faktor-Authentifizierung',
            'settings.twofaHint': 'Ergänze deinen Admin-Login um ein zeitbasiertes Einmalpasswort (TOTP). Verwende eine Authenticator-App wie Aegis, Google Authenticator oder 1Password.',
            'settings.twofaStatus': 'Status',
            'settings.twofaEnabled': 'Aktiviert',
            'settings.twofaDisabled': 'Deaktiviert',
            'settings.twofaEnable': '2FA aktivieren',
            'settings.twofaDisable': '2FA deaktivieren',
            'settings.twofaVerifyEnable': 'Prüfen & Aktivieren',
            'settings.twofaCancel': 'Abbrechen',
            'settings.twofaScan': 'Scanne diesen QR-Code mit deiner Authenticator-App:',
            'settings.twofaManual': 'Oder gib dieses Geheimnis manuell ein:',
            'settings.twofaCode': '6-stelligen Code eingeben',
            'settings.twofaEnabledMsg': 'Zwei-Faktor-Authentifizierung aktiviert',
            'settings.twofaDisabledMsg': 'Zwei-Faktor-Authentifizierung deaktiviert',
            'settings.twofaCodeRequired': 'Bitte gib den 6-stelligen Code ein',
            'settings.twofaSetupFailed': '2FA-Einrichtung konnte nicht gestartet werden',
            'settings.sessions': 'Aktive Sitzungen',
            'sessions.hint': 'Alle bei deinem Konto angemeldeten Geräte. Beende eine Sitzung, die du nicht wiedererkennst.',
            'sessions.hintAdmin': 'Alle aktiven Sitzungen auf diesem Server. Beende eine, die du nicht wiedererkennst.',
            'sessions.current': 'Dieses Gerät',
            'sessions.since': 'Angemeldet',
            'sessions.expires': 'Läuft ab',
            'sessions.revoke': 'Sitzung beenden',
            'sessions.revokeOthers': 'Alle anderen Geräte abmelden',
            'sessions.revokedOthers': '{n} andere(s) Gerät(e) abgemeldet',
            'sessions.revoked': 'Sitzung beendet',
            'sessions.empty': 'Keine weiteren aktiven Sitzungen.',
            'sessions.confirmOthers': 'Von allen anderen Geräten abmelden?',
            'settings.activity': 'Aktivitätsprotokoll',
            'settings.activityHint': 'Wer hat heruntergeladen, hochgeladen, sich angemeldet — mit Adresse und Zeit. Einträge, die älter als die Aufbewahrungsfrist sind, werden automatisch entfernt.',
            'activity.when': 'Wann',
            'activity.kind': 'Ereignis',
            'activity.who': 'Wer',
            'activity.share': 'Freigabe',
            'activity.from': 'Von',
            'activity.detail': 'Detail',
            'activity.empty': 'Noch nichts aufgezeichnet.',
            'activity.export': 'CSV exportieren',
            'activity.allKinds': 'Alle Ereignisse',
            'activity.loadMore': 'Mehr laden',
            'activity.anonymous': 'anonym',
            'activity.system': 'System',
            'activity.forShare': 'Aktivität',
            'activity.loadFailed': 'Aktivitätsprotokoll konnte nicht geladen werden',
            'activity.kind.share.created': 'Freigabe erstellt',
            'activity.kind.share.updated': 'Freigabe geändert',
            'activity.kind.share.deleted': 'Freigabe gelöscht',
            'activity.kind.share.expired': 'Freigabe abgelaufen',
            'activity.kind.share.downloaded': 'Heruntergeladen',
            'activity.kind.share.streamed': 'Gestreamt',
            'activity.kind.receive.uploaded': 'Datei empfangen',
            'activity.kind.auth.login': 'Angemeldet',
            'activity.kind.auth.login_failed': 'Anmeldung fehlgeschlagen',
            'activity.kind.auth.locked': 'Gesperrt',
            'activity.kind.auth.logout': 'Abgemeldet',
            'activity.kind.auth.setup': 'Einrichtung / 2FA geändert',
            'activity.kind.session.revoked': 'Sitzung beendet',
            'activity.kind.security': 'Sicherheit',
            'theme.light': 'Heller Modus',
            'theme.dark': 'Dunkler Modus',
            'load.networkFailed': 'Netzwerkeinstellungen konnten nicht geladen werden',
            'load.restrictionsFailed': 'Dateibeschränkungen konnten nicht geladen werden',
            'load.webhookFailed': 'Webhook-Einstellungen konnten nicht geladen werden',
            'load.usersUnavailable': 'Benutzerverwaltung ist nicht verfügbar',
            'load.twoFAFailed': '2FA-Einstellungen konnten nicht geladen werden',
            'load.apiKeysFailed': 'API-Schlüssel konnten nicht geladen werden',
            'load.smtpFailed': 'SMTP-Einstellungen konnten nicht geladen werden',
            'role.admin': 'Admin',
            'role.user': 'Benutzer',
            'role.viewer': 'Betrachter',
            'settings.apiKeyNamePlaceholder': 'Mein API-Schlüssel',
            'common.copied': 'Kopiert!',
            'smtp.host': 'SMTP-Server',
            'smtp.port': 'Port',
            'smtp.encryption': 'Verschlüsselung',
            'smtp.encryptionNone': 'Keine',
            'smtp.username': 'Benutzername',
            'smtp.password': 'Passwort',
            'smtp.fromEmail': 'Absender-E-Mail',
            'smtp.fromName': 'Absendername',
            'auth.signInAgain': 'Bitte erneut anmelden',
            'shares.expiresWhen': 'Läuft ab {when}',
            'activity.d.loginOk': 'Angemeldet ({role})',
            'activity.d.loginOkApi': 'Per API angemeldet ({role})',
            'activity.d.loginFailed': 'Fehlgeschlagener Anmeldeversuch {n}/{max}',
            'activity.d.loginFailedApi': 'Fehlgeschlagener API-Anmeldeversuch {n}/{max}',
            'activity.d.loginLocked': 'Nach zu vielen Fehlversuchen gesperrt',
            'activity.d.loginLockedApi': 'Nach zu vielen API-Fehlversuchen gesperrt',
            'activity.d.twoFAInvalid': '2FA-Code fehlt oder ist ungültig',
            'activity.d.twoFAInvalidApi': '2FA-Code fehlt oder ist ungültig (API)',
            'activity.d.localAuthDisabled': 'Passwort-Anmeldung deaktiviert (nur SSO)',
            'activity.d.csrfInvalid': 'Ungültiges Sicherheitstoken bei der Anmeldung',
            'activity.d.rateLimited': 'Zu viele Anmeldeversuche',
            'activity.d.loggedOut': 'Abgemeldet',
            'activity.d.setupTokenInvalid': 'Einrichtung abgelehnt: ungültiges Einrichtungstoken',
            'activity.d.setupDone': 'Ersteinrichtung abgeschlossen',
            'activity.d.twoFAEnabled': '2FA aktiviert',
            'activity.d.twoFADisabled': '2FA deaktiviert',
            'activity.d.revokedAll': 'Alle anderen Sitzungen beendet',
            'activity.d.revokedOne': 'Sitzung {id} beendet ({email})',
            'activity.d.bulk': 'Mehrfachlöschung',
        },
        fr: {
            'common.breadcrumb': 'Fil d’Ariane',
            'common.copy': 'Copier',
            'common.copyLink': 'Copier le lien',
            'common.folder': 'Dossier',
            'common.expiring': 'Expire bientôt',
            'common.autoShare': 'Partage auto',
            'common.qr': 'Code QR',
            'common.edit': 'Modifier',
            'common.delete': 'Supprimer',
            'common.remove': 'Retirer',
            'common.close': 'Fermer',
            'common.refresh': 'Actualiser',
            'nav.upload': 'Envoyer',
            'nav.shares': 'Partages',
            'nav.receive': 'Recevoir',
            'nav.settings': 'Paramètres',
            'nav.logout': 'Déconnexion',
            'login.subtitle': 'Partage de fichiers auto-hébergé',
            'login.password': 'Mot de passe',
            'login.submit': 'Se connecter',
            'login.or': 'ou',
            'login.sso': 'Connexion SSO',
            'upload.title': 'Envoyer des fichiers',
            'upload.dropText': 'Glissez-déposez vos fichiers ici ou cliquez pour parcourir',
            'upload.hint': 'Plusieurs fichiers acceptés',
            'upload.options': 'Options d\'envoi',
            'upload.password': 'Mot de passe (optionnel)',
            'upload.expiry': 'Expiration (heures)',
            'upload.maxDownloads': 'Téléchargements max (0 = illimité)',
            'upload.submit': 'Envoyer',
            'upload.uploading': 'Envoi en cours...',
            'upload.complete': 'Envoi terminé',
            'upload.error': 'Échec de l\'envoi',
            'upload.networkError': 'Serveur injoignable',
            'upload.another': 'Envoyer d\'autres fichiers',
            'shares.title': 'Partages actifs',
            'shares.empty': 'Aucun partage actif',
            'shares.downloads': 'téléchargements',
            'shares.expired': 'expiré',
            'shares.never': 'jamais',
            'shares.unlimited': 'illimité',
            'shares.expiryRequired': 'Saisissez une durée en heures ou activez « illimité ».',
            'shares.deleteConfirm': 'Supprimer ce partage ?',
            'shares.copied': 'Lien copié !',
            'shares.size': 'taille',
            'shares.deleted': 'Partage supprimé',
            'shares.edit': 'Modifier le partage',
            'shares.save': 'Enregistrer',
            'shares.updated': 'Partage mis à jour',
            'shares.changePassword': 'modifier ou supprimer',
            'shares.addPassword': 'ajouter',
            'receive.title': 'Liens de réception',
            'receive.create': 'Nouveau lien',
            'receive.formTitle': 'Créer un lien de réception',
            'receive.name': 'Nom',
            'receive.password': 'Mot de passe (optionnel)',
            'receive.maxUploads': 'Envois max (0 = illimité)',
            'receive.maxFileSize': 'Taille max (Mo, 0 = illimité)',
            'receive.expiry': 'Expiration (heures, 0 = jamais)',
            'receive.extensions': 'Extensions autorisées (ex: .pdf,.doc)',
            'receive.autoShare': 'Partager automatiquement les fichiers reçus',
            'receive.submit': 'Créer',
            'receive.cancel': 'Annuler',
            'receive.empty': 'Aucun lien de réception',
            'receive.uploads': 'envois',
            'receive.deleteConfirm': 'Supprimer ce lien de réception ?',
            'receive.created': 'Lien de réception créé',
            'receive.deleted': 'Lien de réception supprimé',
            'settings.title': 'Paramètres',
            'settings.network': 'Configuration réseau',
            'settings.webhook': 'Webhook',
            'settings.users': 'Gestion des utilisateurs',
            'settings.primary': 'Principal',
            'settings.detected': 'Détecté',
            'settings.notDetected': 'Non détecté',
            'settings.enabled': 'Activé',
            'settings.useDetected': 'Utiliser la valeur détectée',
            'settings.save': 'Enregistrer',
            'settings.saved': 'Paramètres enregistrés',
            'settings.testWebhook': 'Tester le webhook',
            'settings.webhookUrl': 'URL du webhook',
            'settings.webhookSecret': 'Secret du webhook',
            'settings.webhookEnabled': 'Webhook activé',
            'settings.webhookEvents': 'Notifier lors de',
            'settings.webhookOnDownload': 'Fichier téléchargé',
            'settings.webhookOnLimit': 'Limite de téléchargement atteinte',
            'settings.webhookOnExpire': 'Partage expiré',
            'settings.webhookSecretKeep': 'inchangé — laisser vide pour conserver',
            'settings.webhookSecretNone': 'aucun secret défini',
            'settings.webhookSecretClear': 'Supprimer le secret enregistré',
            'settings.webhookNeedsUrl': 'Saisissez une URL de webhook ou désactivez le webhook',
            'settings.webhookSent': 'Webhook envoyé',
            'settings.createUser': 'Créer un utilisateur',
            'settings.email': 'E-mail',
            'settings.name': 'Nom',
            'settings.role': 'Rôle',
            'settings.userPassword': 'Mot de passe',
            'settings.deleteUserConfirm': 'Supprimer cet utilisateur ?',
            'settings.userCreated': 'Utilisateur créé',
            'settings.userDeleted': 'Utilisateur supprimé',
            'settings.fillAll': 'Remplissez tous les champs',
            'toast.copied': 'Lien copié dans le presse-papiers',
            'toast.error': 'Une erreur est survenue',
            'stat.totalShares': 'Partages',
            'stat.totalDownloads': 'Téléchargements',
            'stat.totalSize': 'Taille totale',
            'stat.protected': 'Protégés',
            'nav.hostshare': 'Partager depuis l\'hôte',
            'hostshare.title': 'Partager depuis l\'hôte',
            'hostshare.hint': 'Partagez des fichiers et dossiers déjà présents sur le serveur, sans les renvoyer.',
            'hostshare.home': 'Accueil',
            'hostshare.empty': 'Ce dossier est vide',
            'hostshare.error': 'Impossible d\'accéder à cet emplacement',
            'hostshare.folder': 'Dossier',
            'hostshare.share': 'Partager',
            'hostshare.dialogTitle': 'Partager cet élément',
            'hostshare.symlink': 'Lier au lieu de copier (sans espace disque supplémentaire)',
            'hostshare.success': 'Partage créé',
            'hostshare.done': 'Terminé',
            'upload.expiry.1h': '1 heure',
            'upload.expiry.6h': '6 heures',
            'upload.expiry.12h': '12 heures',
            'upload.expiry.1d': '1 jour',
            'upload.expiry.3d': '3 jours',
            'upload.expiry.7d': '7 jours',
            'upload.expiry.14d': '14 jours',
            'upload.expiry.30d': '30 jours',
            'upload.expiry.never': 'Jamais (illimité)',
            'upload.expiry.hours': 'heures',
            'settings.customHint': 'Saisissez votre propre domaine fixe (par ex. reverse proxy ou domaine d\'un tunnel nommé Cloudflare).',
            'settings.cfRotating': 'Génération d\'une nouvelle URL de partage Cloudflare…',
            'settings.cfRotated': 'Nouvelle URL Cloudflare prête :',
            'settings.cfRotateTimeout': 'Aucune nouvelle URL Cloudflare n\'est apparue. Le conteneur du tunnel est-il en cours d\'exécution ?',
            'settings.quota': 'Stockage / Quota',
            'settings.quotaGB': 'Quota (Go, 0 = illimité)',
            'settings.unlimited': 'illimité',
            'settings.quotaUpdated': 'Quota mis à jour',
            'settings.userPasswordHint': 'Optionnel : laisser vide pour les comptes SSO uniquement (8 caractères min. si défini)',
            'settings.passwordTooShort': 'Le mot de passe doit contenir au moins 8 caractères',
            'settings.maxFileSize': 'Taille max. des fichiers',
            'settings.fileRestrictions': 'Restrictions de fichiers',
            'settings.fileRestrictionsHint': 'Définissez les types de fichiers pouvant être envoyés. Les extensions bloquées sont refusées ; les extensions autorisées forment une liste blanche (vide = tout est autorisé sauf les extensions bloquées).',
            'settings.blockedExtensions': 'Extensions bloquées',
            'settings.allowedExtensions': 'Extensions autorisées (liste blanche)',
            'settings.allowedExtensionsHint': 'Vide = tout est autorisé (sauf les extensions bloquées)',
            'shares.sendEmail': 'Envoyer par e-mail',
            'shares.taildrop': 'Envoyer vers un appareil (Taildrop)',
            'taildrop.title': 'Envoyer vers un appareil',
            'taildrop.device': 'Appareil cible',
            'taildrop.send': 'Envoyer',
            'taildrop.sending': 'Envoi en cours…',
            'taildrop.sent': 'Fichier envoyé via Taildrop',
            'taildrop.noDevices': 'Aucun appareil Tailscale disponible',
            'shares.selectAll': 'Tout sélectionner',
            'shares.bulkDelete': 'Supprimer la sélection',
            'shares.selectedCount': 'Sélectionnés : {n}',
            'email.recipientEmail': 'E-mail du destinataire',
            'email.recipientName': 'Nom du destinataire',
            'email.message': 'Message',
            'email.optional': 'Optionnel',
            'email.send': 'Envoyer',
            'email.sent': 'E-mail envoyé avec succès',
            'email.enterRecipient': 'Saisissez l\'e-mail du destinataire',
            'email.sendFailed': 'Impossible d\'envoyer l\'e-mail :',
            'upload.folder': 'Envoyer un dossier',
            'settings.apiKeys': 'Clés API',
            'settings.apiKeysHint': 'Les clés API permettent un accès programmatique à CasaDrop. Utilisez l\'en-tête X-API-Key.',
            'settings.noApiKeys': 'Aucune clé API',
            'settings.createApiKey': 'Créer une clé',
            'settings.apiKeyCreated': 'Clé API créée : copiez-la maintenant !',
            'settings.apiKeyOnlyOnce': 'Cette clé ne sera affichée qu\'une seule fois. Conservez-la en lieu sûr.',
            'settings.deleteApiKeyConfirm': 'Supprimer cette clé API ?',
            'settings.apiKeyDeleted': 'Clé API supprimée',
            'settings.smtp': 'E-mail / SMTP',
            'settings.smtpEnabled': 'Activer l\'envoi d\'e-mails',
            'settings.testSmtp': 'Tester la connexion',
            'settings.smtpTestOk': 'Connexion SMTP réussie',
            'settings.twofa': 'Authentification à deux facteurs',
            'settings.twofaHint': 'Ajoutez un mot de passe à usage unique basé sur le temps (TOTP) à votre connexion administrateur. Utilisez une application d\'authentification comme Aegis, Google Authenticator ou 1Password.',
            'settings.twofaStatus': 'Statut',
            'settings.twofaEnabled': 'Activée',
            'settings.twofaDisabled': 'Désactivée',
            'settings.twofaEnable': 'Activer la 2FA',
            'settings.twofaDisable': 'Désactiver la 2FA',
            'settings.twofaVerifyEnable': 'Vérifier et activer',
            'settings.twofaCancel': 'Annuler',
            'settings.twofaScan': 'Scannez ce code QR avec votre application d\'authentification :',
            'settings.twofaManual': 'Ou saisissez ce secret manuellement :',
            'settings.twofaCode': 'Saisissez le code à 6 chiffres',
            'settings.twofaEnabledMsg': 'Authentification à deux facteurs activée',
            'settings.twofaDisabledMsg': 'Authentification à deux facteurs désactivée',
            'settings.twofaCodeRequired': 'Veuillez saisir le code à 6 chiffres',
            'settings.twofaSetupFailed': 'Impossible de lancer la configuration de la 2FA',
            'settings.sessions': 'Sessions actives',
            'sessions.hint': 'Tous les appareils connectés à votre compte. Fermez toute session que vous ne reconnaissez pas.',
            'sessions.hintAdmin': 'Toutes les sessions actives sur ce serveur. Fermez toute session que vous ne reconnaissez pas.',
            'sessions.current': 'Cet appareil',
            'sessions.since': 'Connecté le',
            'sessions.expires': 'Expire le',
            'sessions.revoke': 'Fermer la session',
            'sessions.revokeOthers': 'Déconnecter tous les autres appareils',
            'sessions.revokedOthers': '{n} autre(s) appareil(s) déconnecté(s)',
            'sessions.revoked': 'Session fermée',
            'sessions.empty': 'Aucune autre session active.',
            'sessions.confirmOthers': 'Se déconnecter de tous les autres appareils ?',
            'settings.activity': 'Journal d\'activité',
            'settings.activityHint': 'Qui a téléchargé, envoyé, s\'est connecté, avec l\'adresse et l\'heure. Les entrées plus anciennes que la durée de conservation sont supprimées automatiquement.',
            'activity.when': 'Quand',
            'activity.kind': 'Événement',
            'activity.who': 'Qui',
            'activity.share': 'Partage',
            'activity.from': 'Origine',
            'activity.detail': 'Détail',
            'activity.empty': 'Rien n\'a encore été enregistré.',
            'activity.export': 'Exporter en CSV',
            'activity.allKinds': 'Tous les événements',
            'activity.loadMore': 'Charger plus',
            'activity.anonymous': 'anonyme',
            'activity.system': 'système',
            'activity.forShare': 'Activité',
            'activity.loadFailed': 'Impossible de charger le journal d\'activité',
            'activity.kind.share.created': 'Partage créé',
            'activity.kind.share.updated': 'Partage mis à jour',
            'activity.kind.share.deleted': 'Partage supprimé',
            'activity.kind.share.expired': 'Partage expiré',
            'activity.kind.share.downloaded': 'Téléchargé',
            'activity.kind.share.streamed': 'Lu en streaming',
            'activity.kind.receive.uploaded': 'Fichier reçu',
            'activity.kind.auth.login': 'Connexion',
            'activity.kind.auth.login_failed': 'Échec de connexion',
            'activity.kind.auth.locked': 'Compte verrouillé',
            'activity.kind.auth.logout': 'Déconnexion',
            'activity.kind.auth.setup': 'Configuration / modification 2FA',
            'activity.kind.session.revoked': 'Session révoquée',
            'activity.kind.security': 'Sécurité',
            'theme.light': 'Mode clair',
            'theme.dark': 'Mode sombre',
            'load.networkFailed': 'Impossible de charger les paramètres réseau',
            'load.restrictionsFailed': 'Impossible de charger les restrictions de fichiers',
            'load.webhookFailed': 'Impossible de charger les paramètres du webhook',
            'load.usersUnavailable': 'La gestion des utilisateurs n’est pas disponible',
            'load.twoFAFailed': 'Impossible de charger les paramètres 2FA',
            'load.apiKeysFailed': 'Impossible de charger les clés API',
            'load.smtpFailed': 'Impossible de charger les paramètres SMTP',
            'role.admin': 'Administrateur',
            'role.user': 'Utilisateur',
            'role.viewer': 'Lecteur',
            'settings.apiKeyNamePlaceholder': 'Ma clé API',
            'common.copied': 'Copié !',
            'smtp.host': 'Serveur SMTP',
            'smtp.port': 'Port',
            'smtp.encryption': 'Chiffrement',
            'smtp.encryptionNone': 'Aucun',
            'smtp.username': 'Nom d’utilisateur',
            'smtp.password': 'Mot de passe',
            'smtp.fromEmail': 'E-mail de l’expéditeur',
            'smtp.fromName': 'Nom de l’expéditeur',
            'auth.signInAgain': 'Veuillez vous reconnecter',
            'shares.expiresWhen': 'Expire {when}',
            'activity.d.loginOk': 'Connexion réussie ({role})',
            'activity.d.loginOkApi': 'Connexion via l’API réussie ({role})',
            'activity.d.loginFailed': 'Échec de connexion {n}/{max}',
            'activity.d.loginFailedApi': 'Échec de connexion via l’API {n}/{max}',
            'activity.d.loginLocked': 'Verrouillé après trop d’échecs',
            'activity.d.loginLockedApi': 'Verrouillé après trop d’échecs via l’API',
            'activity.d.twoFAInvalid': 'Code 2FA manquant ou invalide',
            'activity.d.twoFAInvalidApi': 'Code 2FA manquant ou invalide (API)',
            'activity.d.localAuthDisabled': 'Connexion par mot de passe désactivée (SSO uniquement)',
            'activity.d.csrfInvalid': 'Jeton de sécurité invalide à la connexion',
            'activity.d.rateLimited': 'Trop de tentatives de connexion',
            'activity.d.loggedOut': 'Déconnexion',
            'activity.d.setupTokenInvalid': 'Configuration refusée : jeton de configuration invalide',
            'activity.d.setupDone': 'Configuration initiale terminée',
            'activity.d.twoFAEnabled': '2FA activée',
            'activity.d.twoFADisabled': '2FA désactivée',
            'activity.d.revokedAll': 'Toutes les autres sessions fermées',
            'activity.d.revokedOne': 'Session {id} fermée ({email})',
            'activity.d.bulk': 'suppression groupée',
        },
        es: {
            'common.breadcrumb': 'Ruta de navegación',
            'common.copy': 'Copiar',
            'common.copyLink': 'Copiar enlace',
            'common.folder': 'Carpeta',
            'common.expiring': 'Caduca pronto',
            'common.autoShare': 'Compartir auto.',
            'common.qr': 'Código QR',
            'common.edit': 'Editar',
            'common.delete': 'Eliminar',
            'common.remove': 'Quitar',
            'common.close': 'Cerrar',
            'common.refresh': 'Actualizar',
            'nav.upload': 'Subir',
            'nav.shares': 'Compartidos',
            'nav.receive': 'Recibir',
            'nav.settings': 'Ajustes',
            'nav.logout': 'Cerrar sesión',
            'login.subtitle': 'Compartir archivos autoalojado',
            'login.password': 'Contraseña',
            'login.submit': 'Iniciar sesión',
            'login.or': 'o',
            'login.sso': 'Iniciar sesión con SSO',
            'upload.title': 'Subir archivos',
            'upload.dropText': 'Arrastra archivos aquí o haz clic para explorar',
            'upload.hint': 'Se admiten varios archivos',
            'upload.options': 'Opciones de subida',
            'upload.password': 'Contraseña (opcional)',
            'upload.expiry': 'Expira en (horas)',
            'upload.maxDownloads': 'Descargas máx. (0 = ilimitado)',
            'upload.submit': 'Subir',
            'upload.uploading': 'Subiendo...',
            'upload.complete': 'Subida completada',
            'upload.error': 'Error al subir',
            'upload.networkError': 'Servidor inaccesible',
            'upload.another': 'Subir más',
            'shares.title': 'Compartidos activos',
            'shares.empty': 'Aún no hay compartidos',
            'shares.downloads': 'descargas',
            'shares.expired': 'expirado',
            'shares.never': 'nunca',
            'shares.unlimited': 'ilimitado',
            'shares.expiryRequired': 'Introduce una caducidad en horas o activa «ilimitado».',
            'shares.deleteConfirm': '¿Eliminar este compartido?',
            'shares.copied': '¡Enlace copiado!',
            'shares.size': 'tamaño',
            'shares.deleted': 'Compartido eliminado',
            'shares.edit': 'Editar compartido',
            'shares.save': 'Guardar',
            'shares.updated': 'Compartido actualizado',
            'shares.changePassword': 'cambiar o eliminar',
            'shares.addPassword': 'añadir',
            'receive.title': 'Enlaces de recepción',
            'receive.create': 'Nuevo enlace',
            'receive.formTitle': 'Crear enlace de recepción',
            'receive.name': 'Nombre',
            'receive.password': 'Contraseña (opcional)',
            'receive.maxUploads': 'Subidas máx. (0 = ilimitado)',
            'receive.maxFileSize': 'Tamaño máx. (MB, 0 = ilimitado)',
            'receive.expiry': 'Expira en (horas, 0 = nunca)',
            'receive.extensions': 'Extensiones permitidas (ej: .pdf,.doc)',
            'receive.autoShare': 'Compartir automáticamente los archivos recibidos',
            'receive.submit': 'Crear',
            'receive.cancel': 'Cancelar',
            'receive.empty': 'Aún no hay enlaces de recepción',
            'receive.uploads': 'subidas',
            'receive.deleteConfirm': '¿Eliminar este enlace de recepción?',
            'receive.created': 'Enlace de recepción creado',
            'receive.deleted': 'Enlace de recepción eliminado',
            'settings.title': 'Ajustes',
            'settings.network': 'Configuración de red',
            'settings.webhook': 'Webhook',
            'settings.users': 'Gestión de usuarios',
            'settings.primary': 'Principal',
            'settings.detected': 'Detectado',
            'settings.notDetected': 'No detectado',
            'settings.enabled': 'Habilitado',
            'settings.useDetected': 'Usar valor detectado',
            'settings.save': 'Guardar',
            'settings.saved': 'Ajustes guardados',
            'settings.testWebhook': 'Probar webhook',
            'settings.webhookUrl': 'URL del webhook',
            'settings.webhookSecret': 'Secreto del webhook',
            'settings.webhookEnabled': 'Webhook activado',
            'settings.webhookEvents': 'Notificar cuando',
            'settings.webhookOnDownload': 'Archivo descargado',
            'settings.webhookOnLimit': 'Límite de descargas alcanzado',
            'settings.webhookOnExpire': 'Recurso caducado',
            'settings.webhookSecretKeep': 'sin cambios — dejar vacío para conservar',
            'settings.webhookSecretNone': 'ningún secreto definido',
            'settings.webhookSecretClear': 'Eliminar el secreto guardado',
            'settings.webhookNeedsUrl': 'Introduce una URL de webhook o desactiva el webhook',
            'settings.webhookSent': 'Webhook enviado',
            'settings.createUser': 'Crear usuario',
            'settings.email': 'Correo electrónico',
            'settings.name': 'Nombre',
            'settings.role': 'Rol',
            'settings.userPassword': 'Contraseña',
            'settings.deleteUserConfirm': '¿Eliminar este usuario?',
            'settings.userCreated': 'Usuario creado',
            'settings.userDeleted': 'Usuario eliminado',
            'settings.fillAll': 'Completa todos los campos',
            'toast.copied': 'Enlace copiado al portapapeles',
            'toast.error': 'Algo salió mal',
            'stat.totalShares': 'Compartidos',
            'stat.totalDownloads': 'Descargas',
            'stat.totalSize': 'Tamaño total',
            'stat.protected': 'Protegidos',
            'nav.hostshare': 'Compartir desde el host',
            'hostshare.title': 'Compartir desde el host',
            'hostshare.hint': 'Comparte archivos y carpetas que ya están en el servidor, sin volver a subirlos.',
            'hostshare.home': 'Inicio',
            'hostshare.empty': 'Esta carpeta está vacía',
            'hostshare.error': 'No se pudo acceder a esta ubicación',
            'hostshare.folder': 'Carpeta',
            'hostshare.share': 'Compartir',
            'hostshare.dialogTitle': 'Compartir este elemento',
            'hostshare.symlink': 'Enlazar en lugar de copiar (sin espacio adicional en disco)',
            'hostshare.success': 'Compartido creado',
            'hostshare.done': 'Listo',
            'upload.expiry.1h': '1 hora',
            'upload.expiry.6h': '6 horas',
            'upload.expiry.12h': '12 horas',
            'upload.expiry.1d': '1 día',
            'upload.expiry.3d': '3 días',
            'upload.expiry.7d': '7 días',
            'upload.expiry.14d': '14 días',
            'upload.expiry.30d': '30 días',
            'upload.expiry.never': 'Nunca (ilimitado)',
            'upload.expiry.hours': 'horas',
            'settings.customHint': 'Introduce tu propio dominio fijo (p. ej. un proxy inverso o el dominio de un túnel con nombre de Cloudflare).',
            'settings.cfRotating': 'Generando una nueva URL de Cloudflare para compartir…',
            'settings.cfRotated': 'Nueva URL de Cloudflare lista:',
            'settings.cfRotateTimeout': 'No ha aparecido ninguna URL nueva de Cloudflare. ¿Está en ejecución el contenedor del túnel?',
            'settings.quota': 'Almacenamiento / Cuota',
            'settings.quotaGB': 'Cuota (GB, 0 = ilimitada)',
            'settings.unlimited': 'ilimitado',
            'settings.quotaUpdated': 'Cuota actualizada',
            'settings.userPasswordHint': 'Opcional: déjalo vacío para cuentas solo SSO (mín. 8 caracteres si se define)',
            'settings.passwordTooShort': 'La contraseña debe tener al menos 8 caracteres',
            'settings.maxFileSize': 'Tamaño máx. de archivo',
            'settings.fileRestrictions': 'Restricciones de archivos',
            'settings.fileRestrictionsHint': 'Controla qué tipos de archivo se pueden subir. Las extensiones bloqueadas se rechazan; las extensiones permitidas forman una lista blanca (vacía = se permite todo salvo lo bloqueado).',
            'settings.blockedExtensions': 'Extensiones bloqueadas',
            'settings.allowedExtensions': 'Extensiones permitidas (lista blanca)',
            'settings.allowedExtensionsHint': 'Vacía = se permite todo (salvo lo bloqueado)',
            'shares.sendEmail': 'Enviar por correo electrónico',
            'shares.taildrop': 'Enviar a dispositivo (Taildrop)',
            'taildrop.title': 'Enviar a dispositivo',
            'taildrop.device': 'Dispositivo de destino',
            'taildrop.send': 'Enviar',
            'taildrop.sending': 'Enviando…',
            'taildrop.sent': 'Archivo enviado con Taildrop',
            'taildrop.noDevices': 'No hay dispositivos de Tailscale disponibles',
            'shares.selectAll': 'Seleccionar todo',
            'shares.bulkDelete': 'Eliminar selección',
            'shares.selectedCount': 'Seleccionados: {n}',
            'email.recipientEmail': 'Correo electrónico del destinatario',
            'email.recipientName': 'Nombre del destinatario',
            'email.message': 'Mensaje',
            'email.optional': 'Opcional',
            'email.send': 'Enviar',
            'email.sent': 'Correo electrónico enviado correctamente',
            'email.enterRecipient': 'Introduce el correo electrónico del destinatario',
            'email.sendFailed': 'No se pudo enviar el correo electrónico:',
            'upload.folder': 'Subir carpeta',
            'settings.apiKeys': 'Claves API',
            'settings.apiKeysHint': 'Las claves API permiten el acceso programático a CasaDrop. Usa la cabecera X-API-Key.',
            'settings.noApiKeys': 'Aún no hay claves API',
            'settings.createApiKey': 'Crear clave',
            'settings.apiKeyCreated': 'Clave API creada: ¡cópiala ahora!',
            'settings.apiKeyOnlyOnce': 'Esta clave solo se mostrará una vez. Guárdala en un lugar seguro.',
            'settings.deleteApiKeyConfirm': '¿Eliminar esta clave API?',
            'settings.apiKeyDeleted': 'Clave API eliminada',
            'settings.smtp': 'Correo electrónico / SMTP',
            'settings.smtpEnabled': 'Habilitar el envío de correos',
            'settings.testSmtp': 'Probar conexión',
            'settings.smtpTestOk': 'Conexión SMTP correcta',
            'settings.twofa': 'Autenticación en dos pasos',
            'settings.twofaHint': 'Añade una contraseña de un solo uso basada en el tiempo (TOTP) a tu inicio de sesión de administrador. Usa una app de autenticación como Aegis, Google Authenticator o 1Password.',
            'settings.twofaStatus': 'Estado',
            'settings.twofaEnabled': 'Habilitada',
            'settings.twofaDisabled': 'Deshabilitada',
            'settings.twofaEnable': 'Habilitar 2FA',
            'settings.twofaDisable': 'Deshabilitar 2FA',
            'settings.twofaVerifyEnable': 'Verificar y habilitar',
            'settings.twofaCancel': 'Cancelar',
            'settings.twofaScan': 'Escanea este código QR con tu app de autenticación:',
            'settings.twofaManual': 'O introduce este secreto manualmente:',
            'settings.twofaCode': 'Introduce el código de 6 dígitos',
            'settings.twofaEnabledMsg': 'Autenticación en dos pasos habilitada',
            'settings.twofaDisabledMsg': 'Autenticación en dos pasos deshabilitada',
            'settings.twofaCodeRequired': 'Introduce el código de 6 dígitos',
            'settings.twofaSetupFailed': 'No se pudo iniciar la configuración de 2FA',
            'settings.sessions': 'Sesiones activas',
            'sessions.hint': 'Todos los dispositivos con sesión iniciada en tu cuenta. Cierra cualquier sesión que no reconozcas.',
            'sessions.hintAdmin': 'Todas las sesiones activas en este servidor. Cierra cualquiera que no reconozcas.',
            'sessions.current': 'Este dispositivo',
            'sessions.since': 'Sesión iniciada',
            'sessions.expires': 'Caduca',
            'sessions.revoke': 'Cerrar sesión',
            'sessions.revokeOthers': 'Cerrar sesión en todos los demás dispositivos',
            'sessions.revokedOthers': 'Sesión cerrada en {n} dispositivo(s) más',
            'sessions.revoked': 'Sesión cerrada',
            'sessions.empty': 'No hay otras sesiones activas.',
            'sessions.confirmOthers': '¿Cerrar sesión en todos los demás dispositivos?',
            'settings.activity': 'Registro de actividad',
            'settings.activityHint': 'Quién ha descargado, subido o iniciado sesión, con dirección y hora. Las entradas más antiguas que el periodo de conservación se eliminan automáticamente.',
            'activity.when': 'Cuándo',
            'activity.kind': 'Evento',
            'activity.who': 'Quién',
            'activity.share': 'Compartido',
            'activity.from': 'Origen',
            'activity.detail': 'Detalle',
            'activity.empty': 'Aún no hay nada registrado.',
            'activity.export': 'Exportar CSV',
            'activity.allKinds': 'Todos los eventos',
            'activity.loadMore': 'Cargar más',
            'activity.anonymous': 'anónimo',
            'activity.system': 'sistema',
            'activity.forShare': 'Actividad',
            'activity.loadFailed': 'No se pudo cargar el registro de actividad',
            'activity.kind.share.created': 'Compartido creado',
            'activity.kind.share.updated': 'Compartido actualizado',
            'activity.kind.share.deleted': 'Compartido eliminado',
            'activity.kind.share.expired': 'Compartido caducado',
            'activity.kind.share.downloaded': 'Descargado',
            'activity.kind.share.streamed': 'Reproducido en streaming',
            'activity.kind.receive.uploaded': 'Archivo recibido',
            'activity.kind.auth.login': 'Inicio de sesión',
            'activity.kind.auth.login_failed': 'Inicio de sesión fallido',
            'activity.kind.auth.locked': 'Cuenta bloqueada',
            'activity.kind.auth.logout': 'Cierre de sesión',
            'activity.kind.auth.setup': 'Configuración / cambio de 2FA',
            'activity.kind.session.revoked': 'Sesión revocada',
            'activity.kind.security': 'Seguridad',
            'theme.light': 'Modo claro',
            'theme.dark': 'Modo oscuro',
            'load.networkFailed': 'No se pudo cargar la configuración de red',
            'load.restrictionsFailed': 'No se pudieron cargar las restricciones de archivos',
            'load.webhookFailed': 'No se pudo cargar la configuración del webhook',
            'load.usersUnavailable': 'La gestión de usuarios no está disponible',
            'load.twoFAFailed': 'No se pudo cargar la configuración de 2FA',
            'load.apiKeysFailed': 'No se pudieron cargar las claves API',
            'load.smtpFailed': 'No se pudo cargar la configuración SMTP',
            'role.admin': 'Administrador',
            'role.user': 'Usuario',
            'role.viewer': 'Lector',
            'settings.apiKeyNamePlaceholder': 'Mi clave API',
            'common.copied': '¡Copiado!',
            'smtp.host': 'Servidor SMTP',
            'smtp.port': 'Puerto',
            'smtp.encryption': 'Cifrado',
            'smtp.encryptionNone': 'Ninguno',
            'smtp.username': 'Nombre de usuario',
            'smtp.password': 'Contraseña',
            'smtp.fromEmail': 'Correo del remitente',
            'smtp.fromName': 'Nombre del remitente',
            'auth.signInAgain': 'Vuelve a iniciar sesión',
            'shares.expiresWhen': 'Caduca {when}',
            'activity.d.loginOk': 'Inicio de sesión correcto ({role})',
            'activity.d.loginOkApi': 'Inicio de sesión por API correcto ({role})',
            'activity.d.loginFailed': 'Intento de inicio de sesión fallido {n}/{max}',
            'activity.d.loginFailedApi': 'Intento de inicio de sesión por API fallido {n}/{max}',
            'activity.d.loginLocked': 'Bloqueado tras demasiados intentos fallidos',
            'activity.d.loginLockedApi': 'Bloqueado tras demasiados intentos fallidos por API',
            'activity.d.twoFAInvalid': 'Código 2FA ausente o no válido',
            'activity.d.twoFAInvalidApi': 'Código 2FA ausente o no válido (API)',
            'activity.d.localAuthDisabled': 'Inicio de sesión con contraseña desactivado (solo SSO)',
            'activity.d.csrfInvalid': 'Token de seguridad no válido al iniciar sesión',
            'activity.d.rateLimited': 'Demasiados intentos de inicio de sesión',
            'activity.d.loggedOut': 'Sesión cerrada',
            'activity.d.setupTokenInvalid': 'Configuración rechazada: token de configuración no válido',
            'activity.d.setupDone': 'Configuración inicial completada',
            'activity.d.twoFAEnabled': '2FA activada',
            'activity.d.twoFADisabled': '2FA desactivada',
            'activity.d.revokedAll': 'Todas las demás sesiones cerradas',
            'activity.d.revokedOne': 'Sesión {id} cerrada ({email})',
            'activity.d.bulk': 'eliminación múltiple',
        },
        it: {
            'common.breadcrumb': 'Percorso di navigazione',
            'common.copy': 'Copia',
            'common.copyLink': 'Copia link',
            'common.folder': 'Cartella',
            'common.expiring': 'In scadenza',
            'common.autoShare': 'Condivisione auto',
            'common.qr': 'Codice QR',
            'common.edit': 'Modifica',
            'common.delete': 'Elimina',
            'common.remove': 'Rimuovi',
            'common.close': 'Chiudi',
            'common.refresh': 'Aggiorna',
            'nav.upload': 'Carica',
            'nav.shares': 'Condivisioni',
            'nav.receive': 'Ricevi',
            'nav.settings': 'Impostazioni',
            'nav.logout': 'Esci',
            'login.subtitle': 'Condivisione file self-hosted',
            'login.password': 'Password',
            'login.submit': 'Accedi',
            'login.or': 'o',
            'login.sso': 'Accedi con SSO',
            'upload.title': 'Carica file',
            'upload.dropText': 'Trascina i file qui o clicca per sfogliare',
            'upload.hint': 'File multipli supportati',
            'upload.options': 'Opzioni di caricamento',
            'upload.password': 'Password (opzionale)',
            'upload.expiry': 'Scadenza (ore)',
            'upload.maxDownloads': 'Download max (0 = illimitati)',
            'upload.submit': 'Carica',
            'upload.uploading': 'Caricamento in corso...',
            'upload.complete': 'Caricamento completato',
            'upload.error': 'Caricamento fallito',
            'upload.networkError': 'Server irraggiungibile',
            'upload.another': 'Carica altri file',
            'shares.title': 'Condivisioni attive',
            'shares.empty': 'Nessuna condivisione attiva',
            'shares.downloads': 'download',
            'shares.expired': 'scaduto',
            'shares.never': 'mai',
            'shares.unlimited': 'illimitato',
            'shares.expiryRequired': 'Inserisci una scadenza in ore oppure attiva "illimitato".',
            'shares.deleteConfirm': 'Eliminare questa condivisione?',
            'shares.copied': 'Link copiato!',
            'shares.size': 'dimensione',
            'shares.deleted': 'Condivisione eliminata',
            'shares.edit': 'Modifica condivisione',
            'shares.save': 'Salva',
            'shares.updated': 'Condivisione aggiornata',
            'shares.changePassword': 'modifica o rimuovi',
            'shares.addPassword': 'aggiungi',
            'receive.title': 'Link di ricezione',
            'receive.create': 'Nuovo link',
            'receive.formTitle': 'Crea link di ricezione',
            'receive.name': 'Nome',
            'receive.password': 'Password (opzionale)',
            'receive.maxUploads': 'Caricamenti max (0 = illimitati)',
            'receive.maxFileSize': 'Dimensione max (MB, 0 = illimitata)',
            'receive.expiry': 'Scadenza (ore, 0 = mai)',
            'receive.extensions': 'Estensioni consentite (es: .pdf,.doc)',
            'receive.autoShare': 'Condividi automaticamente i file ricevuti',
            'receive.submit': 'Crea',
            'receive.cancel': 'Annulla',
            'receive.empty': 'Nessun link di ricezione',
            'receive.uploads': 'caricamenti',
            'receive.deleteConfirm': 'Eliminare questo link di ricezione?',
            'receive.created': 'Link di ricezione creato',
            'receive.deleted': 'Link di ricezione eliminato',
            'settings.title': 'Impostazioni',
            'settings.network': 'Configurazione di rete',
            'settings.webhook': 'Webhook',
            'settings.users': 'Gestione utenti',
            'settings.primary': 'Principale',
            'settings.detected': 'Rilevato',
            'settings.notDetected': 'Non rilevato',
            'settings.enabled': 'Abilitato',
            'settings.useDetected': 'Usa valore rilevato',
            'settings.save': 'Salva',
            'settings.saved': 'Impostazioni salvate',
            'settings.testWebhook': 'Testa webhook',
            'settings.webhookUrl': 'URL webhook',
            'settings.webhookSecret': 'Segreto webhook',
            'settings.webhookEnabled': 'Webhook attivo',
            'settings.webhookEvents': 'Notifica quando',
            'settings.webhookOnDownload': 'File scaricato',
            'settings.webhookOnLimit': 'Limite di download raggiunto',
            'settings.webhookOnExpire': 'Condivisione scaduta',
            'settings.webhookSecretKeep': 'invariato — lascia vuoto per mantenere',
            'settings.webhookSecretNone': 'nessun segreto impostato',
            'settings.webhookSecretClear': 'Rimuovi il segreto salvato',
            'settings.webhookNeedsUrl': 'Inserisci un URL webhook o disattiva il webhook',
            'settings.webhookSent': 'Webhook inviato',
            'settings.createUser': 'Crea utente',
            'settings.email': 'E-mail',
            'settings.name': 'Nome',
            'settings.role': 'Ruolo',
            'settings.userPassword': 'Password',
            'settings.deleteUserConfirm': 'Eliminare questo utente?',
            'settings.userCreated': 'Utente creato',
            'settings.userDeleted': 'Utente eliminato',
            'settings.fillAll': 'Compila tutti i campi',
            'toast.copied': 'Link copiato negli appunti',
            'toast.error': 'Qualcosa è andato storto',
            'stat.totalShares': 'Condivisioni',
            'stat.totalDownloads': 'Download',
            'stat.totalSize': 'Dimensione totale',
            'stat.protected': 'Protetti',
            'nav.hostshare': 'Condividi dall\'host',
            'hostshare.title': 'Condividi dall\'host',
            'hostshare.hint': 'Condividi file e cartelle già presenti sul server, senza ricaricarli.',
            'hostshare.home': 'Home',
            'hostshare.empty': 'Questa cartella è vuota',
            'hostshare.error': 'Impossibile accedere a questo percorso',
            'hostshare.folder': 'Cartella',
            'hostshare.share': 'Condividi',
            'hostshare.dialogTitle': 'Condividi questo elemento',
            'hostshare.symlink': 'Collega invece di copiare (nessuno spazio su disco aggiuntivo)',
            'hostshare.success': 'Condivisione creata',
            'hostshare.done': 'Fatto',
            'upload.expiry.1h': '1 ora',
            'upload.expiry.6h': '6 ore',
            'upload.expiry.12h': '12 ore',
            'upload.expiry.1d': '1 giorno',
            'upload.expiry.3d': '3 giorni',
            'upload.expiry.7d': '7 giorni',
            'upload.expiry.14d': '14 giorni',
            'upload.expiry.30d': '30 giorni',
            'upload.expiry.never': 'Mai (illimitato)',
            'upload.expiry.hours': 'ore',
            'settings.customHint': 'Inserisci il tuo dominio fisso (ad es. reverse proxy o dominio di un tunnel con nome Cloudflare).',
            'settings.cfRotating': 'Generazione di un nuovo URL di condivisione Cloudflare…',
            'settings.cfRotated': 'Nuovo URL Cloudflare pronto:',
            'settings.cfRotateTimeout': 'Nessun nuovo URL Cloudflare è comparso. Il container del tunnel è in esecuzione?',
            'settings.quota': 'Spazio / Quota',
            'settings.quotaGB': 'Quota (GB, 0 = illimitata)',
            'settings.unlimited': 'illimitato',
            'settings.quotaUpdated': 'Quota aggiornata',
            'settings.userPasswordHint': 'Opzionale: lascia vuoto per gli account solo SSO (min. 8 caratteri se impostata)',
            'settings.passwordTooShort': 'La password deve contenere almeno 8 caratteri',
            'settings.maxFileSize': 'Dimensione max file',
            'settings.fileRestrictions': 'Restrizioni sui file',
            'settings.fileRestrictionsHint': 'Controlla quali tipi di file possono essere caricati. Le estensioni bloccate vengono rifiutate, quelle consentite formano una whitelist (vuota = tutto consentito tranne le bloccate).',
            'settings.blockedExtensions': 'Estensioni bloccate',
            'settings.allowedExtensions': 'Estensioni consentite (whitelist)',
            'settings.allowedExtensionsHint': 'Vuota = tutto consentito (tranne le bloccate)',
            'shares.sendEmail': 'Invia via e-mail',
            'shares.taildrop': 'Invia a dispositivo (Taildrop)',
            'taildrop.title': 'Invia a dispositivo',
            'taildrop.device': 'Dispositivo di destinazione',
            'taildrop.send': 'Invia',
            'taildrop.sending': 'Invio in corso…',
            'taildrop.sent': 'File inviato tramite Taildrop',
            'taildrop.noDevices': 'Nessun dispositivo Tailscale disponibile',
            'shares.selectAll': 'Seleziona tutto',
            'shares.bulkDelete': 'Elimina selezionati',
            'shares.selectedCount': 'Selezionati: {n}',
            'email.recipientEmail': 'E-mail del destinatario',
            'email.recipientName': 'Nome del destinatario',
            'email.message': 'Messaggio',
            'email.optional': 'Opzionale',
            'email.send': 'Invia',
            'email.sent': 'E-mail inviata correttamente',
            'email.enterRecipient': 'Inserisci l\'e-mail del destinatario',
            'email.sendFailed': 'Impossibile inviare l\'e-mail:',
            'upload.folder': 'Carica cartella',
            'settings.apiKeys': 'Chiavi API',
            'settings.apiKeysHint': 'Le chiavi API consentono l\'accesso programmatico a CasaDrop. Usa l\'header X-API-Key.',
            'settings.noApiKeys': 'Nessuna chiave API',
            'settings.createApiKey': 'Crea chiave',
            'settings.apiKeyCreated': 'Chiave API creata: copiala subito!',
            'settings.apiKeyOnlyOnce': 'Questa chiave verrà mostrata una sola volta. Conservala in un luogo sicuro.',
            'settings.deleteApiKeyConfirm': 'Eliminare questa chiave API?',
            'settings.apiKeyDeleted': 'Chiave API eliminata',
            'settings.smtp': 'E-mail / SMTP',
            'settings.smtpEnabled': 'Abilita l\'invio di e-mail',
            'settings.testSmtp': 'Testa connessione',
            'settings.smtpTestOk': 'Connessione SMTP riuscita',
            'settings.twofa': 'Autenticazione a due fattori',
            'settings.twofaHint': 'Aggiungi una password monouso basata sul tempo (TOTP) al tuo accesso amministratore. Usa un\'app di autenticazione come Aegis, Google Authenticator o 1Password.',
            'settings.twofaStatus': 'Stato',
            'settings.twofaEnabled': 'Abilitata',
            'settings.twofaDisabled': 'Disabilitata',
            'settings.twofaEnable': 'Abilita 2FA',
            'settings.twofaDisable': 'Disabilita 2FA',
            'settings.twofaVerifyEnable': 'Verifica e abilita',
            'settings.twofaCancel': 'Annulla',
            'settings.twofaScan': 'Scansiona questo codice QR con la tua app di autenticazione:',
            'settings.twofaManual': 'Oppure inserisci manualmente questo segreto:',
            'settings.twofaCode': 'Inserisci il codice a 6 cifre',
            'settings.twofaEnabledMsg': 'Autenticazione a due fattori abilitata',
            'settings.twofaDisabledMsg': 'Autenticazione a due fattori disabilitata',
            'settings.twofaCodeRequired': 'Inserisci il codice a 6 cifre',
            'settings.twofaSetupFailed': 'Impossibile avviare la configurazione 2FA',
            'settings.sessions': 'Sessioni attive',
            'sessions.hint': 'Tutti i dispositivi che hanno effettuato l\'accesso al tuo account. Termina le sessioni che non riconosci.',
            'sessions.hintAdmin': 'Tutte le sessioni attive su questo server. Termina quelle che non riconosci.',
            'sessions.current': 'Questo dispositivo',
            'sessions.since': 'Accesso effettuato',
            'sessions.expires': 'Scade',
            'sessions.revoke': 'Termina sessione',
            'sessions.revokeOthers': 'Disconnetti tutti gli altri dispositivi',
            'sessions.revokedOthers': 'Altri dispositivi disconnessi: {n}',
            'sessions.revoked': 'Sessione terminata',
            'sessions.empty': 'Nessun\'altra sessione attiva.',
            'sessions.confirmOthers': 'Disconnettere tutti gli altri dispositivi?',
            'settings.activity': 'Registro attività',
            'settings.activityHint': 'Chi ha scaricato, caricato o effettuato l\'accesso, con indirizzo e ora. Le voci più vecchie del periodo di conservazione vengono rimosse automaticamente.',
            'activity.when': 'Quando',
            'activity.kind': 'Evento',
            'activity.who': 'Chi',
            'activity.share': 'Condivisione',
            'activity.from': 'Origine',
            'activity.detail': 'Dettaglio',
            'activity.empty': 'Ancora nessuna registrazione.',
            'activity.export': 'Esporta CSV',
            'activity.allKinds': 'Tutti gli eventi',
            'activity.loadMore': 'Carica altro',
            'activity.anonymous': 'anonimo',
            'activity.system': 'sistema',
            'activity.forShare': 'Attività',
            'activity.loadFailed': 'Impossibile caricare il registro attività',
            'activity.kind.share.created': 'Condivisione creata',
            'activity.kind.share.updated': 'Condivisione aggiornata',
            'activity.kind.share.deleted': 'Condivisione eliminata',
            'activity.kind.share.expired': 'Condivisione scaduta',
            'activity.kind.share.downloaded': 'Scaricato',
            'activity.kind.share.streamed': 'Riprodotto in streaming',
            'activity.kind.receive.uploaded': 'File ricevuto',
            'activity.kind.auth.login': 'Accesso',
            'activity.kind.auth.login_failed': 'Accesso non riuscito',
            'activity.kind.auth.locked': 'Account bloccato',
            'activity.kind.auth.logout': 'Disconnessione',
            'activity.kind.auth.setup': 'Configurazione / modifica 2FA',
            'activity.kind.session.revoked': 'Sessione revocata',
            'activity.kind.security': 'Sicurezza',
            'theme.light': 'Modalità chiara',
            'theme.dark': 'Modalità scura',
            'load.networkFailed': 'Impossibile caricare le impostazioni di rete',
            'load.restrictionsFailed': 'Impossibile caricare le restrizioni sui file',
            'load.webhookFailed': 'Impossibile caricare le impostazioni del webhook',
            'load.usersUnavailable': 'La gestione utenti non è disponibile',
            'load.twoFAFailed': 'Impossibile caricare le impostazioni 2FA',
            'load.apiKeysFailed': 'Impossibile caricare le chiavi API',
            'load.smtpFailed': 'Impossibile caricare le impostazioni SMTP',
            'role.admin': 'Amministratore',
            'role.user': 'Utente',
            'role.viewer': 'Visualizzatore',
            'settings.apiKeyNamePlaceholder': 'La mia chiave API',
            'common.copied': 'Copiato!',
            'smtp.host': 'Server SMTP',
            'smtp.port': 'Porta',
            'smtp.encryption': 'Crittografia',
            'smtp.encryptionNone': 'Nessuna',
            'smtp.username': 'Nome utente',
            'smtp.password': 'Password',
            'smtp.fromEmail': 'E-mail del mittente',
            'smtp.fromName': 'Nome del mittente',
            'auth.signInAgain': 'Accedi di nuovo',
            'shares.expiresWhen': 'Scade {when}',
            'activity.d.loginOk': 'Accesso riuscito ({role})',
            'activity.d.loginOkApi': 'Accesso tramite API riuscito ({role})',
            'activity.d.loginFailed': 'Tentativo di accesso non riuscito {n}/{max}',
            'activity.d.loginFailedApi': 'Tentativo di accesso tramite API non riuscito {n}/{max}',
            'activity.d.loginLocked': 'Bloccato dopo troppi tentativi falliti',
            'activity.d.loginLockedApi': 'Bloccato dopo troppi tentativi API falliti',
            'activity.d.twoFAInvalid': 'Codice 2FA mancante o non valido',
            'activity.d.twoFAInvalidApi': 'Codice 2FA mancante o non valido (API)',
            'activity.d.localAuthDisabled': 'Accesso con password disattivato (solo SSO)',
            'activity.d.csrfInvalid': 'Token di sicurezza non valido all’accesso',
            'activity.d.rateLimited': 'Troppi tentativi di accesso',
            'activity.d.loggedOut': 'Disconnesso',
            'activity.d.setupTokenInvalid': 'Configurazione rifiutata: token di configurazione non valido',
            'activity.d.setupDone': 'Configurazione iniziale completata',
            'activity.d.twoFAEnabled': '2FA attivata',
            'activity.d.twoFADisabled': '2FA disattivata',
            'activity.d.revokedAll': 'Tutte le altre sessioni terminate',
            'activity.d.revokedOne': 'Sessione {id} terminata ({email})',
            'activity.d.bulk': 'eliminazione multipla',
        },
        pt: {
            'common.breadcrumb': 'Trilha de navegação',
            'common.copy': 'Copiar',
            'common.copyLink': 'Copiar link',
            'common.folder': 'Pasta',
            'common.expiring': 'Expira em breve',
            'common.autoShare': 'Compart. auto.',
            'common.qr': 'Código QR',
            'common.edit': 'Editar',
            'common.delete': 'Excluir',
            'common.remove': 'Remover',
            'common.close': 'Fechar',
            'common.refresh': 'Atualizar',
            'nav.upload': 'Enviar',
            'nav.shares': 'Compartilhamentos',
            'nav.receive': 'Receber',
            'nav.settings': 'Configurações',
            'nav.logout': 'Sair',
            'login.subtitle': 'Compartilhamento de arquivos auto-hospedado',
            'login.password': 'Senha',
            'login.submit': 'Entrar',
            'login.or': 'ou',
            'login.sso': 'Entrar com SSO',
            'upload.title': 'Enviar arquivos',
            'upload.dropText': 'Arraste arquivos aqui ou clique para selecionar',
            'upload.hint': 'Vários arquivos suportados',
            'upload.options': 'Opções de envio',
            'upload.password': 'Senha (opcional)',
            'upload.expiry': 'Expira em (horas)',
            'upload.maxDownloads': 'Downloads máx. (0 = ilimitado)',
            'upload.submit': 'Enviar',
            'upload.uploading': 'Enviando...',
            'upload.complete': 'Envio concluído',
            'upload.error': 'Falha no envio',
            'upload.networkError': 'Servidor inacessível',
            'upload.another': 'Enviar mais',
            'shares.title': 'Compartilhamentos ativos',
            'shares.empty': 'Nenhum compartilhamento ativo',
            'shares.downloads': 'downloads',
            'shares.expired': 'expirado',
            'shares.never': 'nunca',
            'shares.unlimited': 'ilimitado',
            'shares.expiryRequired': 'Indique um prazo em horas ou ative "ilimitado".',
            'shares.deleteConfirm': 'Excluir este compartilhamento?',
            'shares.copied': 'Link copiado!',
            'shares.size': 'tamanho',
            'shares.deleted': 'Compartilhamento excluído',
            'shares.edit': 'Editar compartilhamento',
            'shares.save': 'Salvar',
            'shares.updated': 'Compartilhamento atualizado',
            'shares.changePassword': 'alterar ou remover',
            'shares.addPassword': 'adicionar',
            'receive.title': 'Links de recebimento',
            'receive.create': 'Novo link',
            'receive.formTitle': 'Criar link de recebimento',
            'receive.name': 'Nome',
            'receive.password': 'Senha (opcional)',
            'receive.maxUploads': 'Envios máx. (0 = ilimitado)',
            'receive.maxFileSize': 'Tamanho máx. (MB, 0 = ilimitado)',
            'receive.expiry': 'Expira em (horas, 0 = nunca)',
            'receive.extensions': 'Extensões permitidas (ex: .pdf,.doc)',
            'receive.autoShare': 'Compartilhar automaticamente os arquivos recebidos',
            'receive.submit': 'Criar',
            'receive.cancel': 'Cancelar',
            'receive.empty': 'Nenhum link de recebimento',
            'receive.uploads': 'envios',
            'receive.deleteConfirm': 'Excluir este link de recebimento?',
            'receive.created': 'Link de recebimento criado',
            'receive.deleted': 'Link de recebimento excluído',
            'settings.title': 'Configurações',
            'settings.network': 'Configuração de rede',
            'settings.webhook': 'Webhook',
            'settings.users': 'Gerenciamento de usuários',
            'settings.primary': 'Principal',
            'settings.detected': 'Detectado',
            'settings.notDetected': 'Não detectado',
            'settings.enabled': 'Ativado',
            'settings.useDetected': 'Usar valor detectado',
            'settings.save': 'Salvar',
            'settings.saved': 'Configurações salvas',
            'settings.testWebhook': 'Testar webhook',
            'settings.webhookUrl': 'URL do webhook',
            'settings.webhookSecret': 'Segredo do webhook',
            'settings.webhookEnabled': 'Webhook ativado',
            'settings.webhookEvents': 'Notificar quando',
            'settings.webhookOnDownload': 'Arquivo baixado',
            'settings.webhookOnLimit': 'Limite de downloads atingido',
            'settings.webhookOnExpire': 'Compartilhamento expirado',
            'settings.webhookSecretKeep': 'inalterado — deixe vazio para manter',
            'settings.webhookSecretNone': 'nenhum segredo definido',
            'settings.webhookSecretClear': 'Remover o segredo guardado',
            'settings.webhookNeedsUrl': 'Introduza um URL de webhook ou desative o webhook',
            'settings.webhookSent': 'Webhook enviado',
            'settings.createUser': 'Criar usuário',
            'settings.email': 'E-mail',
            'settings.name': 'Nome',
            'settings.role': 'Função',
            'settings.userPassword': 'Senha',
            'settings.deleteUserConfirm': 'Excluir este usuário?',
            'settings.userCreated': 'Usuário criado',
            'settings.userDeleted': 'Usuário excluído',
            'settings.fillAll': 'Preencha todos os campos',
            'toast.copied': 'Link copiado para a área de transferência',
            'toast.error': 'Algo deu errado',
            'stat.totalShares': 'Compartilhamentos',
            'stat.totalDownloads': 'Downloads',
            'stat.totalSize': 'Tamanho total',
            'stat.protected': 'Protegidos',
            'nav.hostshare': 'Compartilhar do host',
            'hostshare.title': 'Compartilhar do host',
            'hostshare.hint': 'Compartilhe arquivos e pastas que já estão no servidor — sem enviar novamente.',
            'hostshare.home': 'Início',
            'hostshare.empty': 'Esta pasta está vazia',
            'hostshare.error': 'Não foi possível acessar este local',
            'hostshare.folder': 'Pasta',
            'hostshare.share': 'Compartilhar',
            'hostshare.dialogTitle': 'Compartilhar este item',
            'hostshare.symlink': 'Vincular em vez de copiar (sem espaço extra em disco)',
            'hostshare.success': 'Compartilhamento criado',
            'hostshare.done': 'Concluído',
            'upload.expiry.1h': '1 hora',
            'upload.expiry.6h': '6 horas',
            'upload.expiry.12h': '12 horas',
            'upload.expiry.1d': '1 dia',
            'upload.expiry.3d': '3 dias',
            'upload.expiry.7d': '7 dias',
            'upload.expiry.14d': '14 dias',
            'upload.expiry.30d': '30 dias',
            'upload.expiry.never': 'Nunca (ilimitado)',
            'upload.expiry.hours': 'horas',
            'settings.customHint': 'Informe seu próprio domínio fixo (ex.: proxy reverso ou domínio de um túnel nomeado da Cloudflare).',
            'settings.cfRotating': 'Gerando uma nova URL de compartilhamento da Cloudflare…',
            'settings.cfRotated': 'Nova URL da Cloudflare pronta:',
            'settings.cfRotateTimeout': 'Nenhuma nova URL da Cloudflare apareceu — o contêiner do túnel está em execução?',
            'settings.quota': 'Armazenamento / Cota',
            'settings.quotaGB': 'Cota (GB, 0 = ilimitado)',
            'settings.unlimited': 'ilimitado',
            'settings.quotaUpdated': 'Cota atualizada',
            'settings.userPasswordHint': 'Opcional — deixe em branco para contas somente SSO (mín. 8 caracteres, se definida)',
            'settings.passwordTooShort': 'A senha deve ter pelo menos 8 caracteres',
            'settings.maxFileSize': 'Tamanho máx. de arquivo',
            'settings.fileRestrictions': 'Restrições de arquivos',
            'settings.fileRestrictionsHint': 'Controle quais tipos de arquivo podem ser enviados. Extensões bloqueadas são rejeitadas; extensões permitidas formam uma lista de permissões (vazio = tudo permitido, exceto as bloqueadas).',
            'settings.blockedExtensions': 'Extensões bloqueadas',
            'settings.allowedExtensions': 'Extensões permitidas (lista de permissões)',
            'settings.allowedExtensionsHint': 'Vazio = tudo permitido (exceto as bloqueadas)',
            'shares.sendEmail': 'Enviar por e-mail',
            'shares.taildrop': 'Enviar para dispositivo (Taildrop)',
            'taildrop.title': 'Enviar para dispositivo',
            'taildrop.device': 'Dispositivo de destino',
            'taildrop.send': 'Enviar',
            'taildrop.sending': 'Enviando…',
            'taildrop.sent': 'Arquivo enviado via Taildrop',
            'taildrop.noDevices': 'Nenhum dispositivo Tailscale disponível',
            'shares.selectAll': 'Selecionar tudo',
            'shares.bulkDelete': 'Excluir selecionados',
            'shares.selectedCount': 'Selecionados: {n}',
            'email.recipientEmail': 'E-mail do destinatário',
            'email.recipientName': 'Nome do destinatário',
            'email.message': 'Mensagem',
            'email.optional': 'Opcional',
            'email.send': 'Enviar',
            'email.sent': 'E-mail enviado com sucesso',
            'email.enterRecipient': 'Informe o e-mail do destinatário',
            'email.sendFailed': 'Não foi possível enviar o e-mail:',
            'upload.folder': 'Enviar pasta',
            'settings.apiKeys': 'Chaves de API',
            'settings.apiKeysHint': 'Chaves de API permitem acesso programático ao CasaDrop. Use o cabeçalho X-API-Key.',
            'settings.noApiKeys': 'Nenhuma chave de API ainda',
            'settings.createApiKey': 'Criar chave',
            'settings.apiKeyCreated': 'Chave de API criada — copie agora!',
            'settings.apiKeyOnlyOnce': 'Esta chave será exibida apenas uma vez. Guarde-a em local seguro.',
            'settings.deleteApiKeyConfirm': 'Excluir esta chave de API?',
            'settings.apiKeyDeleted': 'Chave de API excluída',
            'settings.smtp': 'E-mail / SMTP',
            'settings.smtpEnabled': 'Ativar envio de e-mails',
            'settings.testSmtp': 'Testar conexão',
            'settings.smtpTestOk': 'Conexão SMTP bem-sucedida',
            'settings.twofa': 'Autenticação de dois fatores',
            'settings.twofaHint': 'Adicione uma senha de uso único baseada em tempo (TOTP) ao login de administrador. Use um app autenticador como Aegis, Google Authenticator ou 1Password.',
            'settings.twofaStatus': 'Status',
            'settings.twofaEnabled': 'Ativada',
            'settings.twofaDisabled': 'Desativada',
            'settings.twofaEnable': 'Ativar 2FA',
            'settings.twofaDisable': 'Desativar 2FA',
            'settings.twofaVerifyEnable': 'Verificar e ativar',
            'settings.twofaCancel': 'Cancelar',
            'settings.twofaScan': 'Escaneie este código QR com seu app autenticador:',
            'settings.twofaManual': 'Ou digite este segredo manualmente:',
            'settings.twofaCode': 'Digite o código de 6 dígitos',
            'settings.twofaEnabledMsg': 'Autenticação de dois fatores ativada',
            'settings.twofaDisabledMsg': 'Autenticação de dois fatores desativada',
            'settings.twofaCodeRequired': 'Digite o código de 6 dígitos',
            'settings.twofaSetupFailed': 'Não foi possível iniciar a configuração do 2FA',
            'settings.sessions': 'Sessões ativas',
            'sessions.hint': 'Todos os dispositivos conectados à sua conta. Encerre as sessões que você não reconhece.',
            'sessions.hintAdmin': 'Todas as sessões ativas neste servidor. Encerre as que você não reconhece.',
            'sessions.current': 'Este dispositivo',
            'sessions.since': 'Conectado em',
            'sessions.expires': 'Expira',
            'sessions.revoke': 'Encerrar sessão',
            'sessions.revokeOthers': 'Sair de todos os outros dispositivos',
            'sessions.revokedOthers': 'Desconectado de {n} outro(s) dispositivo(s)',
            'sessions.revoked': 'Sessão encerrada',
            'sessions.empty': 'Nenhuma outra sessão ativa.',
            'sessions.confirmOthers': 'Sair de todos os outros dispositivos?',
            'settings.activity': 'Registro de atividades',
            'settings.activityHint': 'Quem baixou, enviou, entrou — com endereço e horário. Entradas mais antigas que o período de retenção são removidas automaticamente.',
            'activity.when': 'Quando',
            'activity.kind': 'Evento',
            'activity.who': 'Quem',
            'activity.share': 'Compartilhamento',
            'activity.from': 'Origem',
            'activity.detail': 'Detalhe',
            'activity.empty': 'Nada registrado ainda.',
            'activity.export': 'Exportar CSV',
            'activity.allKinds': 'Todos os eventos',
            'activity.loadMore': 'Carregar mais',
            'activity.anonymous': 'anônimo',
            'activity.system': 'sistema',
            'activity.forShare': 'Atividade',
            'activity.loadFailed': 'Não foi possível carregar o registro de atividades',
            'activity.kind.share.created': 'Compartilhamento criado',
            'activity.kind.share.updated': 'Compartilhamento atualizado',
            'activity.kind.share.deleted': 'Compartilhamento excluído',
            'activity.kind.share.expired': 'Compartilhamento expirado',
            'activity.kind.share.downloaded': 'Baixado',
            'activity.kind.share.streamed': 'Reproduzido',
            'activity.kind.receive.uploaded': 'Arquivo recebido',
            'activity.kind.auth.login': 'Login efetuado',
            'activity.kind.auth.login_failed': 'Falha no login',
            'activity.kind.auth.locked': 'Bloqueado',
            'activity.kind.auth.logout': 'Logout efetuado',
            'activity.kind.auth.setup': 'Configuração / alteração de 2FA',
            'activity.kind.session.revoked': 'Sessão revogada',
            'activity.kind.security': 'Segurança',
            'theme.light': 'Modo claro',
            'theme.dark': 'Modo escuro',
            'load.networkFailed': 'Não foi possível carregar as configurações de rede',
            'load.restrictionsFailed': 'Não foi possível carregar as restrições de arquivos',
            'load.webhookFailed': 'Não foi possível carregar as configurações do webhook',
            'load.usersUnavailable': 'O gerenciamento de usuários não está disponível',
            'load.twoFAFailed': 'Não foi possível carregar as configurações de 2FA',
            'load.apiKeysFailed': 'Não foi possível carregar as chaves de API',
            'load.smtpFailed': 'Não foi possível carregar as configurações SMTP',
            'role.admin': 'Administrador',
            'role.user': 'Usuário',
            'role.viewer': 'Visualizador',
            'settings.apiKeyNamePlaceholder': 'Minha chave de API',
            'common.copied': 'Copiado!',
            'smtp.host': 'Servidor SMTP',
            'smtp.port': 'Porta',
            'smtp.encryption': 'Criptografia',
            'smtp.encryptionNone': 'Nenhuma',
            'smtp.username': 'Nome de usuário',
            'smtp.password': 'Senha',
            'smtp.fromEmail': 'E-mail do remetente',
            'smtp.fromName': 'Nome do remetente',
            'auth.signInAgain': 'Faça login novamente',
            'shares.expiresWhen': 'Expira {when}',
            'activity.d.loginOk': 'Login realizado ({role})',
            'activity.d.loginOkApi': 'Login via API realizado ({role})',
            'activity.d.loginFailed': 'Tentativa de login falhou {n}/{max}',
            'activity.d.loginFailedApi': 'Tentativa de login via API falhou {n}/{max}',
            'activity.d.loginLocked': 'Bloqueado após muitas tentativas falhas',
            'activity.d.loginLockedApi': 'Bloqueado após muitas tentativas falhas via API',
            'activity.d.twoFAInvalid': 'Código 2FA ausente ou inválido',
            'activity.d.twoFAInvalidApi': 'Código 2FA ausente ou inválido (API)',
            'activity.d.localAuthDisabled': 'Login por senha desativado (somente SSO)',
            'activity.d.csrfInvalid': 'Token de segurança inválido no login',
            'activity.d.rateLimited': 'Muitas tentativas de login',
            'activity.d.loggedOut': 'Sessão encerrada',
            'activity.d.setupTokenInvalid': 'Configuração recusada: token de configuração inválido',
            'activity.d.setupDone': 'Configuração inicial concluída',
            'activity.d.twoFAEnabled': '2FA ativada',
            'activity.d.twoFADisabled': '2FA desativada',
            'activity.d.revokedAll': 'Todas as outras sessões encerradas',
            'activity.d.revokedOne': 'Sessão {id} encerrada ({email})',
            'activity.d.bulk': 'exclusão em massa',
        },
        nl: {
            'common.breadcrumb': 'Kruimelpad',
            'common.copy': 'Kopiëren',
            'common.copyLink': 'Link kopiëren',
            'common.folder': 'Map',
            'common.expiring': 'Verloopt binnenkort',
            'common.autoShare': 'Auto-delen',
            'common.qr': 'QR-code',
            'common.edit': 'Bewerken',
            'common.delete': 'Verwijderen',
            'common.remove': 'Weghalen',
            'common.close': 'Sluiten',
            'common.refresh': 'Vernieuwen',
            'nav.upload': 'Uploaden',
            'nav.shares': 'Delingen',
            'nav.receive': 'Ontvangen',
            'nav.settings': 'Instellingen',
            'nav.logout': 'Uitloggen',
            'login.subtitle': 'Zelfgehoste bestandsdeling',
            'login.password': 'Wachtwoord',
            'login.submit': 'Inloggen',
            'login.or': 'of',
            'login.sso': 'Inloggen met SSO',
            'upload.title': 'Bestanden uploaden',
            'upload.dropText': 'Sleep bestanden hierheen of klik om te bladeren',
            'upload.hint': 'Meerdere bestanden mogelijk',
            'upload.options': 'Uploadopties',
            'upload.password': 'Wachtwoord (optioneel)',
            'upload.expiry': 'Verloopt over (uren)',
            'upload.maxDownloads': 'Max downloads (0 = onbeperkt)',
            'upload.submit': 'Uploaden',
            'upload.uploading': 'Bezig met uploaden...',
            'upload.complete': 'Upload voltooid',
            'upload.error': 'Upload mislukt',
            'upload.networkError': 'Server niet bereikbaar',
            'upload.another': 'Meer uploaden',
            'shares.title': 'Actieve delingen',
            'shares.empty': 'Nog geen delingen',
            'shares.downloads': 'downloads',
            'shares.expired': 'verlopen',
            'shares.never': 'nooit',
            'shares.unlimited': 'onbeperkt',
            'shares.expiryRequired': 'Voer een vervaltijd in uren in of zet "onbeperkt" aan.',
            'shares.deleteConfirm': 'Deze deling verwijderen?',
            'shares.copied': 'Link gekopieerd!',
            'shares.size': 'grootte',
            'shares.deleted': 'Deling verwijderd',
            'shares.edit': 'Deling bewerken',
            'shares.save': 'Opslaan',
            'shares.updated': 'Deling bijgewerkt',
            'shares.changePassword': 'wijzigen of verwijderen',
            'shares.addPassword': 'toevoegen',
            'receive.title': 'Ontvangstlinks',
            'receive.create': 'Nieuwe link',
            'receive.formTitle': 'Ontvangstlink aanmaken',
            'receive.name': 'Naam',
            'receive.password': 'Wachtwoord (optioneel)',
            'receive.maxUploads': 'Max uploads (0 = onbeperkt)',
            'receive.maxFileSize': 'Max bestandsgrootte (MB, 0 = onbeperkt)',
            'receive.expiry': 'Verloopt over (uren, 0 = nooit)',
            'receive.extensions': 'Toegestane extensies (bijv. .pdf,.doc)',
            'receive.autoShare': 'Ontvangen bestanden automatisch delen',
            'receive.submit': 'Aanmaken',
            'receive.cancel': 'Annuleren',
            'receive.empty': 'Nog geen ontvangstlinks',
            'receive.uploads': 'uploads',
            'receive.deleteConfirm': 'Deze ontvangstlink verwijderen?',
            'receive.created': 'Ontvangstlink aangemaakt',
            'receive.deleted': 'Ontvangstlink verwijderd',
            'settings.title': 'Instellingen',
            'settings.network': 'Netwerkconfiguratie',
            'settings.webhook': 'Webhook',
            'settings.users': 'Gebruikersbeheer',
            'settings.primary': 'Primair',
            'settings.detected': 'Gedetecteerd',
            'settings.notDetected': 'Niet gedetecteerd',
            'settings.enabled': 'Ingeschakeld',
            'settings.useDetected': 'Gebruik gedetecteerde waarde',
            'settings.save': 'Opslaan',
            'settings.saved': 'Instellingen opgeslagen',
            'settings.testWebhook': 'Webhook testen',
            'settings.webhookUrl': 'Webhook-URL',
            'settings.webhookSecret': 'Webhook-geheim',
            'settings.webhookEnabled': 'Webhook ingeschakeld',
            'settings.webhookEvents': 'Melden bij',
            'settings.webhookOnDownload': 'Bestand gedownload',
            'settings.webhookOnLimit': 'Downloadlimiet bereikt',
            'settings.webhookOnExpire': 'Deling verlopen',
            'settings.webhookSecretKeep': 'ongewijzigd — leeg laten om te behouden',
            'settings.webhookSecretNone': 'geen geheim ingesteld',
            'settings.webhookSecretClear': 'Opgeslagen geheim verwijderen',
            'settings.webhookNeedsUrl': 'Voer een webhook-URL in of schakel de webhook uit',
            'settings.webhookSent': 'Webhook verzonden',
            'settings.createUser': 'Gebruiker aanmaken',
            'settings.email': 'E-mail',
            'settings.name': 'Naam',
            'settings.role': 'Rol',
            'settings.userPassword': 'Wachtwoord',
            'settings.deleteUserConfirm': 'Deze gebruiker verwijderen?',
            'settings.userCreated': 'Gebruiker aangemaakt',
            'settings.userDeleted': 'Gebruiker verwijderd',
            'settings.fillAll': 'Vul alle velden in',
            'toast.copied': 'Link gekopieerd naar klembord',
            'toast.error': 'Er is iets misgegaan',
            'stat.totalShares': 'Delingen',
            'stat.totalDownloads': 'Downloads',
            'stat.totalSize': 'Totale grootte',
            'stat.protected': 'Beschermd',
            'nav.hostshare': 'Delen vanaf host',
            'hostshare.title': 'Delen vanaf host',
            'hostshare.hint': 'Deel bestanden en mappen die al op de server staan — zonder opnieuw te uploaden.',
            'hostshare.home': 'Start',
            'hostshare.empty': 'Deze map is leeg',
            'hostshare.error': 'Deze locatie is niet toegankelijk',
            'hostshare.folder': 'Map',
            'hostshare.share': 'Delen',
            'hostshare.dialogTitle': 'Dit item delen',
            'hostshare.symlink': 'Koppelen in plaats van kopiëren (geen extra schijfruimte)',
            'hostshare.success': 'Deling aangemaakt',
            'hostshare.done': 'Gereed',
            'upload.expiry.1h': '1 uur',
            'upload.expiry.6h': '6 uur',
            'upload.expiry.12h': '12 uur',
            'upload.expiry.1d': '1 dag',
            'upload.expiry.3d': '3 dagen',
            'upload.expiry.7d': '7 dagen',
            'upload.expiry.14d': '14 dagen',
            'upload.expiry.30d': '30 dagen',
            'upload.expiry.never': 'Nooit (onbeperkt)',
            'upload.expiry.hours': 'uur',
            'settings.customHint': 'Voer je eigen vaste domein in (bijv. reverse proxy of een domein van een Cloudflare named tunnel).',
            'settings.cfRotating': 'Nieuwe Cloudflare-deel-URL wordt aangemaakt…',
            'settings.cfRotated': 'Nieuwe Cloudflare-URL gereed:',
            'settings.cfRotateTimeout': 'Er is geen nieuwe Cloudflare-URL verschenen — draait de tunnelcontainer?',
            'settings.quota': 'Opslag / quotum',
            'settings.quotaGB': 'Quotum (GB, 0 = onbeperkt)',
            'settings.unlimited': 'onbeperkt',
            'settings.quotaUpdated': 'Quotum bijgewerkt',
            'settings.userPasswordHint': 'Optioneel — leeg laten voor accounts met alleen SSO (min. 8 tekens indien ingesteld)',
            'settings.passwordTooShort': 'Wachtwoord moet minimaal 8 tekens lang zijn',
            'settings.maxFileSize': 'Max bestandsgrootte',
            'settings.fileRestrictions': 'Bestandsbeperkingen',
            'settings.fileRestrictionsHint': 'Bepaal welke bestandstypen geüpload mogen worden. Geblokkeerde extensies worden geweigerd; toegestane extensies vormen een whitelist (leeg = alles toegestaan behalve geblokkeerde).',
            'settings.blockedExtensions': 'Geblokkeerde extensies',
            'settings.allowedExtensions': 'Toegestane extensies (whitelist)',
            'settings.allowedExtensionsHint': 'Leeg = alles toegestaan (behalve geblokkeerde)',
            'shares.sendEmail': 'Verzenden per e-mail',
            'shares.taildrop': 'Naar apparaat sturen (Taildrop)',
            'taildrop.title': 'Naar apparaat sturen',
            'taildrop.device': 'Doelapparaat',
            'taildrop.send': 'Verzenden',
            'taildrop.sending': 'Bezig met verzenden…',
            'taildrop.sent': 'Bestand verzonden via Taildrop',
            'taildrop.noDevices': 'Geen Tailscale-apparaten beschikbaar',
            'shares.selectAll': 'Alles selecteren',
            'shares.bulkDelete': 'Selectie verwijderen',
            'shares.selectedCount': '{n} geselecteerd',
            'email.recipientEmail': 'E-mailadres ontvanger',
            'email.recipientName': 'Naam ontvanger',
            'email.message': 'Bericht',
            'email.optional': 'Optioneel',
            'email.send': 'Verzenden',
            'email.sent': 'E-mail verzonden',
            'email.enterRecipient': 'Voer het e-mailadres van de ontvanger in',
            'email.sendFailed': 'E-mail kon niet worden verzonden:',
            'upload.folder': 'Map uploaden',
            'settings.apiKeys': 'API-sleutels',
            'settings.apiKeysHint': 'Met API-sleutels krijg je programmatische toegang tot CasaDrop. Gebruik de header X-API-Key.',
            'settings.noApiKeys': 'Nog geen API-sleutels',
            'settings.createApiKey': 'Sleutel aanmaken',
            'settings.apiKeyCreated': 'API-sleutel aangemaakt — kopieer hem nu!',
            'settings.apiKeyOnlyOnce': 'Deze sleutel wordt maar één keer getoond. Bewaar hem op een veilige plek.',
            'settings.deleteApiKeyConfirm': 'Deze API-sleutel verwijderen?',
            'settings.apiKeyDeleted': 'API-sleutel verwijderd',
            'settings.smtp': 'E-mail / SMTP',
            'settings.smtpEnabled': 'E-mailverzending inschakelen',
            'settings.testSmtp': 'Verbinding testen',
            'settings.smtpTestOk': 'SMTP-verbinding geslaagd',
            'settings.twofa': 'Tweestapsverificatie',
            'settings.twofaHint': 'Voeg een tijdgebonden eenmalig wachtwoord (TOTP) toe aan je beheerderslogin. Gebruik een authenticator-app zoals Aegis, Google Authenticator of 1Password.',
            'settings.twofaStatus': 'Status',
            'settings.twofaEnabled': 'Ingeschakeld',
            'settings.twofaDisabled': 'Uitgeschakeld',
            'settings.twofaEnable': '2FA inschakelen',
            'settings.twofaDisable': '2FA uitschakelen',
            'settings.twofaVerifyEnable': 'Verifiëren en inschakelen',
            'settings.twofaCancel': 'Annuleren',
            'settings.twofaScan': 'Scan deze QR-code met je authenticator-app:',
            'settings.twofaManual': 'Of voer deze geheime sleutel handmatig in:',
            'settings.twofaCode': 'Voer de 6-cijferige code in',
            'settings.twofaEnabledMsg': 'Tweestapsverificatie ingeschakeld',
            'settings.twofaDisabledMsg': 'Tweestapsverificatie uitgeschakeld',
            'settings.twofaCodeRequired': 'Voer de 6-cijferige code in',
            'settings.twofaSetupFailed': '2FA-installatie kon niet worden gestart',
            'settings.sessions': 'Actieve sessies',
            'sessions.hint': 'Alle apparaten die bij je account zijn ingelogd. Beëindig sessies die je niet herkent.',
            'sessions.hintAdmin': 'Alle actieve sessies op deze server. Beëindig sessies die je niet herkent.',
            'sessions.current': 'Dit apparaat',
            'sessions.since': 'Ingelogd',
            'sessions.expires': 'Verloopt',
            'sessions.revoke': 'Sessie beëindigen',
            'sessions.revokeOthers': 'Alle andere apparaten uitloggen',
            'sessions.revokedOthers': '{n} ander(e) apparaat/apparaten uitgelogd',
            'sessions.revoked': 'Sessie beëindigd',
            'sessions.empty': 'Geen andere actieve sessies.',
            'sessions.confirmOthers': 'Alle andere apparaten uitloggen?',
            'settings.activity': 'Activiteitenlogboek',
            'settings.activityHint': 'Wie heeft gedownload, geüpload, ingelogd — met adres en tijd. Items ouder dan de bewaartermijn worden automatisch verwijderd.',
            'activity.when': 'Wanneer',
            'activity.kind': 'Gebeurtenis',
            'activity.who': 'Wie',
            'activity.share': 'Deling',
            'activity.from': 'Vanaf',
            'activity.detail': 'Details',
            'activity.empty': 'Nog niets vastgelegd.',
            'activity.export': 'CSV exporteren',
            'activity.allKinds': 'Alle gebeurtenissen',
            'activity.loadMore': 'Meer laden',
            'activity.anonymous': 'anoniem',
            'activity.system': 'systeem',
            'activity.forShare': 'Activiteit',
            'activity.loadFailed': 'Activiteitenlogboek kon niet worden geladen',
            'activity.kind.share.created': 'Deling aangemaakt',
            'activity.kind.share.updated': 'Deling bijgewerkt',
            'activity.kind.share.deleted': 'Deling verwijderd',
            'activity.kind.share.expired': 'Deling verlopen',
            'activity.kind.share.downloaded': 'Gedownload',
            'activity.kind.share.streamed': 'Gestreamd',
            'activity.kind.receive.uploaded': 'Bestand ontvangen',
            'activity.kind.auth.login': 'Ingelogd',
            'activity.kind.auth.login_failed': 'Inloggen mislukt',
            'activity.kind.auth.locked': 'Geblokkeerd',
            'activity.kind.auth.logout': 'Uitgelogd',
            'activity.kind.auth.setup': 'Installatie / 2FA-wijziging',
            'activity.kind.session.revoked': 'Sessie ingetrokken',
            'activity.kind.security': 'Beveiliging',
            'theme.light': 'Lichte modus',
            'theme.dark': 'Donkere modus',
            'load.networkFailed': 'Netwerkinstellingen konden niet worden geladen',
            'load.restrictionsFailed': 'Bestandsbeperkingen konden niet worden geladen',
            'load.webhookFailed': 'Webhook-instellingen konden niet worden geladen',
            'load.usersUnavailable': 'Gebruikersbeheer is niet beschikbaar',
            'load.twoFAFailed': '2FA-instellingen konden niet worden geladen',
            'load.apiKeysFailed': 'API-sleutels konden niet worden geladen',
            'load.smtpFailed': 'SMTP-instellingen konden niet worden geladen',
            'role.admin': 'Beheerder',
            'role.user': 'Gebruiker',
            'role.viewer': 'Kijker',
            'settings.apiKeyNamePlaceholder': 'Mijn API-sleutel',
            'common.copied': 'Gekopieerd!',
            'smtp.host': 'SMTP-server',
            'smtp.port': 'Poort',
            'smtp.encryption': 'Versleuteling',
            'smtp.encryptionNone': 'Geen',
            'smtp.username': 'Gebruikersnaam',
            'smtp.password': 'Wachtwoord',
            'smtp.fromEmail': 'E-mailadres afzender',
            'smtp.fromName': 'Naam afzender',
            'auth.signInAgain': 'Log opnieuw in',
            'shares.expiresWhen': 'Verloopt {when}',
            'activity.d.loginOk': 'Ingelogd ({role})',
            'activity.d.loginOkApi': 'Ingelogd via API ({role})',
            'activity.d.loginFailed': 'Mislukte inlogpoging {n}/{max}',
            'activity.d.loginFailedApi': 'Mislukte inlogpoging via API {n}/{max}',
            'activity.d.loginLocked': 'Geblokkeerd na te veel mislukte pogingen',
            'activity.d.loginLockedApi': 'Geblokkeerd na te veel mislukte API-pogingen',
            'activity.d.twoFAInvalid': '2FA-code ontbreekt of is ongeldig',
            'activity.d.twoFAInvalidApi': '2FA-code ontbreekt of is ongeldig (API)',
            'activity.d.localAuthDisabled': 'Inloggen met wachtwoord uitgeschakeld (alleen SSO)',
            'activity.d.csrfInvalid': 'Ongeldig beveiligingstoken bij inloggen',
            'activity.d.rateLimited': 'Te veel inlogpogingen',
            'activity.d.loggedOut': 'Uitgelogd',
            'activity.d.setupTokenInvalid': 'Installatie geweigerd: ongeldig installatietoken',
            'activity.d.setupDone': 'Eerste installatie voltooid',
            'activity.d.twoFAEnabled': '2FA ingeschakeld',
            'activity.d.twoFADisabled': '2FA uitgeschakeld',
            'activity.d.revokedAll': 'Alle andere sessies beëindigd',
            'activity.d.revokedOne': 'Sessie {id} beëindigd ({email})',
            'activity.d.bulk': 'bulkverwijdering',
        },
        pl: {
            'common.breadcrumb': 'Ścieżka nawigacji',
            'common.copy': 'Kopiuj',
            'common.copyLink': 'Kopiuj link',
            'common.folder': 'Folder',
            'common.expiring': 'Wkrótce wygasa',
            'common.autoShare': 'Auto-udostępnianie',
            'common.qr': 'Kod QR',
            'common.edit': 'Edytuj',
            'common.delete': 'Usuń',
            'common.remove': 'Usuń z listy',
            'common.close': 'Zamknij',
            'common.refresh': 'Odśwież',
            'nav.upload': 'Wyślij',
            'nav.shares': 'Udostępnienia',
            'nav.receive': 'Odbierz',
            'nav.settings': 'Ustawienia',
            'nav.logout': 'Wyloguj',
            'login.subtitle': 'Samodzielnie hostowany sharing plików',
            'login.password': 'Hasło',
            'login.submit': 'Zaloguj się',
            'login.or': 'lub',
            'login.sso': 'Zaloguj przez SSO',
            'upload.title': 'Wyślij pliki',
            'upload.dropText': 'Przeciągnij pliki tutaj lub kliknij, aby przeglądać',
            'upload.hint': 'Obsługa wielu plików',
            'upload.options': 'Opcje wysyłania',
            'upload.password': 'Hasło (opcjonalne)',
            'upload.expiry': 'Wygasa za (godziny)',
            'upload.maxDownloads': 'Maks. pobrań (0 = bez limitu)',
            'upload.submit': 'Wyślij',
            'upload.uploading': 'Wysyłanie...',
            'upload.complete': 'Wysyłanie zakończone',
            'upload.error': 'Wysyłanie nie powiodło się',
            'upload.networkError': 'Serwer niedostępny',
            'upload.another': 'Wyślij więcej',
            'shares.title': 'Aktywne udostępnienia',
            'shares.empty': 'Brak aktywnych udostępnień',
            'shares.downloads': 'pobrania',
            'shares.expired': 'wygasło',
            'shares.never': 'nigdy',
            'shares.unlimited': 'bez limitu',
            'shares.expiryRequired': 'Podaj czas wygasania w godzinach lub włącz „bez limitu”.',
            'shares.deleteConfirm': 'Usunąć to udostępnienie?',
            'shares.copied': 'Link skopiowany!',
            'shares.size': 'rozmiar',
            'shares.deleted': 'Udostępnienie usunięte',
            'shares.edit': 'Edytuj udostępnienie',
            'shares.save': 'Zapisz',
            'shares.updated': 'Udostępnienie zaktualizowane',
            'shares.changePassword': 'zmień lub usuń',
            'shares.addPassword': 'dodaj',
            'receive.title': 'Linki odbioru',
            'receive.create': 'Nowy link',
            'receive.formTitle': 'Utwórz link odbioru',
            'receive.name': 'Nazwa',
            'receive.password': 'Hasło (opcjonalne)',
            'receive.maxUploads': 'Maks. wysłań (0 = bez limitu)',
            'receive.maxFileSize': 'Maks. rozmiar pliku (MB, 0 = bez limitu)',
            'receive.expiry': 'Wygasa za (godziny, 0 = nigdy)',
            'receive.extensions': 'Dozwolone rozszerzenia (np. .pdf,.doc)',
            'receive.autoShare': 'Automatycznie udostępniaj odebrane pliki',
            'receive.submit': 'Utwórz',
            'receive.cancel': 'Anuluj',
            'receive.empty': 'Brak linków odbioru',
            'receive.uploads': 'wysłania',
            'receive.deleteConfirm': 'Usunąć ten link odbioru?',
            'receive.created': 'Link odbioru utworzony',
            'receive.deleted': 'Link odbioru usunięty',
            'settings.title': 'Ustawienia',
            'settings.network': 'Konfiguracja sieci',
            'settings.webhook': 'Webhook',
            'settings.users': 'Zarządzanie użytkownikami',
            'settings.primary': 'Główny',
            'settings.detected': 'Wykryto',
            'settings.notDetected': 'Nie wykryto',
            'settings.enabled': 'Włączony',
            'settings.useDetected': 'Użyj wykrytej wartości',
            'settings.save': 'Zapisz',
            'settings.saved': 'Ustawienia zapisane',
            'settings.testWebhook': 'Testuj webhook',
            'settings.webhookUrl': 'URL webhooka',
            'settings.webhookSecret': 'Sekret webhooka',
            'settings.webhookEnabled': 'Webhook włączony',
            'settings.webhookEvents': 'Powiadom przy',
            'settings.webhookOnDownload': 'Pobrano plik',
            'settings.webhookOnLimit': 'Osiągnięto limit pobrań',
            'settings.webhookOnExpire': 'Udostępnienie wygasło',
            'settings.webhookSecretKeep': 'bez zmian — pozostaw puste, aby zachować',
            'settings.webhookSecretNone': 'brak ustawionego sekretu',
            'settings.webhookSecretClear': 'Usuń zapisany sekret',
            'settings.webhookNeedsUrl': 'Podaj adres URL webhooka lub wyłącz webhook',
            'settings.webhookSent': 'Webhook wysłany',
            'settings.createUser': 'Utwórz użytkownika',
            'settings.email': 'E-mail',
            'settings.name': 'Nazwa',
            'settings.role': 'Rola',
            'settings.userPassword': 'Hasło',
            'settings.deleteUserConfirm': 'Usunąć tego użytkownika?',
            'settings.userCreated': 'Użytkownik utworzony',
            'settings.userDeleted': 'Użytkownik usunięty',
            'settings.fillAll': 'Wypełnij wszystkie pola',
            'toast.copied': 'Link skopiowany do schowka',
            'toast.error': 'Coś poszło nie tak',
            'stat.totalShares': 'Udostępnienia',
            'stat.totalDownloads': 'Pobrania',
            'stat.totalSize': 'Łączny rozmiar',
            'stat.protected': 'Chronione',
            'nav.hostshare': 'Udostępnij z hosta',
            'hostshare.title': 'Udostępnij z hosta',
            'hostshare.hint': 'Udostępniaj pliki i foldery, które już są na serwerze — bez ponownego wysyłania.',
            'hostshare.home': 'Katalog główny',
            'hostshare.empty': 'Ten folder jest pusty',
            'hostshare.error': 'Brak dostępu do tej lokalizacji',
            'hostshare.folder': 'Folder',
            'hostshare.share': 'Udostępnij',
            'hostshare.dialogTitle': 'Udostępnij ten element',
            'hostshare.symlink': 'Dowiązanie zamiast kopii (bez dodatkowego miejsca na dysku)',
            'hostshare.success': 'Udostępnienie utworzone',
            'hostshare.done': 'Gotowe',
            'upload.expiry.1h': '1 godzina',
            'upload.expiry.6h': '6 godzin',
            'upload.expiry.12h': '12 godzin',
            'upload.expiry.1d': '1 dzień',
            'upload.expiry.3d': '3 dni',
            'upload.expiry.7d': '7 dni',
            'upload.expiry.14d': '14 dni',
            'upload.expiry.30d': '30 dni',
            'upload.expiry.never': 'Nigdy (bez limitu)',
            'upload.expiry.hours': 'godz.',
            'settings.customHint': 'Podaj własną stałą domenę (np. reverse proxy lub domenę nazwanego tunelu Cloudflare).',
            'settings.cfRotating': 'Generowanie nowego adresu URL udostępniania Cloudflare…',
            'settings.cfRotated': 'Nowy adres URL Cloudflare gotowy:',
            'settings.cfRotateTimeout': 'Nie pojawił się nowy adres URL Cloudflare — czy kontener tunelu działa?',
            'settings.quota': 'Miejsce / limit',
            'settings.quotaGB': 'Limit (GB, 0 = bez limitu)',
            'settings.unlimited': 'bez limitu',
            'settings.quotaUpdated': 'Limit zaktualizowany',
            'settings.userPasswordHint': 'Opcjonalne — pozostaw puste dla kont tylko z SSO (min. 8 znaków, jeśli ustawione)',
            'settings.passwordTooShort': 'Hasło musi mieć co najmniej 8 znaków',
            'settings.maxFileSize': 'Maks. rozmiar pliku',
            'settings.fileRestrictions': 'Ograniczenia plików',
            'settings.fileRestrictionsHint': 'Określ, jakie typy plików można wysyłać. Zablokowane rozszerzenia są odrzucane, dozwolone rozszerzenia tworzą białą listę (puste = wszystkie dozwolone oprócz zablokowanych).',
            'settings.blockedExtensions': 'Zablokowane rozszerzenia',
            'settings.allowedExtensions': 'Dozwolone rozszerzenia (biała lista)',
            'settings.allowedExtensionsHint': 'Puste = wszystkie dozwolone (oprócz zablokowanych)',
            'shares.sendEmail': 'Wyślij e-mailem',
            'shares.taildrop': 'Wyślij na urządzenie (Taildrop)',
            'taildrop.title': 'Wyślij na urządzenie',
            'taildrop.device': 'Urządzenie docelowe',
            'taildrop.send': 'Wyślij',
            'taildrop.sending': 'Wysyłanie…',
            'taildrop.sent': 'Plik wysłany przez Taildrop',
            'taildrop.noDevices': 'Brak dostępnych urządzeń Tailscale',
            'shares.selectAll': 'Zaznacz wszystko',
            'shares.bulkDelete': 'Usuń zaznaczone',
            'shares.selectedCount': 'Zaznaczono: {n}',
            'email.recipientEmail': 'E-mail odbiorcy',
            'email.recipientName': 'Nazwa odbiorcy',
            'email.message': 'Wiadomość',
            'email.optional': 'Opcjonalne',
            'email.send': 'Wyślij',
            'email.sent': 'E-mail został wysłany',
            'email.enterRecipient': 'Podaj e-mail odbiorcy',
            'email.sendFailed': 'Nie udało się wysłać e-maila:',
            'upload.folder': 'Wyślij folder',
            'settings.apiKeys': 'Klucze API',
            'settings.apiKeysHint': 'Klucze API umożliwiają programowy dostęp do CasaDrop. Użyj nagłówka X-API-Key.',
            'settings.noApiKeys': 'Brak kluczy API',
            'settings.createApiKey': 'Utwórz klucz',
            'settings.apiKeyCreated': 'Klucz API utworzony — skopiuj go teraz!',
            'settings.apiKeyOnlyOnce': 'Ten klucz zostanie pokazany tylko raz. Przechowuj go w bezpiecznym miejscu.',
            'settings.deleteApiKeyConfirm': 'Usunąć ten klucz API?',
            'settings.apiKeyDeleted': 'Klucz API usunięty',
            'settings.smtp': 'E-mail / SMTP',
            'settings.smtpEnabled': 'Włącz wysyłanie e-maili',
            'settings.testSmtp': 'Testuj połączenie',
            'settings.smtpTestOk': 'Połączenie SMTP działa',
            'settings.twofa': 'Uwierzytelnianie dwuskładnikowe',
            'settings.twofaHint': 'Dodaj jednorazowe hasło oparte na czasie (TOTP) do logowania administratora. Użyj aplikacji uwierzytelniającej, np. Aegis, Google Authenticator lub 1Password.',
            'settings.twofaStatus': 'Stan',
            'settings.twofaEnabled': 'Włączone',
            'settings.twofaDisabled': 'Wyłączone',
            'settings.twofaEnable': 'Włącz 2FA',
            'settings.twofaDisable': 'Wyłącz 2FA',
            'settings.twofaVerifyEnable': 'Zweryfikuj i włącz',
            'settings.twofaCancel': 'Anuluj',
            'settings.twofaScan': 'Zeskanuj ten kod QR aplikacją uwierzytelniającą:',
            'settings.twofaManual': 'Lub wpisz ten sekret ręcznie:',
            'settings.twofaCode': 'Wpisz 6-cyfrowy kod',
            'settings.twofaEnabledMsg': 'Uwierzytelnianie dwuskładnikowe włączone',
            'settings.twofaDisabledMsg': 'Uwierzytelnianie dwuskładnikowe wyłączone',
            'settings.twofaCodeRequired': 'Wpisz 6-cyfrowy kod',
            'settings.twofaSetupFailed': 'Nie udało się rozpocząć konfiguracji 2FA',
            'settings.sessions': 'Aktywne sesje',
            'sessions.hint': 'Wszystkie urządzenia zalogowane na Twoje konto. Zakończ sesję, której nie rozpoznajesz.',
            'sessions.hintAdmin': 'Wszystkie aktywne sesje na tym serwerze. Zakończ sesję, której nie rozpoznajesz.',
            'sessions.current': 'To urządzenie',
            'sessions.since': 'Zalogowano',
            'sessions.expires': 'Wygasa',
            'sessions.revoke': 'Zakończ sesję',
            'sessions.revokeOthers': 'Wyloguj wszystkie inne urządzenia',
            'sessions.revokedOthers': 'Wylogowano inne urządzenia: {n}',
            'sessions.revoked': 'Sesja zakończona',
            'sessions.empty': 'Brak innych aktywnych sesji.',
            'sessions.confirmOthers': 'Wylogować wszystkie inne urządzenia?',
            'settings.activity': 'Dziennik aktywności',
            'settings.activityHint': 'Kto pobierał, wysyłał, logował się — z adresem i czasem. Wpisy starsze niż okres przechowywania są usuwane automatycznie.',
            'activity.when': 'Kiedy',
            'activity.kind': 'Zdarzenie',
            'activity.who': 'Kto',
            'activity.share': 'Udostępnienie',
            'activity.from': 'Skąd',
            'activity.detail': 'Szczegóły',
            'activity.empty': 'Brak zapisanych zdarzeń.',
            'activity.export': 'Eksportuj CSV',
            'activity.allKinds': 'Wszystkie zdarzenia',
            'activity.loadMore': 'Wczytaj więcej',
            'activity.anonymous': 'anonimowo',
            'activity.system': 'system',
            'activity.forShare': 'Aktywność',
            'activity.loadFailed': 'Nie udało się wczytać dziennika aktywności',
            'activity.kind.share.created': 'Utworzono udostępnienie',
            'activity.kind.share.updated': 'Zaktualizowano udostępnienie',
            'activity.kind.share.deleted': 'Usunięto udostępnienie',
            'activity.kind.share.expired': 'Udostępnienie wygasło',
            'activity.kind.share.downloaded': 'Pobrano',
            'activity.kind.share.streamed': 'Odtworzono strumieniowo',
            'activity.kind.receive.uploaded': 'Odebrano plik',
            'activity.kind.auth.login': 'Zalogowano',
            'activity.kind.auth.login_failed': 'Nieudane logowanie',
            'activity.kind.auth.locked': 'Zablokowano dostęp',
            'activity.kind.auth.logout': 'Wylogowano',
            'activity.kind.auth.setup': 'Konfiguracja / zmiana 2FA',
            'activity.kind.session.revoked': 'Unieważniono sesję',
            'activity.kind.security': 'Bezpieczeństwo',
            'theme.light': 'Tryb jasny',
            'theme.dark': 'Tryb ciemny',
            'load.networkFailed': 'Nie udało się wczytać ustawień sieci',
            'load.restrictionsFailed': 'Nie udało się wczytać ograniczeń plików',
            'load.webhookFailed': 'Nie udało się wczytać ustawień webhooka',
            'load.usersUnavailable': 'Zarządzanie użytkownikami jest niedostępne',
            'load.twoFAFailed': 'Nie udało się wczytać ustawień 2FA',
            'load.apiKeysFailed': 'Nie udało się wczytać kluczy API',
            'load.smtpFailed': 'Nie udało się wczytać ustawień SMTP',
            'role.admin': 'Administrator',
            'role.user': 'Użytkownik',
            'role.viewer': 'Przeglądający',
            'settings.apiKeyNamePlaceholder': 'Mój klucz API',
            'common.copied': 'Skopiowano!',
            'smtp.host': 'Serwer SMTP',
            'smtp.port': 'Port',
            'smtp.encryption': 'Szyfrowanie',
            'smtp.encryptionNone': 'Brak',
            'smtp.username': 'Nazwa użytkownika',
            'smtp.password': 'Hasło',
            'smtp.fromEmail': 'E-mail nadawcy',
            'smtp.fromName': 'Nazwa nadawcy',
            'auth.signInAgain': 'Zaloguj się ponownie',
            'shares.expiresWhen': 'Wygasa {when}',
            'activity.d.loginOk': 'Zalogowano ({role})',
            'activity.d.loginOkApi': 'Zalogowano przez API ({role})',
            'activity.d.loginFailed': 'Nieudana próba logowania {n}/{max}',
            'activity.d.loginFailedApi': 'Nieudana próba logowania przez API {n}/{max}',
            'activity.d.loginLocked': 'Zablokowano po zbyt wielu nieudanych próbach',
            'activity.d.loginLockedApi': 'Zablokowano po zbyt wielu nieudanych próbach przez API',
            'activity.d.twoFAInvalid': 'Brak kodu 2FA lub kod nieprawidłowy',
            'activity.d.twoFAInvalidApi': 'Brak kodu 2FA lub kod nieprawidłowy (API)',
            'activity.d.localAuthDisabled': 'Logowanie hasłem wyłączone (tylko SSO)',
            'activity.d.csrfInvalid': 'Nieprawidłowy token bezpieczeństwa przy logowaniu',
            'activity.d.rateLimited': 'Zbyt wiele prób logowania',
            'activity.d.loggedOut': 'Wylogowano',
            'activity.d.setupTokenInvalid': 'Konfiguracja odrzucona: nieprawidłowy token konfiguracji',
            'activity.d.setupDone': 'Konfiguracja początkowa zakończona',
            'activity.d.twoFAEnabled': '2FA włączone',
            'activity.d.twoFADisabled': '2FA wyłączone',
            'activity.d.revokedAll': 'Zakończono wszystkie inne sesje',
            'activity.d.revokedOne': 'Zakończono sesję {id} ({email})',
            'activity.d.bulk': 'usuwanie zbiorcze',
        },
        ru: {
            'common.breadcrumb': 'Навигационная цепочка',
            'common.copy': 'Копировать',
            'common.copyLink': 'Копировать ссылку',
            'common.folder': 'Папка',
            'common.expiring': 'Скоро истекает',
            'common.autoShare': 'Автодоступ',
            'common.qr': 'QR-код',
            'common.edit': 'Изменить',
            'common.delete': 'Удалить',
            'common.remove': 'Убрать',
            'common.close': 'Закрыть',
            'common.refresh': 'Обновить',
            'nav.upload': 'Загрузить',
            'nav.shares': 'Общие файлы',
            'nav.receive': 'Получить',
            'nav.settings': 'Настройки',
            'nav.logout': 'Выйти',
            'login.subtitle': 'Самостоятельный хостинг файлов',
            'login.password': 'Пароль',
            'login.submit': 'Войти',
            'login.or': 'или',
            'login.sso': 'Войти через SSO',
            'upload.title': 'Загрузка файлов',
            'upload.dropText': 'Перетащите файлы сюда или нажмите для выбора',
            'upload.hint': 'Поддержка нескольких файлов',
            'upload.options': 'Параметры загрузки',
            'upload.password': 'Пароль (необязательно)',
            'upload.expiry': 'Истекает через (часы)',
            'upload.maxDownloads': 'Макс. скачиваний (0 = без ограничений)',
            'upload.submit': 'Загрузить',
            'upload.uploading': 'Загрузка...',
            'upload.complete': 'Загрузка завершена',
            'upload.error': 'Ошибка загрузки',
            'upload.networkError': 'Сервер недоступен',
            'upload.another': 'Загрузить ещё',
            'shares.title': 'Активные ссылки',
            'shares.empty': 'Пока нет общих файлов',
            'shares.downloads': 'скачивания',
            'shares.expired': 'истёк',
            'shares.never': 'никогда',
            'shares.unlimited': 'без ограничений',
            'shares.expiryRequired': 'Укажите срок в часах или включите «без ограничений».',
            'shares.deleteConfirm': 'Удалить эту ссылку?',
            'shares.copied': 'Ссылка скопирована!',
            'shares.size': 'размер',
            'shares.deleted': 'Ссылка удалена',
            'shares.edit': 'Редактировать',
            'shares.save': 'Сохранить',
            'shares.updated': 'Ссылка обновлена',
            'shares.changePassword': 'изменить или удалить',
            'shares.addPassword': 'добавить',
            'receive.title': 'Ссылки для приёма',
            'receive.create': 'Новая ссылка',
            'receive.formTitle': 'Создать ссылку для приёма',
            'receive.name': 'Название',
            'receive.password': 'Пароль (необязательно)',
            'receive.maxUploads': 'Макс. загрузок (0 = без ограничений)',
            'receive.maxFileSize': 'Макс. размер файла (МБ, 0 = без ограничений)',
            'receive.expiry': 'Истекает через (часы, 0 = никогда)',
            'receive.extensions': 'Разрешённые расширения (напр. .pdf,.doc)',
            'receive.autoShare': 'Автоматически делиться полученными файлами',
            'receive.submit': 'Создать',
            'receive.cancel': 'Отмена',
            'receive.empty': 'Пока нет ссылок для приёма',
            'receive.uploads': 'загрузки',
            'receive.deleteConfirm': 'Удалить эту ссылку для приёма?',
            'receive.created': 'Ссылка для приёма создана',
            'receive.deleted': 'Ссылка для приёма удалена',
            'settings.title': 'Настройки',
            'settings.network': 'Конфигурация сети',
            'settings.webhook': 'Вебхук',
            'settings.users': 'Управление пользователями',
            'settings.primary': 'Основной',
            'settings.detected': 'Обнаружено',
            'settings.notDetected': 'Не обнаружено',
            'settings.enabled': 'Включено',
            'settings.useDetected': 'Использовать обнаруженное',
            'settings.save': 'Сохранить',
            'settings.saved': 'Настройки сохранены',
            'settings.testWebhook': 'Тестировать вебхук',
            'settings.webhookUrl': 'URL вебхука',
            'settings.webhookSecret': 'Секрет вебхука',
            'settings.webhookEnabled': 'Вебхук включён',
            'settings.webhookEvents': 'Уведомлять при',
            'settings.webhookOnDownload': 'Файл скачан',
            'settings.webhookOnLimit': 'Достигнут лимит скачиваний',
            'settings.webhookOnExpire': 'Срок ссылки истёк',
            'settings.webhookSecretKeep': 'без изменений — оставьте пустым, чтобы сохранить',
            'settings.webhookSecretNone': 'секрет не задан',
            'settings.webhookSecretClear': 'Удалить сохранённый секрет',
            'settings.webhookNeedsUrl': 'Укажите URL вебхука или отключите вебхук',
            'settings.webhookSent': 'Вебхук отправлен',
            'settings.createUser': 'Создать пользователя',
            'settings.email': 'Эл. почта',
            'settings.name': 'Имя',
            'settings.role': 'Роль',
            'settings.userPassword': 'Пароль',
            'settings.deleteUserConfirm': 'Удалить этого пользователя?',
            'settings.userCreated': 'Пользователь создан',
            'settings.userDeleted': 'Пользователь удалён',
            'settings.fillAll': 'Заполните все поля',
            'toast.copied': 'Ссылка скопирована в буфер обмена',
            'toast.error': 'Что-то пошло не так',
            'stat.totalShares': 'Общие файлы',
            'stat.totalDownloads': 'Скачивания',
            'stat.totalSize': 'Общий размер',
            'stat.protected': 'Защищённые',
            'nav.hostshare': 'Поделиться с сервера',
            'hostshare.title': 'Поделиться с сервера',
            'hostshare.hint': 'Делитесь файлами и папками, которые уже лежат на сервере, — без повторной загрузки.',
            'hostshare.home': 'Начало',
            'hostshare.empty': 'Папка пуста',
            'hostshare.error': 'Нет доступа к этому расположению',
            'hostshare.folder': 'Папка',
            'hostshare.share': 'Поделиться',
            'hostshare.dialogTitle': 'Поделиться этим элементом',
            'hostshare.symlink': 'Ссылка вместо копии (без дополнительного места на диске)',
            'hostshare.success': 'Ссылка создана',
            'hostshare.done': 'Готово',
            'upload.expiry.1h': '1 час',
            'upload.expiry.6h': '6 часов',
            'upload.expiry.12h': '12 часов',
            'upload.expiry.1d': '1 день',
            'upload.expiry.3d': '3 дня',
            'upload.expiry.7d': '7 дней',
            'upload.expiry.14d': '14 дней',
            'upload.expiry.30d': '30 дней',
            'upload.expiry.never': 'Никогда (без ограничений)',
            'upload.expiry.hours': 'часов',
            'settings.customHint': 'Укажите собственный постоянный домен (например, обратный прокси или домен именованного туннеля Cloudflare).',
            'settings.cfRotating': 'Создаётся новый URL Cloudflare для общего доступа…',
            'settings.cfRotated': 'Новый URL Cloudflare готов:',
            'settings.cfRotateTimeout': 'Новый URL Cloudflare не появился — запущен ли контейнер туннеля?',
            'settings.quota': 'Хранилище / квота',
            'settings.quotaGB': 'Квота (ГБ, 0 = без ограничений)',
            'settings.unlimited': 'без ограничений',
            'settings.quotaUpdated': 'Квота обновлена',
            'settings.userPasswordHint': 'Необязательно — оставьте пустым для учётных записей только с SSO (если задан, не менее 8 символов)',
            'settings.passwordTooShort': 'Пароль должен содержать не менее 8 символов',
            'settings.maxFileSize': 'Макс. размер файла',
            'settings.fileRestrictions': 'Ограничения файлов',
            'settings.fileRestrictionsHint': 'Определяет, какие типы файлов можно загружать. Файлы с запрещёнными расширениями отклоняются; разрешённые расширения образуют белый список (пусто = разрешено всё, кроме запрещённого).',
            'settings.blockedExtensions': 'Запрещённые расширения',
            'settings.allowedExtensions': 'Разрешённые расширения (белый список)',
            'settings.allowedExtensionsHint': 'Пусто = разрешено всё (кроме запрещённого)',
            'shares.sendEmail': 'Отправить по эл. почте',
            'shares.taildrop': 'Отправить на устройство (Taildrop)',
            'taildrop.title': 'Отправить на устройство',
            'taildrop.device': 'Целевое устройство',
            'taildrop.send': 'Отправить',
            'taildrop.sending': 'Отправка…',
            'taildrop.sent': 'Файл отправлен через Taildrop',
            'taildrop.noDevices': 'Нет доступных устройств Tailscale',
            'shares.selectAll': 'Выбрать все',
            'shares.bulkDelete': 'Удалить выбранные',
            'shares.selectedCount': 'Выбрано: {n}',
            'email.recipientEmail': 'Эл. почта получателя',
            'email.recipientName': 'Имя получателя',
            'email.message': 'Сообщение',
            'email.optional': 'Необязательно',
            'email.send': 'Отправить',
            'email.sent': 'Письмо успешно отправлено',
            'email.enterRecipient': 'Введите адрес эл. почты получателя',
            'email.sendFailed': 'Не удалось отправить письмо:',
            'upload.folder': 'Загрузить папку',
            'settings.apiKeys': 'API-ключи',
            'settings.apiKeysHint': 'API-ключи дают программный доступ к CasaDrop. Используйте заголовок X-API-Key.',
            'settings.noApiKeys': 'API-ключей пока нет',
            'settings.createApiKey': 'Создать ключ',
            'settings.apiKeyCreated': 'API-ключ создан — скопируйте его сейчас!',
            'settings.apiKeyOnlyOnce': 'Этот ключ показывается только один раз. Сохраните его в надёжном месте.',
            'settings.deleteApiKeyConfirm': 'Удалить этот API-ключ?',
            'settings.apiKeyDeleted': 'API-ключ удалён',
            'settings.smtp': 'Эл. почта / SMTP',
            'settings.smtpEnabled': 'Включить отправку писем',
            'settings.testSmtp': 'Проверить подключение',
            'settings.smtpTestOk': 'Подключение к SMTP успешно',
            'settings.twofa': 'Двухфакторная аутентификация',
            'settings.twofaHint': 'Добавьте к входу администратора одноразовый пароль на основе времени (TOTP). Используйте приложение-аутентификатор, например Aegis, Google Authenticator или 1Password.',
            'settings.twofaStatus': 'Статус',
            'settings.twofaEnabled': 'Включена',
            'settings.twofaDisabled': 'Отключена',
            'settings.twofaEnable': 'Включить 2FA',
            'settings.twofaDisable': 'Отключить 2FA',
            'settings.twofaVerifyEnable': 'Проверить и включить',
            'settings.twofaCancel': 'Отмена',
            'settings.twofaScan': 'Отсканируйте этот QR-код приложением-аутентификатором:',
            'settings.twofaManual': 'Или введите этот секрет вручную:',
            'settings.twofaCode': 'Введите 6-значный код',
            'settings.twofaEnabledMsg': 'Двухфакторная аутентификация включена',
            'settings.twofaDisabledMsg': 'Двухфакторная аутентификация отключена',
            'settings.twofaCodeRequired': 'Введите 6-значный код',
            'settings.twofaSetupFailed': 'Не удалось начать настройку 2FA',
            'settings.sessions': 'Активные сеансы',
            'sessions.hint': 'Все устройства, на которых выполнен вход в вашу учётную запись. Завершите сеанс, который вам незнаком.',
            'sessions.hintAdmin': 'Все активные сеансы на этом сервере. Завершите сеанс, который вам незнаком.',
            'sessions.current': 'Это устройство',
            'sessions.since': 'Вход выполнен',
            'sessions.expires': 'Истекает',
            'sessions.revoke': 'Завершить сеанс',
            'sessions.revokeOthers': 'Выйти на всех других устройствах',
            'sessions.revokedOthers': 'Выполнен выход на других устройствах: {n}',
            'sessions.revoked': 'Сеанс завершён',
            'sessions.empty': 'Других активных сеансов нет.',
            'sessions.confirmOthers': 'Выйти на всех других устройствах?',
            'settings.activity': 'Журнал активности',
            'settings.activityHint': 'Кто скачивал, загружал, входил в систему — с адресом и временем. Записи старше срока хранения удаляются автоматически.',
            'activity.when': 'Когда',
            'activity.kind': 'Событие',
            'activity.who': 'Кто',
            'activity.share': 'Ссылка',
            'activity.from': 'Откуда',
            'activity.detail': 'Подробности',
            'activity.empty': 'Пока ничего не записано.',
            'activity.export': 'Экспорт в CSV',
            'activity.allKinds': 'Все события',
            'activity.loadMore': 'Загрузить ещё',
            'activity.anonymous': 'аноним',
            'activity.system': 'система',
            'activity.forShare': 'Активность',
            'activity.loadFailed': 'Не удалось загрузить журнал активности',
            'activity.kind.share.created': 'Ссылка создана',
            'activity.kind.share.updated': 'Ссылка изменена',
            'activity.kind.share.deleted': 'Ссылка удалена',
            'activity.kind.share.expired': 'Срок ссылки истёк',
            'activity.kind.share.downloaded': 'Скачано',
            'activity.kind.share.streamed': 'Воспроизведено онлайн',
            'activity.kind.receive.uploaded': 'Файл получен',
            'activity.kind.auth.login': 'Вход',
            'activity.kind.auth.login_failed': 'Неудачный вход',
            'activity.kind.auth.locked': 'Блокировка',
            'activity.kind.auth.logout': 'Выход',
            'activity.kind.auth.setup': 'Настройка / изменение 2FA',
            'activity.kind.session.revoked': 'Сеанс завершён',
            'activity.kind.security': 'Безопасность',
            'theme.light': 'Светлая тема',
            'theme.dark': 'Тёмная тема',
            'load.networkFailed': 'Не удалось загрузить настройки сети',
            'load.restrictionsFailed': 'Не удалось загрузить ограничения файлов',
            'load.webhookFailed': 'Не удалось загрузить настройки webhook',
            'load.usersUnavailable': 'Управление пользователями недоступно',
            'load.twoFAFailed': 'Не удалось загрузить настройки 2FA',
            'load.apiKeysFailed': 'Не удалось загрузить ключи API',
            'load.smtpFailed': 'Не удалось загрузить настройки SMTP',
            'role.admin': 'Администратор',
            'role.user': 'Пользователь',
            'role.viewer': 'Наблюдатель',
            'settings.apiKeyNamePlaceholder': 'Мой ключ API',
            'common.copied': 'Скопировано!',
            'smtp.host': 'SMTP-сервер',
            'smtp.port': 'Порт',
            'smtp.encryption': 'Шифрование',
            'smtp.encryptionNone': 'Нет',
            'smtp.username': 'Имя пользователя',
            'smtp.password': 'Пароль',
            'smtp.fromEmail': 'Email отправителя',
            'smtp.fromName': 'Имя отправителя',
            'auth.signInAgain': 'Войдите снова',
            'shares.expiresWhen': 'Истекает {when}',
            'activity.d.loginOk': 'Вход выполнен ({role})',
            'activity.d.loginOkApi': 'Вход через API выполнен ({role})',
            'activity.d.loginFailed': 'Неудачная попытка входа {n}/{max}',
            'activity.d.loginFailedApi': 'Неудачная попытка входа через API {n}/{max}',
            'activity.d.loginLocked': 'Заблокировано после слишком многих неудачных попыток',
            'activity.d.loginLockedApi': 'Заблокировано после слишком многих неудачных попыток через API',
            'activity.d.twoFAInvalid': 'Код 2FA отсутствует или неверен',
            'activity.d.twoFAInvalidApi': 'Код 2FA отсутствует или неверен (API)',
            'activity.d.localAuthDisabled': 'Вход по паролю отключён (только SSO)',
            'activity.d.csrfInvalid': 'Неверный токен безопасности при входе',
            'activity.d.rateLimited': 'Слишком много попыток входа',
            'activity.d.loggedOut': 'Выход выполнен',
            'activity.d.setupTokenInvalid': 'Настройка отклонена: неверный токен настройки',
            'activity.d.setupDone': 'Начальная настройка завершена',
            'activity.d.twoFAEnabled': '2FA включена',
            'activity.d.twoFADisabled': '2FA отключена',
            'activity.d.revokedAll': 'Все остальные сеансы завершены',
            'activity.d.revokedOne': 'Сеанс {id} завершён ({email})',
            'activity.d.bulk': 'массовое удаление',
        },
        ja: {
            'common.breadcrumb': 'パンくずリスト',
            'common.copy': 'コピー',
            'common.copyLink': 'リンクをコピー',
            'common.folder': 'フォルダー',
            'common.expiring': 'まもなく期限切れ',
            'common.autoShare': '自動共有',
            'common.qr': 'QRコード',
            'common.edit': '編集',
            'common.delete': '削除',
            'common.remove': '外す',
            'common.close': '閉じる',
            'common.refresh': '更新',
            'nav.upload': 'アップロード',
            'nav.shares': '共有',
            'nav.receive': '受信',
            'nav.settings': '設定',
            'nav.logout': 'ログアウト',
            'login.subtitle': 'セルフホスト型ファイル共有',
            'login.password': 'パスワード',
            'login.submit': 'ログイン',
            'login.or': 'または',
            'login.sso': 'SSOでログイン',
            'upload.title': 'ファイルをアップロード',
            'upload.dropText': 'ファイルをここにドラッグ＆ドロップ、またはクリックして選択',
            'upload.hint': '複数ファイル対応',
            'upload.options': 'アップロードオプション',
            'upload.password': 'パスワード（任意）',
            'upload.expiry': '有効期限（時間）',
            'upload.maxDownloads': '最大ダウンロード数（0 = 無制限）',
            'upload.submit': 'アップロード',
            'upload.uploading': 'アップロード中...',
            'upload.complete': 'アップロード完了',
            'upload.error': 'アップロード失敗',
            'upload.networkError': 'サーバーに接続できません',
            'upload.another': '追加アップロード',
            'shares.title': '有効な共有',
            'shares.empty': '共有はまだありません',
            'shares.downloads': 'ダウンロード',
            'shares.expired': '期限切れ',
            'shares.never': 'なし',
            'shares.unlimited': '無制限',
            'shares.expiryRequired': '有効期限を時間で入力するか、「無制限」をオンにしてください。',
            'shares.deleteConfirm': 'この共有を削除しますか？',
            'shares.copied': 'リンクをコピーしました！',
            'shares.size': 'サイズ',
            'shares.deleted': '共有を削除しました',
            'shares.edit': '共有を編集',
            'shares.save': '保存',
            'shares.updated': '共有を更新しました',
            'shares.changePassword': '変更または解除',
            'shares.addPassword': '追加',
            'receive.title': '受信リンク',
            'receive.create': '新規リンク',
            'receive.formTitle': '受信リンクを作成',
            'receive.name': '名前',
            'receive.password': 'パスワード（任意）',
            'receive.maxUploads': '最大アップロード数（0 = 無制限）',
            'receive.maxFileSize': '最大ファイルサイズ（MB、0 = 無制限）',
            'receive.expiry': '有効期限（時間、0 = 無期限）',
            'receive.extensions': '許可する拡張子（例: .pdf,.doc）',
            'receive.autoShare': '受信ファイルを自動的に共有',
            'receive.submit': '作成',
            'receive.cancel': 'キャンセル',
            'receive.empty': '受信リンクはまだありません',
            'receive.uploads': 'アップロード',
            'receive.deleteConfirm': 'この受信リンクを削除しますか？',
            'receive.created': '受信リンクを作成しました',
            'receive.deleted': '受信リンクを削除しました',
            'settings.title': '設定',
            'settings.network': 'ネットワーク設定',
            'settings.webhook': 'Webhook',
            'settings.users': 'ユーザー管理',
            'settings.primary': 'プライマリ',
            'settings.detected': '検出済み',
            'settings.notDetected': '未検出',
            'settings.enabled': '有効',
            'settings.useDetected': '検出値を使用',
            'settings.save': '保存',
            'settings.saved': '設定を保存しました',
            'settings.testWebhook': 'Webhookをテスト',
            'settings.webhookUrl': 'Webhook URL',
            'settings.webhookSecret': 'Webhookシークレット',
            'settings.webhookEnabled': 'Webhook を有効にする',
            'settings.webhookEvents': '通知するイベント',
            'settings.webhookOnDownload': 'ファイルがダウンロードされた',
            'settings.webhookOnLimit': 'ダウンロード上限に到達',
            'settings.webhookOnExpire': '共有の期限切れ',
            'settings.webhookSecretKeep': '変更なし — 保持するには空のまま',
            'settings.webhookSecretNone': 'シークレット未設定',
            'settings.webhookSecretClear': '保存済みシークレットを削除',
            'settings.webhookNeedsUrl': 'Webhook URL を入力するか、Webhook を無効にしてください',
            'settings.webhookSent': 'Webhookを送信しました',
            'settings.createUser': 'ユーザーを作成',
            'settings.email': 'メールアドレス',
            'settings.name': '名前',
            'settings.role': '役割',
            'settings.userPassword': 'パスワード',
            'settings.deleteUserConfirm': 'このユーザーを削除しますか？',
            'settings.userCreated': 'ユーザーを作成しました',
            'settings.userDeleted': 'ユーザーを削除しました',
            'settings.fillAll': 'すべてのフィールドを入力してください',
            'toast.copied': 'リンクをクリップボードにコピーしました',
            'toast.error': 'エラーが発生しました',
            'stat.totalShares': '共有数',
            'stat.totalDownloads': 'ダウンロード数',
            'stat.totalSize': '合計サイズ',
            'stat.protected': '保護済み',
            'nav.hostshare': 'ホストから共有',
            'hostshare.title': 'ホストから共有',
            'hostshare.hint': 'サーバー上にすでにあるファイルやフォルダーを、再アップロードせずに共有できます。',
            'hostshare.home': 'ホーム',
            'hostshare.empty': 'このフォルダーは空です',
            'hostshare.error': 'この場所にアクセスできません',
            'hostshare.folder': 'フォルダー',
            'hostshare.share': '共有',
            'hostshare.dialogTitle': 'この項目を共有',
            'hostshare.symlink': 'コピーせずにリンク（追加のディスク容量不要）',
            'hostshare.success': '共有を作成しました',
            'hostshare.done': '完了',
            'upload.expiry.1h': '1時間',
            'upload.expiry.6h': '6時間',
            'upload.expiry.12h': '12時間',
            'upload.expiry.1d': '1日',
            'upload.expiry.3d': '3日',
            'upload.expiry.7d': '7日',
            'upload.expiry.14d': '14日',
            'upload.expiry.30d': '30日',
            'upload.expiry.never': '無期限',
            'upload.expiry.hours': '時間',
            'settings.customHint': '独自の固定ドメインを入力してください（例: リバースプロキシや Cloudflare の名前付きトンネルのドメイン）。',
            'settings.cfRotating': '新しい Cloudflare 共有 URL を生成しています…',
            'settings.cfRotated': '新しい Cloudflare URL の準備ができました：',
            'settings.cfRotateTimeout': '新しい Cloudflare URL が表示されません。トンネルコンテナーは起動していますか？',
            'settings.quota': 'ストレージ / クォータ',
            'settings.quotaGB': 'クォータ（GB、0 = 無制限）',
            'settings.unlimited': '無制限',
            'settings.quotaUpdated': 'クォータを更新しました',
            'settings.userPasswordHint': '任意 — SSO専用アカウントの場合は空欄（設定する場合は8文字以上）',
            'settings.passwordTooShort': 'パスワードは8文字以上にしてください',
            'settings.maxFileSize': '最大ファイルサイズ',
            'settings.fileRestrictions': 'ファイル制限',
            'settings.fileRestrictionsHint': 'アップロードできるファイルの種類を制御します。ブロックした拡張子は拒否され、許可した拡張子はホワイトリストになります（空欄 = ブロック以外すべて許可）。',
            'settings.blockedExtensions': 'ブロックする拡張子',
            'settings.allowedExtensions': '許可する拡張子（ホワイトリスト）',
            'settings.allowedExtensionsHint': '空欄 = すべて許可（ブロック対象を除く）',
            'shares.sendEmail': 'メールで送信',
            'shares.taildrop': 'デバイスに送信（Taildrop）',
            'taildrop.title': 'デバイスに送信',
            'taildrop.device': '送信先デバイス',
            'taildrop.send': '送信',
            'taildrop.sending': '送信中…',
            'taildrop.sent': 'Taildrop でファイルを送信しました',
            'taildrop.noDevices': '利用できる Tailscale デバイスがありません',
            'shares.selectAll': 'すべて選択',
            'shares.bulkDelete': '選択した項目を削除',
            'shares.selectedCount': '{n} 件選択',
            'email.recipientEmail': '受信者のメールアドレス',
            'email.recipientName': '受信者名',
            'email.message': 'メッセージ',
            'email.optional': '任意',
            'email.send': '送信',
            'email.sent': 'メールを送信しました',
            'email.enterRecipient': '受信者のメールアドレスを入力',
            'email.sendFailed': 'メールを送信できませんでした：',
            'upload.folder': 'フォルダーをアップロード',
            'settings.apiKeys': 'APIキー',
            'settings.apiKeysHint': 'APIキーを使うと CasaDrop にプログラムからアクセスできます。X-API-Key ヘッダーを使用してください。',
            'settings.noApiKeys': 'APIキーはまだありません',
            'settings.createApiKey': 'キーを作成',
            'settings.apiKeyCreated': 'APIキーを作成しました — 今すぐコピーしてください！',
            'settings.apiKeyOnlyOnce': 'このキーは一度だけ表示されます。安全な場所に保管してください。',
            'settings.deleteApiKeyConfirm': 'このAPIキーを削除しますか？',
            'settings.apiKeyDeleted': 'APIキーを削除しました',
            'settings.smtp': 'メール / SMTP',
            'settings.smtpEnabled': 'メール送信を有効にする',
            'settings.testSmtp': '接続をテスト',
            'settings.smtpTestOk': 'SMTP接続に成功しました',
            'settings.twofa': '二要素認証',
            'settings.twofaHint': '管理者ログインに時間ベースのワンタイムパスワード（TOTP）を追加します。Aegis、Google Authenticator、1Password などの認証アプリを使用してください。',
            'settings.twofaStatus': 'ステータス',
            'settings.twofaEnabled': '有効',
            'settings.twofaDisabled': '無効',
            'settings.twofaEnable': '2FAを有効にする',
            'settings.twofaDisable': '2FAを無効にする',
            'settings.twofaVerifyEnable': '確認して有効化',
            'settings.twofaCancel': 'キャンセル',
            'settings.twofaScan': '認証アプリでこのQRコードをスキャンしてください：',
            'settings.twofaManual': 'または、このシークレットを手動で入力してください：',
            'settings.twofaCode': '6桁のコードを入力',
            'settings.twofaEnabledMsg': '二要素認証を有効にしました',
            'settings.twofaDisabledMsg': '二要素認証を無効にしました',
            'settings.twofaCodeRequired': '6桁のコードを入力してください',
            'settings.twofaSetupFailed': '2FAの設定を開始できませんでした',
            'settings.sessions': 'アクティブなセッション',
            'sessions.hint': 'あなたのアカウントにログインしているすべてのデバイスです。心当たりのないセッションは終了してください。',
            'sessions.hintAdmin': 'このサーバー上のすべてのアクティブなセッションです。心当たりのないものは終了してください。',
            'sessions.current': 'このデバイス',
            'sessions.since': 'ログイン日時',
            'sessions.expires': '有効期限',
            'sessions.revoke': 'セッションを終了',
            'sessions.revokeOthers': '他のすべてのデバイスからログアウト',
            'sessions.revokedOthers': '他の {n} 台のデバイスからログアウトしました',
            'sessions.revoked': 'セッションを終了しました',
            'sessions.empty': '他にアクティブなセッションはありません。',
            'sessions.confirmOthers': '他のすべてのデバイスからログアウトしますか？',
            'settings.activity': 'アクティビティログ',
            'settings.activityHint': 'ダウンロード、アップロード、ログインの実行者を、アドレスと日時とともに記録します。保存期間を過ぎたエントリーは自動的に削除されます。',
            'activity.when': '日時',
            'activity.kind': 'イベント',
            'activity.who': 'ユーザー',
            'activity.share': '共有',
            'activity.from': '送信元',
            'activity.detail': '詳細',
            'activity.empty': 'まだ記録はありません。',
            'activity.export': 'CSVをエクスポート',
            'activity.allKinds': 'すべてのイベント',
            'activity.loadMore': 'さらに読み込む',
            'activity.anonymous': '匿名',
            'activity.system': 'システム',
            'activity.forShare': 'アクティビティ',
            'activity.loadFailed': 'アクティビティログを読み込めませんでした',
            'activity.kind.share.created': '共有を作成',
            'activity.kind.share.updated': '共有を更新',
            'activity.kind.share.deleted': '共有を削除',
            'activity.kind.share.expired': '共有の期限切れ',
            'activity.kind.share.downloaded': 'ダウンロード',
            'activity.kind.share.streamed': 'ストリーミング再生',
            'activity.kind.receive.uploaded': 'ファイルを受信',
            'activity.kind.auth.login': 'ログイン',
            'activity.kind.auth.login_failed': 'ログイン失敗',
            'activity.kind.auth.locked': 'ロックアウト',
            'activity.kind.auth.logout': 'ログアウト',
            'activity.kind.auth.setup': 'セットアップ / 2FA変更',
            'activity.kind.session.revoked': 'セッション終了',
            'activity.kind.security': 'セキュリティ',
            'theme.light': 'ライトモード',
            'theme.dark': 'ダークモード',
            'load.networkFailed': 'ネットワーク設定を読み込めませんでした',
            'load.restrictionsFailed': 'ファイル制限を読み込めませんでした',
            'load.webhookFailed': 'Webhook 設定を読み込めませんでした',
            'load.usersUnavailable': 'ユーザー管理は利用できません',
            'load.twoFAFailed': '2FA 設定を読み込めませんでした',
            'load.apiKeysFailed': 'API キーを読み込めませんでした',
            'load.smtpFailed': 'SMTP 設定を読み込めませんでした',
            'role.admin': '管理者',
            'role.user': 'ユーザー',
            'role.viewer': '閲覧者',
            'settings.apiKeyNamePlaceholder': 'マイ API キー',
            'common.copied': 'コピーしました',
            'smtp.host': 'SMTP サーバー',
            'smtp.port': 'ポート',
            'smtp.encryption': '暗号化',
            'smtp.encryptionNone': 'なし',
            'smtp.username': 'ユーザー名',
            'smtp.password': 'パスワード',
            'smtp.fromEmail': '送信者のメールアドレス',
            'smtp.fromName': '送信者名',
            'auth.signInAgain': 'もう一度サインインしてください',
            'shares.expiresWhen': '{when}に期限切れ',
            'activity.d.loginOk': 'サインインしました（{role}）',
            'activity.d.loginOkApi': 'API でサインインしました（{role}）',
            'activity.d.loginFailed': 'サインイン失敗 {n}/{max}',
            'activity.d.loginFailedApi': 'API サインイン失敗 {n}/{max}',
            'activity.d.loginLocked': '失敗が多すぎるためロックされました',
            'activity.d.loginLockedApi': 'API での失敗が多すぎるためロックされました',
            'activity.d.twoFAInvalid': '2FA コードがないか無効です',
            'activity.d.twoFAInvalidApi': '2FA コードがないか無効です（API）',
            'activity.d.localAuthDisabled': 'パスワードでのサインインは無効です（SSO のみ）',
            'activity.d.csrfInvalid': 'サインイン時のセキュリティトークンが無効です',
            'activity.d.rateLimited': 'サインインの試行回数が多すぎます',
            'activity.d.loggedOut': 'サインアウトしました',
            'activity.d.setupTokenInvalid': 'セットアップ拒否：セットアップトークンが無効です',
            'activity.d.setupDone': '初期セットアップが完了しました',
            'activity.d.twoFAEnabled': '2FA を有効にしました',
            'activity.d.twoFADisabled': '2FA を無効にしました',
            'activity.d.revokedAll': '他のすべてのセッションを終了しました',
            'activity.d.revokedOne': 'セッション {id} を終了しました（{email}）',
            'activity.d.bulk': '一括削除',
        },
        zh: {
            'common.breadcrumb': '面包屑导航',
            'common.copy': '复制',
            'common.copyLink': '复制链接',
            'common.folder': '文件夹',
            'common.expiring': '即将过期',
            'common.autoShare': '自动分享',
            'common.qr': '二维码',
            'common.edit': '编辑',
            'common.delete': '删除',
            'common.remove': '移除',
            'common.close': '关闭',
            'common.refresh': '刷新',
            'nav.upload': '上传',
            'nav.shares': '共享',
            'nav.receive': '接收',
            'nav.settings': '设置',
            'nav.logout': '退出登录',
            'login.subtitle': '自托管文件共享',
            'login.password': '密码',
            'login.submit': '登录',
            'login.or': '或',
            'login.sso': '通过SSO登录',
            'upload.title': '上传文件',
            'upload.dropText': '将文件拖放到此处，或点击浏览',
            'upload.hint': '支持多个文件',
            'upload.options': '上传选项',
            'upload.password': '密码（可选）',
            'upload.expiry': '过期时间（小时）',
            'upload.maxDownloads': '最大下载次数（0 = 不限）',
            'upload.submit': '上传',
            'upload.uploading': '正在上传...',
            'upload.complete': '上传完成',
            'upload.error': '上传失败',
            'upload.networkError': '无法连接服务器',
            'upload.another': '继续上传',
            'shares.title': '活跃共享',
            'shares.empty': '暂无共享',
            'shares.downloads': '次下载',
            'shares.expired': '已过期',
            'shares.never': '永不',
            'shares.unlimited': '不限',
            'shares.expiryRequired': '请输入以小时为单位的有效期，或开启“不限”。',
            'shares.deleteConfirm': '确定删除此共享？',
            'shares.copied': '链接已复制！',
            'shares.size': '大小',
            'shares.deleted': '共享已删除',
            'shares.edit': '编辑共享',
            'shares.save': '保存',
            'shares.updated': '共享已更新',
            'shares.changePassword': '修改或移除',
            'shares.addPassword': '添加',
            'receive.title': '接收链接',
            'receive.create': '新建链接',
            'receive.formTitle': '创建接收链接',
            'receive.name': '名称',
            'receive.password': '密码（可选）',
            'receive.maxUploads': '最大上传数（0 = 不限）',
            'receive.maxFileSize': '最大文件大小（MB，0 = 不限）',
            'receive.expiry': '过期时间（小时，0 = 永不）',
            'receive.extensions': '允许的扩展名（如 .pdf,.doc）',
            'receive.autoShare': '自动共享接收的文件',
            'receive.submit': '创建',
            'receive.cancel': '取消',
            'receive.empty': '暂无接收链接',
            'receive.uploads': '次上传',
            'receive.deleteConfirm': '确定删除此接收链接？',
            'receive.created': '接收链接已创建',
            'receive.deleted': '接收链接已删除',
            'settings.title': '设置',
            'settings.network': '网络配置',
            'settings.webhook': 'Webhook',
            'settings.users': '用户管理',
            'settings.primary': '主要',
            'settings.detected': '已检测',
            'settings.notDetected': '未检测到',
            'settings.enabled': '已启用',
            'settings.useDetected': '使用检测值',
            'settings.save': '保存',
            'settings.saved': '设置已保存',
            'settings.testWebhook': '测试Webhook',
            'settings.webhookUrl': 'Webhook地址',
            'settings.webhookSecret': 'Webhook密钥',
            'settings.webhookEnabled': '启用 Webhook',
            'settings.webhookEvents': '通知事件',
            'settings.webhookOnDownload': '文件被下载',
            'settings.webhookOnLimit': '达到下载上限',
            'settings.webhookOnExpire': '分享已过期',
            'settings.webhookSecretKeep': '保持不变 — 留空即保留',
            'settings.webhookSecretNone': '未设置密钥',
            'settings.webhookSecretClear': '删除已保存的密钥',
            'settings.webhookNeedsUrl': '请填写 Webhook 地址，或关闭 Webhook',
            'settings.webhookSent': 'Webhook已发送',
            'settings.createUser': '创建用户',
            'settings.email': '电子邮箱',
            'settings.name': '名称',
            'settings.role': '角色',
            'settings.userPassword': '密码',
            'settings.deleteUserConfirm': '确定删除此用户？',
            'settings.userCreated': '用户已创建',
            'settings.userDeleted': '用户已删除',
            'settings.fillAll': '请填写所有字段',
            'toast.copied': '链接已复制到剪贴板',
            'toast.error': '出了点问题',
            'stat.totalShares': '共享数',
            'stat.totalDownloads': '下载次数',
            'stat.totalSize': '总大小',
            'stat.protected': '受保护',
            'nav.hostshare': '从主机共享',
            'hostshare.title': '从主机共享',
            'hostshare.hint': '共享服务器上已有的文件和文件夹，无需重新上传。',
            'hostshare.home': '首页',
            'hostshare.empty': '此文件夹为空',
            'hostshare.error': '无法访问此位置',
            'hostshare.folder': '文件夹',
            'hostshare.share': '共享',
            'hostshare.dialogTitle': '共享此项目',
            'hostshare.symlink': '链接而非复制（不占用额外磁盘空间）',
            'hostshare.success': '共享已创建',
            'hostshare.done': '完成',
            'upload.expiry.1h': '1 小时',
            'upload.expiry.6h': '6 小时',
            'upload.expiry.12h': '12 小时',
            'upload.expiry.1d': '1 天',
            'upload.expiry.3d': '3 天',
            'upload.expiry.7d': '7 天',
            'upload.expiry.14d': '14 天',
            'upload.expiry.30d': '30 天',
            'upload.expiry.never': '永不（不限）',
            'upload.expiry.hours': '小时',
            'settings.customHint': '输入您自己的固定域名（例如反向代理或 Cloudflare 命名隧道的域名）。',
            'settings.cfRotating': '正在生成新的 Cloudflare 共享 URL…',
            'settings.cfRotated': '新的 Cloudflare URL 已就绪：',
            'settings.cfRotateTimeout': '未出现新的 Cloudflare URL — 隧道容器是否在运行？',
            'settings.quota': '存储 / 配额',
            'settings.quotaGB': '配额（GB，0 = 不限）',
            'settings.unlimited': '不限',
            'settings.quotaUpdated': '配额已更新',
            'settings.userPasswordHint': '可选 — 仅使用 SSO 的账户请留空（如设置，至少 8 个字符）',
            'settings.passwordTooShort': '密码至少需要 8 个字符',
            'settings.maxFileSize': '最大文件大小',
            'settings.fileRestrictions': '文件限制',
            'settings.fileRestrictionsHint': '控制可上传的文件类型。被阻止的扩展名将被拒绝，允许的扩展名构成白名单（留空 = 除被阻止的外全部允许）。',
            'settings.blockedExtensions': '阻止的扩展名',
            'settings.allowedExtensions': '允许的扩展名（白名单）',
            'settings.allowedExtensionsHint': '留空 = 全部允许（被阻止的除外）',
            'shares.sendEmail': '通过电子邮件发送',
            'shares.taildrop': '发送到设备（Taildrop）',
            'taildrop.title': '发送到设备',
            'taildrop.device': '目标设备',
            'taildrop.send': '发送',
            'taildrop.sending': '正在发送…',
            'taildrop.sent': '文件已通过 Taildrop 发送',
            'taildrop.noDevices': '没有可用的 Tailscale 设备',
            'shares.selectAll': '全选',
            'shares.bulkDelete': '删除所选',
            'shares.selectedCount': '已选择 {n} 项',
            'email.recipientEmail': '收件人邮箱',
            'email.recipientName': '收件人姓名',
            'email.message': '留言',
            'email.optional': '可选',
            'email.send': '发送',
            'email.sent': '邮件发送成功',
            'email.enterRecipient': '请输入收件人邮箱',
            'email.sendFailed': '邮件发送失败：',
            'upload.folder': '上传文件夹',
            'settings.apiKeys': 'API 密钥',
            'settings.apiKeysHint': 'API 密钥用于以编程方式访问 CasaDrop。请使用 X-API-Key 请求头。',
            'settings.noApiKeys': '暂无 API 密钥',
            'settings.createApiKey': '创建密钥',
            'settings.apiKeyCreated': 'API 密钥已创建 — 请立即复制！',
            'settings.apiKeyOnlyOnce': '此密钥仅显示一次，请妥善保存。',
            'settings.deleteApiKeyConfirm': '确定删除此 API 密钥？',
            'settings.apiKeyDeleted': 'API 密钥已删除',
            'settings.smtp': '电子邮件 / SMTP',
            'settings.smtpEnabled': '启用邮件发送',
            'settings.testSmtp': '测试连接',
            'settings.smtpTestOk': 'SMTP 连接成功',
            'settings.twofa': '双因素认证',
            'settings.twofaHint': '为管理员登录添加基于时间的一次性密码（TOTP）。请使用 Aegis、Google Authenticator 或 1Password 等身份验证器应用。',
            'settings.twofaStatus': '状态',
            'settings.twofaEnabled': '已启用',
            'settings.twofaDisabled': '已停用',
            'settings.twofaEnable': '启用 2FA',
            'settings.twofaDisable': '停用 2FA',
            'settings.twofaVerifyEnable': '验证并启用',
            'settings.twofaCancel': '取消',
            'settings.twofaScan': '请使用身份验证器应用扫描此二维码：',
            'settings.twofaManual': '或手动输入此密钥：',
            'settings.twofaCode': '输入 6 位验证码',
            'settings.twofaEnabledMsg': '双因素认证已启用',
            'settings.twofaDisabledMsg': '双因素认证已停用',
            'settings.twofaCodeRequired': '请输入 6 位验证码',
            'settings.twofaSetupFailed': '无法开始设置 2FA',
            'settings.sessions': '活动会话',
            'sessions.hint': '已登录您账户的所有设备。如有您不认识的会话，请将其结束。',
            'sessions.hintAdmin': '此服务器上的所有活动会话。如有您不认识的会话，请将其结束。',
            'sessions.current': '此设备',
            'sessions.since': '登录时间',
            'sessions.expires': '过期时间',
            'sessions.revoke': '结束会话',
            'sessions.revokeOthers': '退出所有其他设备',
            'sessions.revokedOthers': '已退出 {n} 台其他设备',
            'sessions.revoked': '会话已结束',
            'sessions.empty': '没有其他活动会话。',
            'sessions.confirmOthers': '确定退出所有其他设备？',
            'settings.activity': '活动日志',
            'settings.activityHint': '记录谁下载、上传、登录，以及地址和时间。超过保留期限的条目会被自动删除。',
            'activity.when': '时间',
            'activity.kind': '事件',
            'activity.who': '用户',
            'activity.share': '共享',
            'activity.from': '来源',
            'activity.detail': '详情',
            'activity.empty': '暂无记录。',
            'activity.export': '导出 CSV',
            'activity.allKinds': '所有事件',
            'activity.loadMore': '加载更多',
            'activity.anonymous': '匿名',
            'activity.system': '系统',
            'activity.forShare': '活动',
            'activity.loadFailed': '无法加载活动日志',
            'activity.kind.share.created': '共享已创建',
            'activity.kind.share.updated': '共享已更新',
            'activity.kind.share.deleted': '共享已删除',
            'activity.kind.share.expired': '共享已过期',
            'activity.kind.share.downloaded': '已下载',
            'activity.kind.share.streamed': '已在线播放',
            'activity.kind.receive.uploaded': '已接收文件',
            'activity.kind.auth.login': '已登录',
            'activity.kind.auth.login_failed': '登录失败',
            'activity.kind.auth.locked': '已锁定',
            'activity.kind.auth.logout': '已退出登录',
            'activity.kind.auth.setup': '设置 / 2FA 变更',
            'activity.kind.session.revoked': '会话已结束',
            'activity.kind.security': '安全',
            'theme.light': '浅色模式',
            'theme.dark': '深色模式',
            'load.networkFailed': '无法加载网络设置',
            'load.restrictionsFailed': '无法加载文件限制',
            'load.webhookFailed': '无法加载 Webhook 设置',
            'load.usersUnavailable': '用户管理不可用',
            'load.twoFAFailed': '无法加载 2FA 设置',
            'load.apiKeysFailed': '无法加载 API 密钥',
            'load.smtpFailed': '无法加载 SMTP 设置',
            'role.admin': '管理员',
            'role.user': '用户',
            'role.viewer': '查看者',
            'settings.apiKeyNamePlaceholder': '我的 API 密钥',
            'common.copied': '已复制',
            'smtp.host': 'SMTP 服务器',
            'smtp.port': '端口',
            'smtp.encryption': '加密',
            'smtp.encryptionNone': '无',
            'smtp.username': '用户名',
            'smtp.password': '密码',
            'smtp.fromEmail': '发件人邮箱',
            'smtp.fromName': '发件人名称',
            'auth.signInAgain': '请重新登录',
            'shares.expiresWhen': '{when}过期',
            'activity.d.loginOk': '登录成功（{role}）',
            'activity.d.loginOkApi': '通过 API 登录成功（{role}）',
            'activity.d.loginFailed': '登录失败 {n}/{max}',
            'activity.d.loginFailedApi': 'API 登录失败 {n}/{max}',
            'activity.d.loginLocked': '失败次数过多，已锁定',
            'activity.d.loginLockedApi': 'API 失败次数过多，已锁定',
            'activity.d.twoFAInvalid': '2FA 代码缺失或无效',
            'activity.d.twoFAInvalidApi': '2FA 代码缺失或无效（API）',
            'activity.d.localAuthDisabled': '密码登录已禁用（仅 SSO）',
            'activity.d.csrfInvalid': '登录时安全令牌无效',
            'activity.d.rateLimited': '登录尝试次数过多',
            'activity.d.loggedOut': '已退出登录',
            'activity.d.setupTokenInvalid': '设置被拒绝：设置令牌无效',
            'activity.d.setupDone': '初始设置已完成',
            'activity.d.twoFAEnabled': '已启用 2FA',
            'activity.d.twoFADisabled': '已禁用 2FA',
            'activity.d.revokedAll': '已结束所有其他会话',
            'activity.d.revokedOne': '已结束会话 {id}（{email}）',
            'activity.d.bulk': '批量删除',
        },
        ko: {
            'common.breadcrumb': '탐색 경로',
            'common.copy': '복사',
            'common.copyLink': '링크 복사',
            'common.folder': '폴더',
            'common.expiring': '곧 만료',
            'common.autoShare': '자동 공유',
            'common.qr': 'QR 코드',
            'common.edit': '편집',
            'common.delete': '삭제',
            'common.remove': '제거',
            'common.close': '닫기',
            'common.refresh': '새로 고침',
            'nav.upload': '업로드',
            'nav.shares': '공유',
            'nav.receive': '수신',
            'nav.settings': '설정',
            'nav.logout': '로그아웃',
            'login.subtitle': '셀프 호스팅 파일 공유',
            'login.password': '비밀번호',
            'login.submit': '로그인',
            'login.or': '또는',
            'login.sso': 'SSO로 로그인',
            'upload.title': '파일 업로드',
            'upload.dropText': '파일을 여기에 끌어다 놓거나 클릭하여 선택',
            'upload.hint': '여러 파일 지원',
            'upload.options': '업로드 옵션',
            'upload.password': '비밀번호 (선택사항)',
            'upload.expiry': '만료 시간 (시간)',
            'upload.maxDownloads': '최대 다운로드 수 (0 = 무제한)',
            'upload.submit': '업로드',
            'upload.uploading': '업로드 중...',
            'upload.complete': '업로드 완료',
            'upload.error': '업로드 실패',
            'upload.networkError': '서버에 연결할 수 없습니다',
            'upload.another': '추가 업로드',
            'shares.title': '활성 공유',
            'shares.empty': '공유가 아직 없습니다',
            'shares.downloads': '다운로드',
            'shares.expired': '만료됨',
            'shares.never': '없음',
            'shares.unlimited': '무제한',
            'shares.expiryRequired': '만료 시간을 시간 단위로 입력하거나 "무제한"을 켜세요.',
            'shares.deleteConfirm': '이 공유를 삭제하시겠습니까?',
            'shares.copied': '링크가 복사되었습니다!',
            'shares.size': '크기',
            'shares.deleted': '공유가 삭제되었습니다',
            'shares.edit': '공유 편집',
            'shares.save': '저장',
            'shares.updated': '공유가 업데이트되었습니다',
            'shares.changePassword': '변경 또는 제거',
            'shares.addPassword': '추가',
            'receive.title': '수신 링크',
            'receive.create': '새 링크',
            'receive.formTitle': '수신 링크 만들기',
            'receive.name': '이름',
            'receive.password': '비밀번호 (선택사항)',
            'receive.maxUploads': '최대 업로드 수 (0 = 무제한)',
            'receive.maxFileSize': '최대 파일 크기 (MB, 0 = 무제한)',
            'receive.expiry': '만료 시간 (시간, 0 = 무기한)',
            'receive.extensions': '허용 확장자 (예: .pdf,.doc)',
            'receive.autoShare': '수신된 파일 자동 공유',
            'receive.submit': '만들기',
            'receive.cancel': '취소',
            'receive.empty': '수신 링크가 아직 없습니다',
            'receive.uploads': '업로드',
            'receive.deleteConfirm': '이 수신 링크를 삭제하시겠습니까?',
            'receive.created': '수신 링크가 생성되었습니다',
            'receive.deleted': '수신 링크가 삭제되었습니다',
            'settings.title': '설정',
            'settings.network': '네트워크 설정',
            'settings.webhook': 'Webhook',
            'settings.users': '사용자 관리',
            'settings.primary': '기본',
            'settings.detected': '감지됨',
            'settings.notDetected': '감지되지 않음',
            'settings.enabled': '활성화',
            'settings.useDetected': '감지된 값 사용',
            'settings.save': '저장',
            'settings.saved': '설정이 저장되었습니다',
            'settings.testWebhook': 'Webhook 테스트',
            'settings.webhookUrl': 'Webhook URL',
            'settings.webhookSecret': 'Webhook 비밀키',
            'settings.webhookEnabled': 'Webhook 사용',
            'settings.webhookEvents': '알림 이벤트',
            'settings.webhookOnDownload': '파일 다운로드됨',
            'settings.webhookOnLimit': '다운로드 한도 도달',
            'settings.webhookOnExpire': '공유 만료됨',
            'settings.webhookSecretKeep': '변경 없음 — 유지하려면 비워 두세요',
            'settings.webhookSecretNone': '비밀키 없음',
            'settings.webhookSecretClear': '저장된 비밀키 삭제',
            'settings.webhookNeedsUrl': 'Webhook URL을 입력하거나 Webhook을 끄세요',
            'settings.webhookSent': 'Webhook이 전송되었습니다',
            'settings.createUser': '사용자 만들기',
            'settings.email': '이메일',
            'settings.name': '이름',
            'settings.role': '역할',
            'settings.userPassword': '비밀번호',
            'settings.deleteUserConfirm': '이 사용자를 삭제하시겠습니까?',
            'settings.userCreated': '사용자가 생성되었습니다',
            'settings.userDeleted': '사용자가 삭제되었습니다',
            'settings.fillAll': '모든 필드를 입력해 주세요',
            'toast.copied': '링크가 클립보드에 복사되었습니다',
            'toast.error': '문제가 발생했습니다',
            'stat.totalShares': '공유 수',
            'stat.totalDownloads': '다운로드 수',
            'stat.totalSize': '총 크기',
            'stat.protected': '보호됨',
            'nav.hostshare': '호스트에서 공유',
            'hostshare.title': '호스트에서 공유',
            'hostshare.hint': '서버에 이미 있는 파일과 폴더를 다시 업로드하지 않고 공유합니다.',
            'hostshare.home': '홈',
            'hostshare.empty': '이 폴더는 비어 있습니다',
            'hostshare.error': '이 위치에 액세스할 수 없습니다',
            'hostshare.folder': '폴더',
            'hostshare.share': '공유',
            'hostshare.dialogTitle': '이 항목 공유',
            'hostshare.symlink': '복사 대신 링크 (추가 디스크 공간 없음)',
            'hostshare.success': '공유가 생성되었습니다',
            'hostshare.done': '완료',
            'upload.expiry.1h': '1시간',
            'upload.expiry.6h': '6시간',
            'upload.expiry.12h': '12시간',
            'upload.expiry.1d': '1일',
            'upload.expiry.3d': '3일',
            'upload.expiry.7d': '7일',
            'upload.expiry.14d': '14일',
            'upload.expiry.30d': '30일',
            'upload.expiry.never': '없음 (무제한)',
            'upload.expiry.hours': '시간',
            'settings.customHint': '고정 도메인을 직접 입력하세요 (예: 리버스 프록시 또는 Cloudflare 이름 지정 터널 도메인).',
            'settings.cfRotating': '새 Cloudflare 공유 URL을 생성하는 중…',
            'settings.cfRotated': '새 Cloudflare URL 준비 완료:',
            'settings.cfRotateTimeout': '새 Cloudflare URL이 나타나지 않았습니다. 터널 컨테이너가 실행 중인가요?',
            'settings.quota': '저장 공간 / 할당량',
            'settings.quotaGB': '할당량 (GB, 0 = 무제한)',
            'settings.unlimited': '무제한',
            'settings.quotaUpdated': '할당량이 업데이트되었습니다',
            'settings.userPasswordHint': '선택사항 — SSO 전용 계정은 비워 두세요 (설정 시 최소 8자)',
            'settings.passwordTooShort': '비밀번호는 8자 이상이어야 합니다',
            'settings.maxFileSize': '최대 파일 크기',
            'settings.fileRestrictions': '파일 제한',
            'settings.fileRestrictionsHint': '업로드할 수 있는 파일 형식을 지정합니다. 차단된 확장자는 거부되고, 허용 확장자를 입력하면 허용 목록이 됩니다 (비워 두면 차단된 확장자를 제외하고 모두 허용).',
            'settings.blockedExtensions': '차단된 확장자',
            'settings.allowedExtensions': '허용 확장자 (허용 목록)',
            'settings.allowedExtensionsHint': '비워 두면 모두 허용 (차단된 확장자 제외)',
            'shares.sendEmail': '이메일로 보내기',
            'shares.taildrop': '기기로 보내기 (Taildrop)',
            'taildrop.title': '기기로 보내기',
            'taildrop.device': '대상 기기',
            'taildrop.send': '보내기',
            'taildrop.sending': '보내는 중…',
            'taildrop.sent': 'Taildrop으로 파일을 보냈습니다',
            'taildrop.noDevices': '사용 가능한 Tailscale 기기가 없습니다',
            'shares.selectAll': '모두 선택',
            'shares.bulkDelete': '선택 항목 삭제',
            'shares.selectedCount': '{n}개 선택됨',
            'email.recipientEmail': '받는 사람 이메일',
            'email.recipientName': '받는 사람 이름',
            'email.message': '메시지',
            'email.optional': '선택사항',
            'email.send': '보내기',
            'email.sent': '이메일을 보냈습니다',
            'email.enterRecipient': '받는 사람 이메일을 입력하세요',
            'email.sendFailed': '이메일을 보내지 못했습니다:',
            'upload.folder': '폴더 업로드',
            'settings.apiKeys': 'API 키',
            'settings.apiKeysHint': 'API 키를 사용하면 프로그램에서 CasaDrop에 액세스할 수 있습니다. X-API-Key 헤더를 사용하세요.',
            'settings.noApiKeys': 'API 키가 아직 없습니다',
            'settings.createApiKey': '키 만들기',
            'settings.apiKeyCreated': 'API 키가 생성되었습니다 — 지금 복사하세요!',
            'settings.apiKeyOnlyOnce': '이 키는 한 번만 표시됩니다. 안전한 곳에 보관하세요.',
            'settings.deleteApiKeyConfirm': '이 API 키를 삭제하시겠습니까?',
            'settings.apiKeyDeleted': 'API 키가 삭제되었습니다',
            'settings.smtp': '이메일 / SMTP',
            'settings.smtpEnabled': '이메일 발송 사용',
            'settings.testSmtp': '연결 테스트',
            'settings.smtpTestOk': 'SMTP 연결에 성공했습니다',
            'settings.twofa': '2단계 인증',
            'settings.twofaHint': '관리자 로그인에 시간 기반 일회용 비밀번호(TOTP)를 추가합니다. Aegis, Google Authenticator, 1Password 같은 인증 앱을 사용하세요.',
            'settings.twofaStatus': '상태',
            'settings.twofaEnabled': '활성화됨',
            'settings.twofaDisabled': '비활성화됨',
            'settings.twofaEnable': '2FA 활성화',
            'settings.twofaDisable': '2FA 비활성화',
            'settings.twofaVerifyEnable': '확인 및 활성화',
            'settings.twofaCancel': '취소',
            'settings.twofaScan': '인증 앱으로 이 QR 코드를 스캔하세요:',
            'settings.twofaManual': '또는 이 비밀키를 직접 입력하세요:',
            'settings.twofaCode': '6자리 코드를 입력하세요',
            'settings.twofaEnabledMsg': '2단계 인증이 활성화되었습니다',
            'settings.twofaDisabledMsg': '2단계 인증이 비활성화되었습니다',
            'settings.twofaCodeRequired': '6자리 코드를 입력해 주세요',
            'settings.twofaSetupFailed': '2FA 설정을 시작할 수 없습니다',
            'settings.sessions': '활성 세션',
            'sessions.hint': '계정에 로그인된 모든 기기입니다. 알 수 없는 세션은 종료하세요.',
            'sessions.hintAdmin': '이 서버의 모든 활성 세션입니다. 알 수 없는 세션은 종료하세요.',
            'sessions.current': '이 기기',
            'sessions.since': '로그인 시각',
            'sessions.expires': '만료',
            'sessions.revoke': '세션 종료',
            'sessions.revokeOthers': '다른 모든 기기에서 로그아웃',
            'sessions.revokedOthers': '다른 기기 {n}대에서 로그아웃했습니다',
            'sessions.revoked': '세션이 종료되었습니다',
            'sessions.empty': '다른 활성 세션이 없습니다.',
            'sessions.confirmOthers': '다른 모든 기기에서 로그아웃하시겠습니까?',
            'settings.activity': '활동 로그',
            'settings.activityHint': '누가 다운로드, 업로드, 로그인했는지 주소와 시간과 함께 기록합니다. 보존 기간이 지난 항목은 자동으로 삭제됩니다.',
            'activity.when': '시간',
            'activity.kind': '이벤트',
            'activity.who': '사용자',
            'activity.share': '공유',
            'activity.from': '출처',
            'activity.detail': '세부 정보',
            'activity.empty': '아직 기록된 내용이 없습니다.',
            'activity.export': 'CSV 내보내기',
            'activity.allKinds': '모든 이벤트',
            'activity.loadMore': '더 불러오기',
            'activity.anonymous': '익명',
            'activity.system': '시스템',
            'activity.forShare': '활동',
            'activity.loadFailed': '활동 로그를 불러오지 못했습니다',
            'activity.kind.share.created': '공유 생성됨',
            'activity.kind.share.updated': '공유 업데이트됨',
            'activity.kind.share.deleted': '공유 삭제됨',
            'activity.kind.share.expired': '공유 만료됨',
            'activity.kind.share.downloaded': '다운로드됨',
            'activity.kind.share.streamed': '스트리밍됨',
            'activity.kind.receive.uploaded': '파일 수신됨',
            'activity.kind.auth.login': '로그인',
            'activity.kind.auth.login_failed': '로그인 실패',
            'activity.kind.auth.locked': '계정 잠김',
            'activity.kind.auth.logout': '로그아웃',
            'activity.kind.auth.setup': '설정 / 2FA 변경',
            'activity.kind.session.revoked': '세션 취소됨',
            'activity.kind.security': '보안',
            'theme.light': '라이트 모드',
            'theme.dark': '다크 모드',
            'load.networkFailed': '네트워크 설정을 불러오지 못했습니다',
            'load.restrictionsFailed': '파일 제한을 불러오지 못했습니다',
            'load.webhookFailed': 'Webhook 설정을 불러오지 못했습니다',
            'load.usersUnavailable': '사용자 관리를 사용할 수 없습니다',
            'load.twoFAFailed': '2FA 설정을 불러오지 못했습니다',
            'load.apiKeysFailed': 'API 키를 불러오지 못했습니다',
            'load.smtpFailed': 'SMTP 설정을 불러오지 못했습니다',
            'role.admin': '관리자',
            'role.user': '사용자',
            'role.viewer': '뷰어',
            'settings.apiKeyNamePlaceholder': '내 API 키',
            'common.copied': '복사됨',
            'smtp.host': 'SMTP 서버',
            'smtp.port': '포트',
            'smtp.encryption': '암호화',
            'smtp.encryptionNone': '없음',
            'smtp.username': '사용자 이름',
            'smtp.password': '비밀번호',
            'smtp.fromEmail': '보낸 사람 이메일',
            'smtp.fromName': '보낸 사람 이름',
            'auth.signInAgain': '다시 로그인하세요',
            'shares.expiresWhen': '{when} 만료',
            'activity.d.loginOk': '로그인함 ({role})',
            'activity.d.loginOkApi': 'API로 로그인함 ({role})',
            'activity.d.loginFailed': '로그인 실패 {n}/{max}',
            'activity.d.loginFailedApi': 'API 로그인 실패 {n}/{max}',
            'activity.d.loginLocked': '실패 횟수가 너무 많아 잠김',
            'activity.d.loginLockedApi': 'API 실패 횟수가 너무 많아 잠김',
            'activity.d.twoFAInvalid': '2FA 코드가 없거나 잘못됨',
            'activity.d.twoFAInvalidApi': '2FA 코드가 없거나 잘못됨 (API)',
            'activity.d.localAuthDisabled': '비밀번호 로그인 비활성화 (SSO 전용)',
            'activity.d.csrfInvalid': '로그인 시 보안 토큰이 잘못됨',
            'activity.d.rateLimited': '로그인 시도가 너무 많음',
            'activity.d.loggedOut': '로그아웃함',
            'activity.d.setupTokenInvalid': '설정 거부됨: 설정 토큰이 잘못됨',
            'activity.d.setupDone': '초기 설정 완료',
            'activity.d.twoFAEnabled': '2FA 활성화됨',
            'activity.d.twoFADisabled': '2FA 비활성화됨',
            'activity.d.revokedAll': '다른 모든 세션 종료됨',
            'activity.d.revokedOne': '세션 {id} 종료됨 ({email})',
            'activity.d.bulk': '일괄 삭제',
        },
        tr: {
            'common.breadcrumb': 'Gezinme yolu',
            'common.copy': 'Kopyala',
            'common.copyLink': 'Bağlantıyı kopyala',
            'common.folder': 'Klasör',
            'common.expiring': 'Yakında sona eriyor',
            'common.autoShare': 'Otomatik paylaşım',
            'common.qr': 'QR kodu',
            'common.edit': 'Düzenle',
            'common.delete': 'Sil',
            'common.remove': 'Kaldır',
            'common.close': 'Kapat',
            'common.refresh': 'Yenile',
            'nav.upload': 'Yükle',
            'nav.shares': 'Paylaşımlar',
            'nav.receive': 'Al',
            'nav.settings': 'Ayarlar',
            'nav.logout': 'Çıkış',
            'login.subtitle': 'Kendi sunucunuzda dosya paylaşımı',
            'login.password': 'Şifre',
            'login.submit': 'Giriş Yap',
            'login.or': 'veya',
            'login.sso': 'SSO ile Giriş',
            'upload.title': 'Dosya Yükle',
            'upload.dropText': 'Dosyaları buraya sürükleyin veya tıklayarak seçin',
            'upload.hint': 'Birden fazla dosya desteklenir',
            'upload.options': 'Yükleme Seçenekleri',
            'upload.password': 'Şifre (isteğe bağlı)',
            'upload.expiry': 'Süre (saat)',
            'upload.maxDownloads': 'Maks. indirme (0 = sınırsız)',
            'upload.submit': 'Yükle',
            'upload.uploading': 'Yükleniyor...',
            'upload.complete': 'Yükleme tamamlandı',
            'upload.error': 'Yükleme başarısız',
            'upload.networkError': 'Sunucuya ulaşılamıyor',
            'upload.another': 'Daha fazla yükle',
            'shares.title': 'Aktif Paylaşımlar',
            'shares.empty': 'Henüz paylaşım yok',
            'shares.downloads': 'indirme',
            'shares.expired': 'sona erdi',
            'shares.never': 'asla',
            'shares.unlimited': 'sınırsız',
            'shares.expiryRequired': 'Süreyi saat olarak girin veya "sınırsız" seçeneğini açın.',
            'shares.deleteConfirm': 'Bu paylaşımı silmek istiyor musunuz?',
            'shares.copied': 'Bağlantı kopyalandı!',
            'shares.size': 'boyut',
            'shares.deleted': 'Paylaşım silindi',
            'shares.edit': 'Paylaşımı Düzenle',
            'shares.save': 'Kaydet',
            'shares.updated': 'Paylaşım güncellendi',
            'shares.changePassword': 'değiştir veya kaldır',
            'shares.addPassword': 'ekle',
            'receive.title': 'Alma Bağlantıları',
            'receive.create': 'Yeni Bağlantı',
            'receive.formTitle': 'Alma Bağlantısı Oluştur',
            'receive.name': 'Ad',
            'receive.password': 'Şifre (isteğe bağlı)',
            'receive.maxUploads': 'Maks. yükleme (0 = sınırsız)',
            'receive.maxFileSize': 'Maks. dosya boyutu (MB, 0 = sınırsız)',
            'receive.expiry': 'Süre (saat, 0 = süresiz)',
            'receive.extensions': 'İzin verilen uzantılar (ör: .pdf,.doc)',
            'receive.autoShare': 'Alınan dosyaları otomatik paylaş',
            'receive.submit': 'Oluştur',
            'receive.cancel': 'İptal',
            'receive.empty': 'Henüz alma bağlantısı yok',
            'receive.uploads': 'yükleme',
            'receive.deleteConfirm': 'Bu alma bağlantısını silmek istiyor musunuz?',
            'receive.created': 'Alma bağlantısı oluşturuldu',
            'receive.deleted': 'Alma bağlantısı silindi',
            'settings.title': 'Ayarlar',
            'settings.network': 'Ağ Yapılandırması',
            'settings.webhook': 'Webhook',
            'settings.users': 'Kullanıcı Yönetimi',
            'settings.primary': 'Birincil',
            'settings.detected': 'Algılandı',
            'settings.notDetected': 'Algılanmadı',
            'settings.enabled': 'Etkin',
            'settings.useDetected': 'Algılanan değeri kullan',
            'settings.save': 'Kaydet',
            'settings.saved': 'Ayarlar kaydedildi',
            'settings.testWebhook': 'Webhook Test Et',
            'settings.webhookUrl': 'Webhook URL',
            'settings.webhookSecret': 'Webhook Gizli Anahtarı',
            'settings.webhookEnabled': 'Webhook etkin',
            'settings.webhookEvents': 'Şu durumda bildir',
            'settings.webhookOnDownload': 'Dosya indirildi',
            'settings.webhookOnLimit': 'İndirme sınırına ulaşıldı',
            'settings.webhookOnExpire': 'Paylaşım süresi doldu',
            'settings.webhookSecretKeep': 'değişmedi — korumak için boş bırakın',
            'settings.webhookSecretNone': 'gizli anahtar tanımlı değil',
            'settings.webhookSecretClear': 'Kayıtlı gizli anahtarı kaldır',
            'settings.webhookNeedsUrl': 'Bir webhook adresi girin veya webhook\'u kapatın',
            'settings.webhookSent': 'Webhook gönderildi',
            'settings.createUser': 'Kullanıcı Oluştur',
            'settings.email': 'E-posta',
            'settings.name': 'Ad',
            'settings.role': 'Rol',
            'settings.userPassword': 'Şifre',
            'settings.deleteUserConfirm': 'Bu kullanıcıyı silmek istiyor musunuz?',
            'settings.userCreated': 'Kullanıcı oluşturuldu',
            'settings.userDeleted': 'Kullanıcı silindi',
            'settings.fillAll': 'Tüm alanları doldurun',
            'toast.copied': 'Bağlantı panoya kopyalandı',
            'toast.error': 'Bir hata oluştu',
            'stat.totalShares': 'Paylaşımlar',
            'stat.totalDownloads': 'İndirmeler',
            'stat.totalSize': 'Toplam Boyut',
            'stat.protected': 'Korumalı',
            'nav.hostshare': 'Sunucudan paylaş',
            'hostshare.title': 'Sunucudan paylaş',
            'hostshare.hint': 'Sunucuda zaten bulunan dosya ve klasörleri yeniden yüklemeden paylaşın.',
            'hostshare.home': 'Ana dizin',
            'hostshare.empty': 'Bu klasör boş',
            'hostshare.error': 'Bu konuma erişilemedi',
            'hostshare.folder': 'Klasör',
            'hostshare.share': 'Paylaş',
            'hostshare.dialogTitle': 'Bu öğeyi paylaş',
            'hostshare.symlink': 'Kopyalamak yerine bağla (ek disk alanı gerekmez)',
            'hostshare.success': 'Paylaşım oluşturuldu',
            'hostshare.done': 'Bitti',
            'upload.expiry.1h': '1 saat',
            'upload.expiry.6h': '6 saat',
            'upload.expiry.12h': '12 saat',
            'upload.expiry.1d': '1 gün',
            'upload.expiry.3d': '3 gün',
            'upload.expiry.7d': '7 gün',
            'upload.expiry.14d': '14 gün',
            'upload.expiry.30d': '30 gün',
            'upload.expiry.never': 'Asla (sınırsız)',
            'upload.expiry.hours': 'saat',
            'settings.customHint': 'Kendi sabit alan adınızı girin (ör. ters proxy veya adlandırılmış bir Cloudflare tüneli alan adı).',
            'settings.cfRotating': 'Yeni bir Cloudflare paylaşım URL\'si oluşturuluyor…',
            'settings.cfRotated': 'Yeni Cloudflare URL\'si hazır:',
            'settings.cfRotateTimeout': 'Yeni bir Cloudflare URL\'si görünmedi — tünel konteyneri çalışıyor mu?',
            'settings.quota': 'Depolama / Kota',
            'settings.quotaGB': 'Kota (GB, 0 = sınırsız)',
            'settings.unlimited': 'sınırsız',
            'settings.quotaUpdated': 'Kota güncellendi',
            'settings.userPasswordHint': 'İsteğe bağlı — yalnızca SSO kullanan hesaplar için boş bırakın (girilirse en az 8 karakter)',
            'settings.passwordTooShort': 'Şifre en az 8 karakter olmalıdır',
            'settings.maxFileSize': 'Maks. dosya boyutu',
            'settings.fileRestrictions': 'Dosya Kısıtlamaları',
            'settings.fileRestrictionsHint': 'Hangi dosya türlerinin yüklenebileceğini belirleyin. Engellenen uzantılar reddedilir; izin verilen uzantılar bir beyaz liste oluşturur (boş = engellenenler dışında tümüne izin verilir).',
            'settings.blockedExtensions': 'Engellenen uzantılar',
            'settings.allowedExtensions': 'İzin verilen uzantılar (beyaz liste)',
            'settings.allowedExtensionsHint': 'Boş = tümüne izin verilir (engellenenler hariç)',
            'shares.sendEmail': 'E-posta ile Gönder',
            'shares.taildrop': 'Cihaza gönder (Taildrop)',
            'taildrop.title': 'Cihaza gönder',
            'taildrop.device': 'Hedef cihaz',
            'taildrop.send': 'Gönder',
            'taildrop.sending': 'Gönderiliyor…',
            'taildrop.sent': 'Dosya Taildrop ile gönderildi',
            'taildrop.noDevices': 'Kullanılabilir Tailscale cihazı yok',
            'shares.selectAll': 'Tümünü Seç',
            'shares.bulkDelete': 'Seçilenleri Sil',
            'shares.selectedCount': '{n} seçildi',
            'email.recipientEmail': 'Alıcı E-postası',
            'email.recipientName': 'Alıcı Adı',
            'email.message': 'Mesaj',
            'email.optional': 'İsteğe bağlı',
            'email.send': 'Gönder',
            'email.sent': 'E-posta başarıyla gönderildi',
            'email.enterRecipient': 'Alıcının e-posta adresini girin',
            'email.sendFailed': 'E-posta gönderilemedi:',
            'upload.folder': 'Klasör Yükle',
            'settings.apiKeys': 'API Anahtarları',
            'settings.apiKeysHint': 'API anahtarları CasaDrop\'a programlı erişim sağlar. X-API-Key başlığını kullanın.',
            'settings.noApiKeys': 'Henüz API anahtarı yok',
            'settings.createApiKey': 'Anahtar Oluştur',
            'settings.apiKeyCreated': 'API Anahtarı Oluşturuldu — Şimdi kopyalayın!',
            'settings.apiKeyOnlyOnce': 'Bu anahtar yalnızca bir kez gösterilir. Güvenli bir yerde saklayın.',
            'settings.deleteApiKeyConfirm': 'Bu API anahtarını silmek istiyor musunuz?',
            'settings.apiKeyDeleted': 'API anahtarı silindi',
            'settings.smtp': 'E-posta / SMTP',
            'settings.smtpEnabled': 'E-posta gönderimini etkinleştir',
            'settings.testSmtp': 'Bağlantıyı Test Et',
            'settings.smtpTestOk': 'SMTP bağlantısı başarılı',
            'settings.twofa': 'İki Faktörlü Kimlik Doğrulama',
            'settings.twofaHint': 'Yönetici girişinize zamana dayalı tek kullanımlık şifre (TOTP) ekleyin. Aegis, Google Authenticator veya 1Password gibi bir doğrulama uygulaması kullanın.',
            'settings.twofaStatus': 'Durum',
            'settings.twofaEnabled': 'Etkin',
            'settings.twofaDisabled': 'Devre dışı',
            'settings.twofaEnable': '2FA\'yı Etkinleştir',
            'settings.twofaDisable': '2FA\'yı Devre Dışı Bırak',
            'settings.twofaVerifyEnable': 'Doğrula ve Etkinleştir',
            'settings.twofaCancel': 'İptal',
            'settings.twofaScan': 'Bu QR kodunu doğrulama uygulamanızla tarayın:',
            'settings.twofaManual': 'Veya bu gizli anahtarı elle girin:',
            'settings.twofaCode': '6 haneli kodu girin',
            'settings.twofaEnabledMsg': 'İki faktörlü kimlik doğrulama etkinleştirildi',
            'settings.twofaDisabledMsg': 'İki faktörlü kimlik doğrulama devre dışı bırakıldı',
            'settings.twofaCodeRequired': 'Lütfen 6 haneli kodu girin',
            'settings.twofaSetupFailed': '2FA kurulumu başlatılamadı',
            'settings.sessions': 'Aktif Oturumlar',
            'sessions.hint': 'Hesabınızda oturum açmış tüm cihazlar. Tanımadığınız bir oturumu sonlandırın.',
            'sessions.hintAdmin': 'Bu sunucudaki tüm aktif oturumlar. Tanımadığınız bir oturumu sonlandırın.',
            'sessions.current': 'Bu cihaz',
            'sessions.since': 'Oturum açılma',
            'sessions.expires': 'Sona erer',
            'sessions.revoke': 'Oturumu sonlandır',
            'sessions.revokeOthers': 'Diğer tüm cihazlarda oturumu kapat',
            'sessions.revokedOthers': '{n} diğer cihazda oturum kapatıldı',
            'sessions.revoked': 'Oturum sonlandırıldı',
            'sessions.empty': 'Başka aktif oturum yok.',
            'sessions.confirmOthers': 'Diğer tüm cihazlarda oturum kapatılsın mı?',
            'settings.activity': 'Etkinlik günlüğü',
            'settings.activityHint': 'Kimin indirdiği, yüklediği, oturum açtığı — adres ve zamanla birlikte. Saklama süresini aşan kayıtlar otomatik olarak silinir.',
            'activity.when': 'Zaman',
            'activity.kind': 'Olay',
            'activity.who': 'Kim',
            'activity.share': 'Paylaşım',
            'activity.from': 'Kaynak',
            'activity.detail': 'Ayrıntı',
            'activity.empty': 'Henüz kayıt yok.',
            'activity.export': 'CSV Dışa Aktar',
            'activity.allKinds': 'Tüm olaylar',
            'activity.loadMore': 'Daha fazla yükle',
            'activity.anonymous': 'anonim',
            'activity.system': 'sistem',
            'activity.forShare': 'Etkinlik',
            'activity.loadFailed': 'Etkinlik günlüğü yüklenemedi',
            'activity.kind.share.created': 'Paylaşım oluşturuldu',
            'activity.kind.share.updated': 'Paylaşım güncellendi',
            'activity.kind.share.deleted': 'Paylaşım silindi',
            'activity.kind.share.expired': 'Paylaşım süresi doldu',
            'activity.kind.share.downloaded': 'İndirildi',
            'activity.kind.share.streamed': 'Akışla oynatıldı',
            'activity.kind.receive.uploaded': 'Dosya alındı',
            'activity.kind.auth.login': 'Oturum açıldı',
            'activity.kind.auth.login_failed': 'Oturum açma başarısız',
            'activity.kind.auth.locked': 'Hesap kilitlendi',
            'activity.kind.auth.logout': 'Oturum kapatıldı',
            'activity.kind.auth.setup': 'Kurulum / 2FA değişikliği',
            'activity.kind.session.revoked': 'Oturum iptal edildi',
            'activity.kind.security': 'Güvenlik',
            'theme.light': 'Açık mod',
            'theme.dark': 'Koyu mod',
            'load.networkFailed': 'Ağ ayarları yüklenemedi',
            'load.restrictionsFailed': 'Dosya kısıtlamaları yüklenemedi',
            'load.webhookFailed': 'Webhook ayarları yüklenemedi',
            'load.usersUnavailable': 'Kullanıcı yönetimi kullanılamıyor',
            'load.twoFAFailed': '2FA ayarları yüklenemedi',
            'load.apiKeysFailed': 'API anahtarları yüklenemedi',
            'load.smtpFailed': 'SMTP ayarları yüklenemedi',
            'role.admin': 'Yönetici',
            'role.user': 'Kullanıcı',
            'role.viewer': 'Görüntüleyici',
            'settings.apiKeyNamePlaceholder': 'API anahtarım',
            'common.copied': 'Kopyalandı!',
            'smtp.host': 'SMTP sunucusu',
            'smtp.port': 'Port',
            'smtp.encryption': 'Şifreleme',
            'smtp.encryptionNone': 'Yok',
            'smtp.username': 'Kullanıcı adı',
            'smtp.password': 'Parola',
            'smtp.fromEmail': 'Gönderen e-postası',
            'smtp.fromName': 'Gönderen adı',
            'auth.signInAgain': 'Lütfen yeniden oturum açın',
            'shares.expiresWhen': '{when} sona eriyor',
            'activity.d.loginOk': 'Oturum açıldı ({role})',
            'activity.d.loginOkApi': 'API ile oturum açıldı ({role})',
            'activity.d.loginFailed': 'Başarısız oturum açma denemesi {n}/{max}',
            'activity.d.loginFailedApi': 'Başarısız API oturum açma denemesi {n}/{max}',
            'activity.d.loginLocked': 'Çok fazla başarısız denemeden sonra kilitlendi',
            'activity.d.loginLockedApi': 'Çok fazla başarısız API denemesinden sonra kilitlendi',
            'activity.d.twoFAInvalid': '2FA kodu eksik veya geçersiz',
            'activity.d.twoFAInvalidApi': '2FA kodu eksik veya geçersiz (API)',
            'activity.d.localAuthDisabled': 'Parola ile oturum açma devre dışı (yalnızca SSO)',
            'activity.d.csrfInvalid': 'Oturum açarken geçersiz güvenlik belirteci',
            'activity.d.rateLimited': 'Çok fazla oturum açma denemesi',
            'activity.d.loggedOut': 'Oturum kapatıldı',
            'activity.d.setupTokenInvalid': 'Kurulum reddedildi: geçersiz kurulum belirteci',
            'activity.d.setupDone': 'İlk kurulum tamamlandı',
            'activity.d.twoFAEnabled': '2FA etkinleştirildi',
            'activity.d.twoFADisabled': '2FA devre dışı bırakıldı',
            'activity.d.revokedAll': 'Diğer tüm oturumlar sonlandırıldı',
            'activity.d.revokedOne': '{id} oturumu sonlandırıldı ({email})',
            'activity.d.bulk': 'toplu silme',
        },
        // NOTE: Arabic (ar) is an RTL language. Full RTL layout support (CSS direction: rtl) is not yet implemented.
        // The translations below are correct but the UI will display LTR until RTL CSS support is added.
        ar: {
            'common.breadcrumb': 'مسار التنقل',
            'common.copy': 'نسخ',
            'common.copyLink': 'نسخ الرابط',
            'common.folder': 'مجلد',
            'common.expiring': 'تنتهي قريبًا',
            'common.autoShare': 'مشاركة تلقائية',
            'common.qr': 'رمز QR',
            'common.edit': 'تعديل',
            'common.delete': 'حذف',
            'common.remove': 'إزالة',
            'common.close': 'إغلاق',
            'common.refresh': 'تحديث',
            'nav.upload': 'رفع',
            'nav.shares': 'المشاركات',
            'nav.receive': 'استقبال',
            'nav.settings': 'الإعدادات',
            'nav.logout': 'تسجيل الخروج',
            'login.subtitle': 'مشاركة ملفات مستضافة ذاتيًا',
            'login.password': 'كلمة المرور',
            'login.submit': 'تسجيل الدخول',
            'login.or': 'أو',
            'login.sso': 'تسجيل الدخول عبر SSO',
            'upload.title': 'رفع الملفات',
            'upload.dropText': 'اسحب الملفات إلى هنا أو انقر للاستعراض',
            'upload.hint': 'يدعم ملفات متعددة',
            'upload.options': 'خيارات الرفع',
            'upload.password': 'كلمة المرور (اختياري)',
            'upload.expiry': 'تنتهي خلال (ساعات)',
            'upload.maxDownloads': 'أقصى عدد تنزيلات (0 = غير محدود)',
            'upload.submit': 'رفع',
            'upload.uploading': 'جارٍ الرفع...',
            'upload.complete': 'اكتمل الرفع',
            'upload.error': 'فشل الرفع',
            'upload.networkError': 'تعذّر الوصول إلى الخادم',
            'upload.another': 'رفع المزيد',
            'shares.title': 'المشاركات النشطة',
            'shares.empty': 'لا توجد مشاركات بعد',
            'shares.downloads': 'تنزيلات',
            'shares.expired': 'منتهية',
            'shares.never': 'أبدًا',
            'shares.unlimited': 'غير محدود',
            'shares.expiryRequired': 'أدخل مدة الصلاحية بالساعات أو فعّل "غير محدود".',
            'shares.deleteConfirm': 'هل تريد حذف هذه المشاركة؟',
            'shares.copied': 'تم نسخ الرابط!',
            'shares.size': 'الحجم',
            'shares.deleted': 'تم حذف المشاركة',
            'shares.edit': 'تعديل المشاركة',
            'shares.save': 'حفظ',
            'shares.updated': 'تم تحديث المشاركة',
            'shares.changePassword': 'تغيير أو إزالة',
            'shares.addPassword': 'إضافة',
            'receive.title': 'روابط الاستقبال',
            'receive.create': 'رابط جديد',
            'receive.formTitle': 'إنشاء رابط استقبال',
            'receive.name': 'الاسم',
            'receive.password': 'كلمة المرور (اختياري)',
            'receive.maxUploads': 'أقصى عدد رفع (0 = غير محدود)',
            'receive.maxFileSize': 'أقصى حجم ملف (ميغابايت، 0 = غير محدود)',
            'receive.expiry': 'تنتهي خلال (ساعات، 0 = أبدًا)',
            'receive.extensions': 'الامتدادات المسموحة (مثل: .pdf,.doc)',
            'receive.autoShare': 'مشاركة الملفات المستلمة تلقائيًا',
            'receive.submit': 'إنشاء',
            'receive.cancel': 'إلغاء',
            'receive.empty': 'لا توجد روابط استقبال بعد',
            'receive.uploads': 'رفع',
            'receive.deleteConfirm': 'هل تريد حذف رابط الاستقبال هذا؟',
            'receive.created': 'تم إنشاء رابط الاستقبال',
            'receive.deleted': 'تم حذف رابط الاستقبال',
            'settings.title': 'الإعدادات',
            'settings.network': 'إعدادات الشبكة',
            'settings.webhook': 'Webhook',
            'settings.users': 'إدارة المستخدمين',
            'settings.primary': 'أساسي',
            'settings.detected': 'تم الكشف',
            'settings.notDetected': 'لم يتم الكشف',
            'settings.enabled': 'مفعّل',
            'settings.useDetected': 'استخدم القيمة المكتشفة',
            'settings.save': 'حفظ',
            'settings.saved': 'تم حفظ الإعدادات',
            'settings.testWebhook': 'اختبار Webhook',
            'settings.webhookUrl': 'رابط Webhook',
            'settings.webhookSecret': 'مفتاح Webhook السري',
            'settings.webhookEnabled': 'تفعيل الويب هوك',
            'settings.webhookEvents': 'إشعار عند',
            'settings.webhookOnDownload': 'تنزيل الملف',
            'settings.webhookOnLimit': 'بلوغ حد التنزيلات',
            'settings.webhookOnExpire': 'انتهاء صلاحية المشاركة',
            'settings.webhookSecretKeep': 'دون تغيير — اتركه فارغًا للإبقاء عليه',
            'settings.webhookSecretNone': 'لا يوجد مفتاح سري',
            'settings.webhookSecretClear': 'إزالة المفتاح السري المحفوظ',
            'settings.webhookNeedsUrl': 'أدخل عنوان الويب هوك أو عطّل الويب هوك',
            'settings.webhookSent': 'تم إرسال Webhook',
            'settings.createUser': 'إنشاء مستخدم',
            'settings.email': 'البريد الإلكتروني',
            'settings.name': 'الاسم',
            'settings.role': 'الدور',
            'settings.userPassword': 'كلمة المرور',
            'settings.deleteUserConfirm': 'هل تريد حذف هذا المستخدم؟',
            'settings.userCreated': 'تم إنشاء المستخدم',
            'settings.userDeleted': 'تم حذف المستخدم',
            'settings.fillAll': 'يرجى ملء جميع الحقول',
            'toast.copied': 'تم نسخ الرابط إلى الحافظة',
            'toast.error': 'حدث خطأ ما',
            'stat.totalShares': 'المشاركات',
            'stat.totalDownloads': 'التنزيلات',
            'stat.totalSize': 'الحجم الإجمالي',
            'stat.protected': 'محمية',
            'nav.hostshare': 'المشاركة من الخادم',
            'hostshare.title': 'المشاركة من الخادم',
            'hostshare.hint': 'شارك الملفات والمجلدات الموجودة بالفعل على الخادم — دون إعادة رفعها.',
            'hostshare.home': 'الرئيسية',
            'hostshare.empty': 'هذا المجلد فارغ',
            'hostshare.error': 'تعذّر الوصول إلى هذا الموقع',
            'hostshare.folder': 'مجلد',
            'hostshare.share': 'مشاركة',
            'hostshare.dialogTitle': 'مشاركة هذا العنصر',
            'hostshare.symlink': 'ربط بدلًا من النسخ (دون مساحة قرص إضافية)',
            'hostshare.success': 'تم إنشاء المشاركة',
            'hostshare.done': 'تم',
            'upload.expiry.1h': 'ساعة واحدة',
            'upload.expiry.6h': '6 ساعات',
            'upload.expiry.12h': '12 ساعة',
            'upload.expiry.1d': 'يوم واحد',
            'upload.expiry.3d': '3 أيام',
            'upload.expiry.7d': '7 أيام',
            'upload.expiry.14d': '14 يومًا',
            'upload.expiry.30d': '30 يومًا',
            'upload.expiry.never': 'أبدًا (غير محدود)',
            'upload.expiry.hours': 'ساعات',
            'settings.customHint': 'أدخل نطاقك الثابت الخاص (مثل وكيل عكسي أو نطاق نفق Cloudflare مُسمّى).',
            'settings.cfRotating': 'جارٍ إنشاء رابط مشاركة Cloudflare جديد…',
            'settings.cfRotated': 'رابط Cloudflare الجديد جاهز:',
            'settings.cfRotateTimeout': 'لم يظهر رابط Cloudflare جديد — هل حاوية النفق قيد التشغيل؟',
            'settings.quota': 'التخزين / الحصة',
            'settings.quotaGB': 'الحصة (غيغابايت، 0 = غير محدود)',
            'settings.unlimited': 'غير محدود',
            'settings.quotaUpdated': 'تم تحديث الحصة',
            'settings.userPasswordHint': 'اختياري — اتركه فارغًا للحسابات التي تستخدم SSO فقط (8 أحرف على الأقل إن تم تعيينه)',
            'settings.passwordTooShort': 'يجب أن تتكون كلمة المرور من 8 أحرف على الأقل',
            'settings.maxFileSize': 'أقصى حجم للملف',
            'settings.fileRestrictions': 'قيود الملفات',
            'settings.fileRestrictionsHint': 'تحكّم في أنواع الملفات التي يمكن رفعها. تُرفض الامتدادات المحظورة، وتُنشئ الامتدادات المسموحة قائمة سماح (فارغ = السماح بالكل باستثناء المحظورة).',
            'settings.blockedExtensions': 'الامتدادات المحظورة',
            'settings.allowedExtensions': 'الامتدادات المسموحة (قائمة السماح)',
            'settings.allowedExtensionsHint': 'فارغ = السماح بالكل (باستثناء المحظورة)',
            'shares.sendEmail': 'إرسال عبر البريد الإلكتروني',
            'shares.taildrop': 'إرسال إلى جهاز (Taildrop)',
            'taildrop.title': 'إرسال إلى جهاز',
            'taildrop.device': 'الجهاز الهدف',
            'taildrop.send': 'إرسال',
            'taildrop.sending': 'جارٍ الإرسال…',
            'taildrop.sent': 'تم إرسال الملف عبر Taildrop',
            'taildrop.noDevices': 'لا توجد أجهزة Tailscale متاحة',
            'shares.selectAll': 'تحديد الكل',
            'shares.bulkDelete': 'حذف المحدد',
            'shares.selectedCount': 'المحدد: {n}',
            'email.recipientEmail': 'البريد الإلكتروني للمستلم',
            'email.recipientName': 'اسم المستلم',
            'email.message': 'الرسالة',
            'email.optional': 'اختياري',
            'email.send': 'إرسال',
            'email.sent': 'تم إرسال البريد الإلكتروني بنجاح',
            'email.enterRecipient': 'أدخل البريد الإلكتروني للمستلم',
            'email.sendFailed': 'تعذّر إرسال البريد الإلكتروني:',
            'upload.folder': 'رفع مجلد',
            'settings.apiKeys': 'مفاتيح API',
            'settings.apiKeysHint': 'تتيح مفاتيح API الوصول البرمجي إلى CasaDrop. استخدم ترويسة X-API-Key.',
            'settings.noApiKeys': 'لا توجد مفاتيح API بعد',
            'settings.createApiKey': 'إنشاء مفتاح',
            'settings.apiKeyCreated': 'تم إنشاء مفتاح API — انسخه الآن!',
            'settings.apiKeyOnlyOnce': 'سيُعرض هذا المفتاح مرة واحدة فقط. احفظه في مكان آمن.',
            'settings.deleteApiKeyConfirm': 'هل تريد حذف مفتاح API هذا؟',
            'settings.apiKeyDeleted': 'تم حذف مفتاح API',
            'settings.smtp': 'البريد الإلكتروني / SMTP',
            'settings.smtpEnabled': 'تفعيل إرسال البريد الإلكتروني',
            'settings.testSmtp': 'اختبار الاتصال',
            'settings.smtpTestOk': 'تم الاتصال بـ SMTP بنجاح',
            'settings.twofa': 'المصادقة الثنائية',
            'settings.twofaHint': 'أضف كلمة مرور لمرة واحدة مستندة إلى الوقت (TOTP) إلى تسجيل دخول المسؤول. استخدم تطبيق مصادقة مثل Aegis أو Google Authenticator أو 1Password.',
            'settings.twofaStatus': 'الحالة',
            'settings.twofaEnabled': 'مفعّلة',
            'settings.twofaDisabled': 'معطّلة',
            'settings.twofaEnable': 'تفعيل 2FA',
            'settings.twofaDisable': 'تعطيل 2FA',
            'settings.twofaVerifyEnable': 'تحقق وفعّل',
            'settings.twofaCancel': 'إلغاء',
            'settings.twofaScan': 'امسح رمز QR هذا بتطبيق المصادقة:',
            'settings.twofaManual': 'أو أدخل هذا المفتاح السري يدويًا:',
            'settings.twofaCode': 'أدخل الرمز المكوّن من 6 أرقام',
            'settings.twofaEnabledMsg': 'تم تفعيل المصادقة الثنائية',
            'settings.twofaDisabledMsg': 'تم تعطيل المصادقة الثنائية',
            'settings.twofaCodeRequired': 'يرجى إدخال الرمز المكوّن من 6 أرقام',
            'settings.twofaSetupFailed': 'تعذّر بدء إعداد 2FA',
            'settings.sessions': 'الجلسات النشطة',
            'sessions.hint': 'جميع الأجهزة المسجّل دخولها إلى حسابك. أنهِ أي جلسة لا تتعرف عليها.',
            'sessions.hintAdmin': 'جميع الجلسات النشطة على هذا الخادم. أنهِ أي جلسة لا تتعرف عليها.',
            'sessions.current': 'هذا الجهاز',
            'sessions.since': 'تسجيل الدخول',
            'sessions.expires': 'تنتهي',
            'sessions.revoke': 'إنهاء الجلسة',
            'sessions.revokeOthers': 'تسجيل الخروج من جميع الأجهزة الأخرى',
            'sessions.revokedOthers': 'تم تسجيل الخروج من الأجهزة الأخرى (العدد: {n})',
            'sessions.revoked': 'تم إنهاء الجلسة',
            'sessions.empty': 'لا توجد جلسات نشطة أخرى.',
            'sessions.confirmOthers': 'هل تريد تسجيل الخروج من جميع الأجهزة الأخرى؟',
            'settings.activity': 'سجل النشاط',
            'settings.activityHint': 'من قام بالتنزيل أو الرفع أو تسجيل الدخول — مع العنوان والوقت. تُحذف الإدخالات الأقدم من مدة الاحتفاظ تلقائيًا.',
            'activity.when': 'الوقت',
            'activity.kind': 'الحدث',
            'activity.who': 'المستخدم',
            'activity.share': 'المشاركة',
            'activity.from': 'المصدر',
            'activity.detail': 'التفاصيل',
            'activity.empty': 'لم يُسجَّل شيء بعد.',
            'activity.export': 'تصدير CSV',
            'activity.allKinds': 'جميع الأحداث',
            'activity.loadMore': 'تحميل المزيد',
            'activity.anonymous': 'مجهول',
            'activity.system': 'النظام',
            'activity.forShare': 'النشاط',
            'activity.loadFailed': 'تعذّر تحميل سجل النشاط',
            'activity.kind.share.created': 'تم إنشاء المشاركة',
            'activity.kind.share.updated': 'تم تحديث المشاركة',
            'activity.kind.share.deleted': 'تم حذف المشاركة',
            'activity.kind.share.expired': 'انتهت صلاحية المشاركة',
            'activity.kind.share.downloaded': 'تم التنزيل',
            'activity.kind.share.streamed': 'تم البث',
            'activity.kind.receive.uploaded': 'تم استلام ملف',
            'activity.kind.auth.login': 'تسجيل الدخول',
            'activity.kind.auth.login_failed': 'فشل تسجيل الدخول',
            'activity.kind.auth.locked': 'تم قفل الحساب',
            'activity.kind.auth.logout': 'تسجيل الخروج',
            'activity.kind.auth.setup': 'الإعداد / تغيير 2FA',
            'activity.kind.session.revoked': 'تم إلغاء الجلسة',
            'activity.kind.security': 'الأمان',
            'theme.light': 'الوضع الفاتح',
            'theme.dark': 'الوضع الداكن',
            'load.networkFailed': 'تعذّر تحميل إعدادات الشبكة',
            'load.restrictionsFailed': 'تعذّر تحميل قيود الملفات',
            'load.webhookFailed': 'تعذّر تحميل إعدادات Webhook',
            'load.usersUnavailable': 'إدارة المستخدمين غير متاحة',
            'load.twoFAFailed': 'تعذّر تحميل إعدادات 2FA',
            'load.apiKeysFailed': 'تعذّر تحميل مفاتيح API',
            'load.smtpFailed': 'تعذّر تحميل إعدادات SMTP',
            'role.admin': 'مسؤول',
            'role.user': 'مستخدم',
            'role.viewer': 'مشاهد',
            'settings.apiKeyNamePlaceholder': 'مفتاح API الخاص بي',
            'common.copied': 'تم النسخ!',
            'smtp.host': 'خادم SMTP',
            'smtp.port': 'المنفذ',
            'smtp.encryption': 'التشفير',
            'smtp.encryptionNone': 'بدون',
            'smtp.username': 'اسم المستخدم',
            'smtp.password': 'كلمة المرور',
            'smtp.fromEmail': 'بريد المرسل',
            'smtp.fromName': 'اسم المرسل',
            'auth.signInAgain': 'يرجى تسجيل الدخول مرة أخرى',
            'shares.expiresWhen': 'تنتهي {when}',
            'activity.d.loginOk': 'تم تسجيل الدخول ({role})',
            'activity.d.loginOkApi': 'تم تسجيل الدخول عبر API ({role})',
            'activity.d.loginFailed': 'محاولة تسجيل دخول فاشلة {n}/{max}',
            'activity.d.loginFailedApi': 'محاولة تسجيل دخول فاشلة عبر API {n}/{max}',
            'activity.d.loginLocked': 'تم القفل بعد محاولات فاشلة كثيرة',
            'activity.d.loginLockedApi': 'تم القفل بعد محاولات API فاشلة كثيرة',
            'activity.d.twoFAInvalid': 'رمز 2FA مفقود أو غير صالح',
            'activity.d.twoFAInvalidApi': 'رمز 2FA مفقود أو غير صالح (API)',
            'activity.d.localAuthDisabled': 'تسجيل الدخول بكلمة المرور معطّل (SSO فقط)',
            'activity.d.csrfInvalid': 'رمز أمان غير صالح عند تسجيل الدخول',
            'activity.d.rateLimited': 'محاولات تسجيل دخول كثيرة جدًا',
            'activity.d.loggedOut': 'تم تسجيل الخروج',
            'activity.d.setupTokenInvalid': 'تم رفض الإعداد: رمز الإعداد غير صالح',
            'activity.d.setupDone': 'اكتمل الإعداد الأولي',
            'activity.d.twoFAEnabled': 'تم تفعيل 2FA',
            'activity.d.twoFADisabled': 'تم تعطيل 2FA',
            'activity.d.revokedAll': 'تم إنهاء جميع الجلسات الأخرى',
            'activity.d.revokedOne': 'تم إنهاء الجلسة {id} ({email})',
            'activity.d.bulk': 'حذف جماعي',
        }
    };

    function t(key) {
        return I18N[LANG]?.[key] || I18N['en']?.[key] || key;
    }

    // Role names come from the server as admin/user/viewer; show them translated.
    function roleLabel(role) {
        const key = 'role.' + String(role || '').toLowerCase();
        return I18N.en[key] ? t(key) : role;
    }

    function applyI18n() {
        document.querySelectorAll('[data-i18n]').forEach(el => {
            const key = el.getAttribute('data-i18n');
            const translated = t(key);
            if (el.tagName === 'INPUT' && el.type !== 'submit') {
                el.placeholder = translated;
            } else {
                el.textContent = translated;
            }
        });
        document.querySelectorAll('[data-i18n-title]').forEach(el => {
            el.title = t(el.getAttribute('data-i18n-title'));
        });
        document.querySelectorAll('[data-i18n-aria]').forEach(el => {
            el.setAttribute('aria-label', t(el.getAttribute('data-i18n-aria')));
        });
    }

    // ==========================================
    // Utilities
    // ==========================================
    function formatSize(bytes) {
        if (bytes === 0) return '0 B';
        const units = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(1024));
        return (bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1) + ' ' + units[i];
    }

    // relTime formats a point in time relative to now in the UI language
    // ("in 5 hr." / "vor 3 Min." / "5時間後") with the browser's own
    // Intl.RelativeTimeFormat — it covers every UI language, a hand-written
    // table covered de and en only.
    function relTime(dateStr) {
        const diffMs = new Date(dateStr) - new Date();
        const rtf = new Intl.RelativeTimeFormat(LANG, { numeric: 'auto', style: 'short' });
        const abs = Math.abs(diffMs);
        if (abs < 60000) return rtf.format(0, 'minute');
        if (abs < 3600000) return rtf.format(Math.round(diffMs / 60000), 'minute');
        if (abs < 86400000) return rtf.format(Math.round(diffMs / 3600000), 'hour');
        return rtf.format(Math.round(diffMs / 86400000), 'day');
    }

    function relativeExpiry(dateStr) {
        if (!dateStr) return t('shares.never');
        const date = new Date(dateStr);
        // Sentinel: backend stores 9999-12-31 for "unbegrenzt" shares because
        // the ExpiresAt column is NOT NULL. Any date past year 9000 is never.
        if (date.getFullYear() > 9000) return t('shares.never');
        const now = new Date();
        if (date <= now) return t('shares.expired');
        return t('shares.expiresWhen').replace('{when}', relTime(dateStr));
    }

    // shareThumbHTML returns the inner HTML for a share-card's thumbnail slot.
    // For image/* shares we hit /thumbnail/{id}; on load-error, a delegated
    // handler attached after innerHTML (see loadShares) swaps the <img> for
    // the type icon. Password-protected images skip the live fetch because
    // the endpoint requires ?password=.
    //
    // No inline onerror: JSON.stringify(mime) produces literal double quotes
    // which break an onerror="..." attribute; the delegated listener is both
    // safer and trivially easier to reason about.
    function shareThumbHTML(share) {
        const mime = (share.mime_type || '').toLowerCase();
        const id = escapeHtml(share.id);
        if (share.is_directory) return iconFolder();
        if (mime.startsWith('image/') && !share.has_password) {
            return `<img src="/thumbnail/${id}" alt="" loading="lazy" data-thumb-fallback="${escapeHtml(mime)}">`;
        }
        return shareIconForMime(mime);
    }
    function iconFolder() {
        return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>';
    }
    function shareIconForMime(mime) {
        if (mime.startsWith('image/')) return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>';
        if (mime.startsWith('video/')) return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg>';
        if (mime.startsWith('audio/')) return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>';
        if (mime === 'application/pdf') return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>';
        if (mime === 'application/zip' || mime === 'application/gzip' || mime === 'application/x-tar' || mime === 'application/x-7z-compressed') return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>';
        return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>';
    }

    // attachThumbFallbacks wires up the onerror fallback for every
    // <img data-thumb-fallback="..."> in a freshly-rendered share list.
    // Called after loadShares() sets innerHTML.
    function attachThumbFallbacks(root) {
        root.querySelectorAll('img[data-thumb-fallback]').forEach(img => {
            img.addEventListener('error', () => {
                const mime = img.getAttribute('data-thumb-fallback') || '';
                const parent = img.parentNode;
                if (parent) parent.innerHTML = shareIconForMime(mime);
            }, { once: true });
        });
    }

    function fileTypeInfo(name) {
        const ext = (name.split('.').pop() || '').toLowerCase();
        const map = {
            jpg: ['IMG', 'file-icon-image'], jpeg: ['IMG', 'file-icon-image'], png: ['IMG', 'file-icon-image'],
            gif: ['GIF', 'file-icon-image'], webp: ['IMG', 'file-icon-image'], svg: ['SVG', 'file-icon-image'],
            mp4: ['MP4', 'file-icon-video'], mkv: ['MKV', 'file-icon-video'], avi: ['AVI', 'file-icon-video'],
            mov: ['MOV', 'file-icon-video'], webm: ['VID', 'file-icon-video'],
            mp3: ['MP3', 'file-icon-audio'], wav: ['WAV', 'file-icon-audio'], flac: ['FLC', 'file-icon-audio'],
            ogg: ['OGG', 'file-icon-audio'],
            pdf: ['PDF', 'file-icon-pdf'],
            doc: ['DOC', 'file-icon-doc'], docx: ['DOC', 'file-icon-doc'], txt: ['TXT', 'file-icon-doc'],
            md: ['MD', 'file-icon-doc'], rtf: ['RTF', 'file-icon-doc'],
            zip: ['ZIP', 'file-icon-zip'], tar: ['TAR', 'file-icon-zip'], gz: ['GZ', 'file-icon-zip'],
            '7z': ['7Z', 'file-icon-zip'], rar: ['RAR', 'file-icon-zip'],
            js: ['JS', 'file-icon-code'], ts: ['TS', 'file-icon-code'], py: ['PY', 'file-icon-code'],
            go: ['GO', 'file-icon-code'], rs: ['RS', 'file-icon-code'], html: ['HTM', 'file-icon-code'],
            css: ['CSS', 'file-icon-code'], json: ['JSN', 'file-icon-code'], xml: ['XML', 'file-icon-code'],
            yaml: ['YML', 'file-icon-code'], yml: ['YML', 'file-icon-code'],
        };
        const info = map[ext];
        if (info) return { label: info[0], cls: info[1] };
        return { label: ext ? ext.slice(0, 3).toUpperCase() : 'FIL', cls: 'file-icon-default' };
    }

    // Escapes for BOTH element and attribute contexts. The browser's
    // textContent→innerHTML trick does not escape quotes, so values placed into
    // double-quoted attributes (title=, data-*=, value=) could break out and
    // inject markup — e.g. an uploaded filename like  x" onmouseover="…  reaching
    // the admin's shares list. Escaping quotes here is visually identical in text
    // context and closes the attribute-injection across every interpolation site.
    function escapeHtml(str) {
        if (!str) return '';
        return String(str)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#39;');
    }

    async function api(path, opts = {}) {
        const res = await fetch(path, {
            credentials: 'same-origin',
            ...opts,
            headers: {
                ...(opts.body && typeof opts.body === 'string' ? { 'Content-Type': 'application/json' } : {}),
                ...opts.headers,
            },
        });
        // Only a 401 means the session is actually gone — that's the one case
        // where dropping the user to the login screen is right. A 403 is
        // "authenticated but not allowed" (e.g. a non-admin hitting an admin
        // endpoint, or a host path outside SHARE_ALLOWED_PATHS). Re-logging-in
        // can't fix a 403, so surfacing the login modal there is misleading —
        // let the caller handle it as a normal error (inline message + toast).
        if (res.status === 401) {
            showLogin();
            throw new Error(t('auth.signInAgain'));
        }
        return res;
    }

    // ==========================================
    // Toast Notifications
    // ==========================================
    function toast(message, type = 'info') {
        const container = document.getElementById('toast-container');
        const el = document.createElement('div');
        el.className = `toast toast-${type}`;

        const iconMap = {
            success: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 11-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>',
            error: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>',
            info: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>',
            warning: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>',
        };

        el.innerHTML = `
            <span class="toast-icon">${iconMap[type] || iconMap.info}</span>
            <span class="toast-message">${escapeHtml(message)}</span>
            <button class="toast-close" aria-label="${escapeHtml(t('common.close'))}">&times;</button>
        `;

        el.querySelector('.toast-close').addEventListener('click', () => {
            el.classList.add('removing');
            setTimeout(() => el.remove(), 300);
        });

        container.appendChild(el);
        setTimeout(() => {
            el.classList.add('removing');
            setTimeout(() => el.remove(), 300);
        }, 4000);
    }

    // ==========================================
    // Modal
    // ==========================================
    function showModal(html) {
        const overlay = document.getElementById('modal-overlay');
        const content = document.getElementById('modal-content');
        content.innerHTML = html;
        overlay.style.display = 'flex';
        overlay.onclick = (e) => { if (e.target === overlay) closeModal(); };
    }

    function closeModal() {
        document.getElementById('modal-overlay').style.display = 'none';
    }

    // ==========================================
    // Clipboard
    // ==========================================
    async function copyToClipboard(text) {
        try {
            await navigator.clipboard.writeText(text);
            toast(t('toast.copied'), 'success');
        } catch {
            const ta = document.createElement('textarea');
            ta.value = text;
            ta.style.cssText = 'position:fixed;opacity:0';
            document.body.appendChild(ta);
            ta.select();
            document.execCommand('copy');
            ta.remove();
            toast(t('toast.copied'), 'success');
        }
    }

    // ==========================================
    // State
    // ==========================================
    let currentView = 'upload';
    let selectedFiles = [];
    let currentUser = null;
    let currentShares = [];
    let taildropState = { available: false, devices: [] };
    let networkConfig = null;
    let currentHostPath = '/';
    // Allowed host-share roots (from SHARE_ALLOWED_PATHS), learned from the
    // root ("/") browse listing. Used to keep breadcrumb ancestors that sit
    // ABOVE every allowed root (e.g. "/DATA" when only "/DATA/Media" is shared)
    // non-clickable, since browsing them would 403.
    let hostRoots = [];

    // ==========================================
    // Navigation
    // ==========================================
    function showView(name) {
        currentView = name;
        document.querySelectorAll('.view').forEach(v => v.style.display = 'none');
        const view = document.getElementById(`${name}-view`);
        if (view) view.style.display = '';

        document.querySelectorAll('.nav-btn[data-view]').forEach(b => {
            b.classList.toggle('active', b.dataset.view === name);
        });

        if (name === 'shares') loadShares();
        if (name === 'receive') loadReceiveLinks();
        if (name === 'settings') loadSettings();
        if (name === 'hostshare') loadHostBrowser(currentHostPath);

        // Close mobile sidebar
        document.getElementById('sidebar')?.classList.remove('open');
        document.querySelectorAll('.mobile-overlay').forEach(e => e.remove());
    }

    function showLogin() {
        document.querySelectorAll('.view').forEach(v => v.style.display = 'none');
        document.getElementById('login-view').style.display = 'flex';
        document.getElementById('sidebar').style.display = 'none';
        const mh = document.querySelector('.mobile-header');
        if (mh) mh.style.display = 'none';

        // Check SSO availability
        checkSSOAvailable();
    }

    function showApp() {
        document.getElementById('sidebar').style.display = '';
        const mh = document.querySelector('.mobile-header');
        if (mh) mh.style.display = '';
        showView(currentView);
    }

    // ==========================================
    // Auth
    // ==========================================
    async function checkAuth() {
        try {
            const res = await fetch('/api/auth/status', { credentials: 'same-origin' });
            const data = await res.json();

            if (data.setupRequired) {
                window.location.href = '/setup';
                return;
            }

            if (data.authenticated) {
                loadCurrentUser();
                showApp();
            } else {
                showLogin();
            }
        } catch {
            showLogin();
        }
    }

    async function checkSSOAvailable() {
        try {
            const res = await fetch('/api/auth/oidc/status', { credentials: 'same-origin' });
            if (res.ok) {
                const data = await res.json();
                const ssoSection = document.getElementById('sso-section');
                if (data.enabled && ssoSection) {
                    ssoSection.style.display = '';
                }
            }
        } catch { /* SSO not available */ }
    }

    async function doLogin(password) {
        const res = await fetch('/login', {
            method: 'POST',
            credentials: 'same-origin',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ password }),
        });

        if (res.ok) {
            const data = await res.json();
            if (data.success) {
                loadCurrentUser();
                showApp();
                return;
            }
        }

        let errMsg = 'Invalid password';
        try {
            const data = await res.json();
            if (data?.error) errMsg = data.error;
        } catch { /* use default */ }
        throw new Error(errMsg);
    }

    async function doLogout() {
        try {
            await fetch('/logout', {
                credentials: 'same-origin',
                headers: { 'Accept': 'application/json' },
            });
        } catch { /* ignore */ }
        currentUser = null;
        showLogin();
    }

    async function loadCurrentUser() {
        try {
            const res = await api('/api/me');
            if (res.ok) {
                currentUser = await res.json();
                renderUserInfo();
            }
        } catch { /* might be single-user mode */ }
        // Always render sidebar controls (language, theme) regardless of /api/me
        renderSidebarControls();
        applyRoleVisibility();
    }

    // Reveal admin-only UI. The host-share endpoints (/api/browse,
    // /api/share-from-path, /api/share-folder) are admin-gated server-side;
    // hiding the entry for non-admins just avoids dead 403 buttons. Default
    // role is 'admin' (single-user mode), so it shows unless we know otherwise.
    function applyRoleVisibility() {
        const isAdmin = (currentUser?.role || 'admin') === 'admin';
        const btn = document.getElementById('nav-hostshare');
        if (btn) btn.style.display = isAdmin ? '' : 'none';
    }

    function renderUserInfo() {
        const el = document.getElementById('user-info');
        if (!el) return;

        // The shared admin account (ADMIN_PASSWORD / setup wizard) has no user
        // record; /api/me then sends the fixed placeholder name "Admin"
        // (handlers/users.go). It is a label, not a chosen name — translate it.
        const name = (!currentUser?.name || currentUser.name === 'Admin') ? t('role.admin') : currentUser.name;
        const role = currentUser?.role || 'admin';
        const initial = name.charAt(0).toUpperCase();

        el.innerHTML = `
            <div class="user-avatar">${escapeHtml(initial)}</div>
            <div>
                <div class="user-name">${escapeHtml(name)}</div>
                <div class="user-role">${escapeHtml(roleLabel(role))}</div>
            </div>
        `;
    }

    function renderSidebarControls() {
        const footer = document.querySelector('.sidebar-footer');
        if (!footer || document.getElementById('sidebar-controls')) return;

        const controls = document.createElement('div');
        controls.id = 'sidebar-controls';
        controls.style.cssText = 'padding:8px 12px;display:flex;flex-direction:column;gap:8px;';

        // Language switcher
        const langNames = {en:'English',de:'Deutsch',fr:'Français',es:'Español',it:'Italiano',pt:'Português',nl:'Nederlands',pl:'Polski',ru:'Русский',ja:'日本語',zh:'中文',ko:'한국어',tr:'Türkçe',ar:'العربية'};
        const langSelect = document.createElement('select');
        langSelect.id = 'lang-select';
        langSelect.style.cssText = 'width:100%;padding:6px 10px;border:1px solid var(--border);border-radius:8px;background:var(--bg-card);color:var(--text-secondary);font-size:0.8rem;cursor:pointer;outline:none;';
        SUPPORTED_LANGS.forEach(code => {
            const opt = document.createElement('option');
            opt.value = code;
            opt.textContent = langNames[code] || code;
            if (code === LANG) opt.selected = true;
            langSelect.appendChild(opt);
        });
        langSelect.onchange = () => {
            localStorage.setItem('casadrop_lang', langSelect.value);
            location.reload();
        };
        controls.appendChild(langSelect);

        // Dark/Light mode toggle
        const themeBtn = document.createElement('button');
        themeBtn.id = 'theme-toggle';
        themeBtn.className = 'nav-btn';
        themeBtn.style.cssText = 'padding:8px 12px;';
        const isDark = localStorage.getItem('casadrop_theme') === 'dark';
        themeBtn.innerHTML = isDark
            ? '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg> <span>' + escapeHtml(t('theme.light')) + '</span>'
            : '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg> <span>' + escapeHtml(t('theme.dark')) + '</span>';
        themeBtn.onclick = () => {
            const current = document.documentElement.getAttribute('data-theme');
            const next = current === 'dark' ? 'light' : 'dark';
            document.documentElement.setAttribute('data-theme', next);
            localStorage.setItem('casadrop_theme', next);
            location.reload();
        };
        controls.appendChild(themeBtn);

        // Insert before logout button
        const logoutBtn = document.getElementById('logout-btn');
        if (logoutBtn) {
            footer.insertBefore(controls, logoutBtn);
        } else {
            footer.appendChild(controls);
        }
    }

    // ==========================================
    // Upload
    // ==========================================
    function initUpload() {
        const dropZone = document.getElementById('drop-zone');
        const fileInput = document.getElementById('file-input');

        dropZone.addEventListener('click', () => fileInput.click());

        dropZone.addEventListener('dragover', (e) => {
            e.preventDefault();
            dropZone.classList.add('drag-over');
        });

        dropZone.addEventListener('dragleave', (e) => {
            // Only remove if leaving the drop zone entirely
            if (!dropZone.contains(e.relatedTarget)) {
                dropZone.classList.remove('drag-over');
            }
        });

        dropZone.addEventListener('drop', (e) => {
            e.preventDefault();
            dropZone.classList.remove('drag-over');
            const files = Array.from(e.dataTransfer.files);
            if (files.length) addFiles(files);
        });

        fileInput.addEventListener('change', () => {
            const files = Array.from(fileInput.files);
            if (files.length) addFiles(files);
            fileInput.value = '';
        });

        // Folder upload
        const folderInput = document.createElement('input');
        folderInput.type = 'file';
        folderInput.webkitdirectory = true;
        folderInput.directory = true;
        folderInput.multiple = true;
        folderInput.style.display = 'none';
        document.body.appendChild(folderInput);

        folderInput.onchange = () => {
            if (folderInput.files.length) {
                addFiles(Array.from(folderInput.files));
            }
            folderInput.value = '';
        };

        const folderBtn = document.createElement('button');
        folderBtn.className = 'btn btn-ghost btn-sm';
        folderBtn.style.marginTop = '12px';
        folderBtn.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg> ${t('upload.folder')}`;
        folderBtn.onclick = (e) => { e.stopPropagation(); folderInput.click(); };
        dropZone.appendChild(folderBtn);

        document.getElementById('upload-btn').addEventListener('click', startUpload);
    }

    function addFiles(files) {
        selectedFiles.push(...files);
        renderFileList();
        document.getElementById('upload-options').style.display = '';
        document.getElementById('upload-results').style.display = 'none';
        document.getElementById('upload-progress').style.display = 'none';
    }

    function removeFile(index) {
        selectedFiles.splice(index, 1);
        renderFileList();
        if (selectedFiles.length === 0) {
            document.getElementById('upload-options').style.display = 'none';
        }
    }

    // Object URLs handed to the preview <img> elements. They pin the blob until
    // revoked, so every re-render releases the previous batch — otherwise
    // picking and unpicking a folder of photos a few times leaks all of them.
    let previewUrls = [];

    function releasePreviewUrls() {
        previewUrls.forEach(URL.revokeObjectURL);
        previewUrls = [];
    }

    // Only real raster images get a thumbnail. SVG is deliberately excluded:
    // rendering an untrusted SVG in the admin page is an XSS surface, and the
    // type badge is the honest fallback.
    function isPreviewableImage(file) {
        return /^image\/(png|jpeg|gif|webp|bmp|avif)$/.test(file.type);
    }

    function renderFileList() {
        const container = document.getElementById('file-list');
        releasePreviewUrls();
        container.innerHTML = selectedFiles.map((f, i) => {
            const info = fileTypeInfo(f.name);
            let visual;
            if (isPreviewableImage(f)) {
                const url = URL.createObjectURL(f);
                previewUrls.push(url);
                // data-fallback carries the badge markup for the error handler
                // below; no inline handler, the CSP forbids it.
                visual = `<img class="file-item-thumb" src="${url}" alt="" loading="lazy"
                               data-fallback-cls="${info.cls}" data-fallback-label="${escapeHtml(info.label)}">`;
            } else {
                visual = `<div class="file-item-icon ${info.cls}">${info.label}</div>`;
            }
            return `
            <div class="file-item">
                <div class="file-item-info">
                    ${visual}
                    <span class="file-item-name" title="${escapeHtml(f.name)}">${escapeHtml(f.name)}</span>
                </div>
                <span class="file-item-size">${formatSize(f.size)}</span>
                <button class="file-item-remove" data-index="${i}" title="${t('common.remove')}">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                </button>
            </div>`;
        }).join('');

        // A file the browser cannot decode (corrupt, or a mislabelled type)
        // falls back to the type badge rather than leaving a broken-image icon.
        container.querySelectorAll('.file-item-thumb').forEach(img => {
            img.addEventListener('error', () => {
                const badge = document.createElement('div');
                badge.className = 'file-item-icon ' + img.dataset.fallbackCls;
                badge.textContent = img.dataset.fallbackLabel;
                img.replaceWith(badge);
            });
        });

        container.querySelectorAll('.file-item-remove').forEach(btn => {
            btn.addEventListener('click', () => removeFile(parseInt(btn.dataset.index)));
        });
    }

    const CHUNK_SIZE = 8 * 1024 * 1024;        // 8MB chunks (CrowdSec-compatible)
    const CHUNK_THRESHOLD = 100 * 1024 * 1024; // Use chunked upload for files >100MB

    async function startUpload() {
        if (selectedFiles.length === 0) return;

        const password = document.getElementById('upload-password').value;
        const expiresIn = parseInt(document.getElementById('upload-expiry').value) || 24;
        const maxDownloads = parseInt(document.getElementById('upload-max-downloads').value) || 0;

        const uploadBtn = document.getElementById('upload-btn');
        uploadBtn.disabled = true;
        uploadBtn.textContent = t('upload.uploading');

        const progressDiv = document.getElementById('upload-progress');
        const resultsDiv = document.getElementById('upload-results');
        progressDiv.style.display = '';
        progressDiv.innerHTML = '';
        resultsDiv.style.display = '';
        resultsDiv.innerHTML = '';

        for (const file of selectedFiles) {
            const progressId = 'prog-' + Math.random().toString(36).slice(2, 8);
            progressDiv.innerHTML += `
                <div class="progress-wrapper" id="${progressId}">
                    <div class="progress-header">
                        <span class="progress-name">${escapeHtml(file.name)}</span>
                        <span class="progress-percent">0%</span>
                    </div>
                    <div class="progress-bar-outer">
                        <div class="progress-bar-inner"></div>
                    </div>
                </div>
            `;

            try {
                let result;
                if (file.size > CHUNK_THRESHOLD) {
                    result = await uploadChunked(file, password, expiresIn, maxDownloads, progressId);
                } else {
                    result = await uploadSimple(file, password, expiresIn, maxDownloads, progressId);
                }
                updateProgress(progressId, 100, true);
                renderUploadResult(resultsDiv, result, file);
            } catch (err) {
                // A dead/unreachable server surfaces as a TypeError from fetch
                // ("Failed to fetch") — useless to a user, so name the actual
                // condition instead.
                const msg = (err instanceof TypeError)
                    ? t('upload.networkError')
                    : (err.message || t('upload.error'));
                updateProgress(progressId, 100, false, true, msg);
                toast(`${file.name}: ${msg}`, 'error');
            }
        }

        selectedFiles = [];
        renderFileList();
        document.getElementById('upload-options').style.display = 'none';
        uploadBtn.disabled = false;
        uploadBtn.textContent = t('upload.submit');
    }

    // A failed upload must say so. Showing "100%" next to a red bar reads as
    // "finished" or "stuck" — the toast that carried the reason is gone seconds
    // later, so the reason is written into the progress row and stays there.
    function updateProgress(id, percent, complete = false, error = false, message = '') {
        const wrapper = document.getElementById(id);
        if (!wrapper) return;
        const bar = wrapper.querySelector('.progress-bar-inner');
        const pct = wrapper.querySelector('.progress-percent');
        bar.style.width = percent + '%';
        pct.textContent = Math.round(percent) + '%';
        if (complete) bar.classList.add('complete');
        if (error) {
            bar.classList.add('error');
            pct.textContent = t('upload.error');
            pct.classList.add('error');
            let detail = wrapper.querySelector('.progress-error');
            if (!detail) {
                detail = document.createElement('div');
                detail.className = 'progress-error';
                wrapper.appendChild(detail);
            }
            // textContent: the message can be a raw server response body.
            detail.textContent = message || t('upload.error');
        }
    }

    async function uploadSimple(file, password, expiresIn, maxDownloads, progressId) {
        const formData = new FormData();
        formData.append('file', file);
        if (password) formData.append('password', password);
        formData.append('expires_in', expiresIn.toString());
        if (maxDownloads) formData.append('max_downloads', maxDownloads.toString());

        return new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest();
            xhr.open('POST', '/api/upload');
            xhr.withCredentials = true;

            xhr.upload.addEventListener('progress', (e) => {
                if (e.lengthComputable) {
                    updateProgress(progressId, (e.loaded / e.total) * 100);
                }
            });

            xhr.addEventListener('load', () => {
                if (xhr.status >= 200 && xhr.status < 300) {
                    try {
                        resolve(JSON.parse(xhr.responseText));
                    } catch {
                        resolve({ url: xhr.responseText.trim() });
                    }
                } else {
                    // Keep the status code when the body is empty — "failed"
                    // alone tells nobody whether it was 413, 401 or 500.
                    reject(new Error(xhr.responseText.trim() || `${t('upload.error')} (HTTP ${xhr.status})`));
                }
            });

            xhr.addEventListener('error', () => reject(new Error(t('upload.networkError'))));
            xhr.addEventListener('abort', () => reject(new Error(t('upload.networkError'))));
            xhr.send(formData);
        });
    }

    // A resumable upload is remembered per (name, size, mtime, chunk size) so a
    // reload or a dropped connection continues instead of starting over. The
    // key deliberately excludes content: hashing gigabytes to make a key would
    // cost more than re-sending a few chunks. A resumed upload only skips
    // chunks the SERVER confirms it holds, so a stale key can never corrupt a
    // file — at worst it wastes one status round trip.
    const RESUME_PREFIX = 'casadrop_resume_';
    const RESUME_TTL_MS = 24 * 60 * 60 * 1000; // matches the server-side chunk TTL

    function resumeKey(file) {
        return `${RESUME_PREFIX}${file.name}|${file.size}|${file.lastModified}|${CHUNK_SIZE}`;
    }

    function loadResume(file) {
        try {
            const raw = localStorage.getItem(resumeKey(file));
            if (!raw) return null;
            const rec = JSON.parse(raw);
            if (!rec.uploadId || Date.now() - (rec.savedAt || 0) > RESUME_TTL_MS) {
                localStorage.removeItem(resumeKey(file));
                return null;
            }
            return rec;
        } catch { return null; }
    }

    function saveResume(file, uploadId) {
        try {
            localStorage.setItem(resumeKey(file), JSON.stringify({ uploadId, savedAt: Date.now() }));
        } catch { /* storage disabled/full: resume is a convenience, not required */ }
    }

    function clearResume(file) {
        try { localStorage.removeItem(resumeKey(file)); } catch { /* ignore */ }
    }

    // Ask the server which chunks a remembered upload already has. Any failure
    // (expired, gone, storage cleared) means "start fresh", never an error.
    async function resumeState(file) {
        const rec = loadResume(file);
        if (!rec) return null;
        try {
            const res = await api(`/api/upload/chunk/${rec.uploadId}`);
            if (!res.ok) { clearResume(file); return null; }
            const status = await res.json();
            return { uploadId: rec.uploadId, received: new Set(status.received || []) };
        } catch { clearResume(file); return null; }
    }

    async function uploadChunked(file, password, expiresIn, maxDownloads, progressId) {
        const totalChunks = Math.ceil(file.size / CHUNK_SIZE);

        // Resume a remembered upload if the server still has it; otherwise init.
        let uploadId, received;
        const resumed = await resumeState(file);
        if (resumed) {
            uploadId = resumed.uploadId;
            received = resumed.received;
        } else {
            const initRes = await api('/api/upload/chunk/init', {
                method: 'POST',
                body: JSON.stringify({ fileName: file.name, totalSize: file.size, totalChunks }),
            });
            if (!initRes.ok) throw new Error(await initRes.text());
            uploadId = (await initRes.json()).uploadId;
            received = new Set();
            saveResume(file, uploadId);
        }

        for (let i = 0; i < totalChunks; i++) {
            if (received.has(i)) {
                updateProgress(progressId, ((i + 1) / totalChunks) * 95);
                continue; // already on the server from an earlier attempt
            }
            const chunk = file.slice(i * CHUNK_SIZE, Math.min((i + 1) * CHUNK_SIZE, file.size));
            await sendChunkWithRetry(uploadId, i, chunk);
            updateProgress(progressId, ((i + 1) / totalChunks) * 95);
        }

        const finalRes = await api(`/api/upload/chunk/${uploadId}/finalize`, {
            method: 'POST',
            body: JSON.stringify({ password: password || '', expires_in: expiresIn, max_downloads: maxDownloads }),
        });
        if (!finalRes.ok) throw new Error(await finalRes.text());

        clearResume(file); // the upload is now a share; nothing left to resume
        return await finalRes.json();
    }

    // One transient network failure per chunk should not restart a multi-GB
    // upload. Retry a few times with backoff; a 4xx (bad index, size cap, gone)
    // is a real answer and is not retried.
    async function sendChunkWithRetry(uploadId, index, chunk, attempts = 3) {
        for (let attempt = 1; ; attempt++) {
            try {
                const res = await api(`/api/upload/chunk/${uploadId}?index=${index}`, {
                    method: 'POST',
                    body: chunk,
                    headers: { 'Content-Type': 'application/octet-stream' },
                });
                if (res.ok) return;
                // 4xx is a decision, not a hiccup — surface it immediately.
                if (res.status >= 400 && res.status < 500) throw new Error(await res.text());
                if (attempt >= attempts) throw new Error(await res.text());
            } catch (err) {
                if (attempt >= attempts) throw err;
            }
            await new Promise(r => setTimeout(r, 500 * attempt));
        }
    }

    function renderUploadResult(container, result, file) {
        const shareUrl = result.share_url || result.url || result.shareUrl || '';
        const shareId = result.id || shareUrl.split('/').pop() || '';

        const card = document.createElement('div');
        card.className = 'result-card';
        card.innerHTML = `
            <div class="result-header">
                <div class="result-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 11-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
                </div>
                <div>
                    <div class="result-filename">${escapeHtml(file.name)}</div>
                    <div class="result-size">${formatSize(file.size)}</div>
                </div>
            </div>
            ${shareUrl ? `
            <div class="result-url-row">
                <div class="result-url" title="${escapeHtml(shareUrl)}">${escapeHtml(shareUrl)}</div>
                <button class="btn btn-ghost btn-sm copy-url-btn" data-url="${escapeHtml(shareUrl)}">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>
                    ${t('common.copy')}
                </button>
            </div>
            ${shareId ? `<div class="result-qr"><img src="/qr/${escapeHtml(shareId)}" alt="QR Code" loading="lazy"></div>` : ''}
            ` : ''}
        `;
        container.appendChild(card);

        card.querySelector('.copy-url-btn')?.addEventListener('click', (e) => {
            copyToClipboard(e.currentTarget.dataset.url);
        });
    }

    // ==========================================
    // Host Share (browse server paths, share without re-upload) — admin only
    // ==========================================
    const ICON_FOLDER = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>';
    const ICON_FILE = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><path d="M13 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V9z"/><polyline points="13 2 13 9 20 9"/></svg>';

    async function loadHostBrowser(path) {
        const listEl = document.getElementById('hostshare-list');
        const crumbEl = document.getElementById('hostshare-breadcrumb');
        if (!listEl) return;
        listEl.innerHTML = '<div style="text-align:center;padding:40px"><span class="spinner"></span></div>';

        let data;
        try {
            const res = await api(`/api/browse?path=${encodeURIComponent(path || '/')}`);
            if (!res.ok) throw new Error(await res.text());
            data = await res.json();
        } catch (err) {
            listEl.innerHTML = `<div class="empty-state"><p>${escapeHtml(err.message || t('hostshare.error'))}</p></div>`;
            toast(t('hostshare.error'), 'error');
            return;
        }

        currentHostPath = data.path || '/';
        // The root listing enumerates exactly the allowed roots — remember them
        // so the breadcrumb knows which ancestor segments are browsable.
        if (currentHostPath === '/') {
            hostRoots = (data.entries || []).map(e => e.path).filter(Boolean);
        }
        renderHostBreadcrumb(crumbEl, currentHostPath, data.parent);

        const entries = data.entries || [];
        if (entries.length === 0) {
            listEl.innerHTML = `<div class="empty-state"><p>${t('hostshare.empty')}</p></div>`;
            return;
        }

        listEl.innerHTML = entries.map(e => {
            // Image files get a real preview instead of the generic file icon.
            // The server marks them with is_image (only formats it can decode),
            // and serves the preview admin-gated from the host path itself.
            // loading="lazy": a photo folder must not fire hundreds of requests.
            const icon = e.is_dir
                ? ICON_FOLDER
                : (e.is_image
                    ? `<img class="host-thumb" src="/api/browse/thumbnail?path=${encodeURIComponent(e.path)}" alt="" loading="lazy" data-host-thumb>`
                    : ICON_FILE);
            const meta = e.is_dir ? t('hostshare.folder') : formatSize(e.size || 0);
            // Folders: clicking the name navigates in. Both files and folders get a Share button.
            const nameCell = e.is_dir
                ? `<button class="host-open host-name" data-path="${escapeHtml(e.path)}">${icon}<span>${escapeHtml(e.name)}</span></button>`
                : `<span class="host-name">${icon}<span>${escapeHtml(e.name)}</span></span>`;
            return `
                <div class="host-row">
                    ${nameCell}
                    <span class="host-meta">${escapeHtml(meta)}</span>
                    <button class="btn btn-primary btn-sm host-share-btn"
                            data-path="${escapeHtml(e.path)}"
                            data-name="${escapeHtml(e.name)}"
                            data-isdir="${e.is_dir ? '1' : '0'}"
                            data-size="${e.size || 0}"
                            data-i18n="hostshare.share">Share</button>
                </div>`;
        }).join('');
        attachHostThumbFallbacks(listEl);
        applyI18n();
    }

    // A preview can fail for a file the extension promised was an image
    // (truncated, mislabelled, unreadable). Swap the <img> for the file icon
    // rather than leaving a broken-image box. Replaces only the <img> itself —
    // the row's name and buttons must survive.
    //
    // No inline onerror: the strict CSP (script-src 'self') blocks it.
    function attachHostThumbFallbacks(root) {
        root.querySelectorAll('img[data-host-thumb]').forEach(img => {
            img.addEventListener('error', () => {
                const span = document.createElement('span');
                span.innerHTML = ICON_FILE;
                img.replaceWith(span.firstElementChild || span);
            }, { once: true });
        });
    }

    // A host path is browsable if it is the virtual root ("/", which lists the
    // allowed roots) or sits at/below one of the allowed roots. Ancestors above
    // every allowed root (e.g. "/DATA" when only "/DATA/Media" is shared) are
    // NOT browsable and must not be turned into links that 403.
    function isHostBrowsable(p) {
        if (p === '/') return true;
        if (!hostRoots.length) return true; // roots not learned yet — don't over-restrict
        return hostRoots.some(r => p === r || p.startsWith(r + '/'));
    }

    function renderHostBreadcrumb(el, path, parent) {
        if (!el) return;
        const crumbs = [`<button class="crumb host-open" data-path="/">${t('hostshare.home')}</button>`];
        if (path && path !== '/') {
            // Build cumulative absolute paths for each segment (server re-validates each).
            const segs = path.split('/').filter(Boolean);
            let acc = '';
            segs.forEach((seg, i) => {
                acc += '/' + seg;
                const last = i === segs.length - 1;
                crumbs.push('<span class="crumb-sep">/</span>');
                if (last) {
                    crumbs.push(`<span class="crumb crumb-current">${escapeHtml(seg)}</span>`);
                } else if (isHostBrowsable(acc)) {
                    crumbs.push(`<button class="crumb host-open" data-path="${escapeHtml(acc)}">${escapeHtml(seg)}</button>`);
                } else {
                    // Non-browsable ancestor: inert text, not a link (avoids the
                    // 403 → login-screen trap). "Start" still returns to roots.
                    crumbs.push(`<span class="crumb crumb-muted">${escapeHtml(seg)}</span>`);
                }
            });
        }
        // parent shortcut is implicit via the breadcrumb segments; keep signature stable.
        void parent;
        el.innerHTML = crumbs.join('');
    }

    function openHostShareDialog(entry) {
        const isDir = entry.isdir === '1';
        const expiryOpts = [
            ['1', 'upload.expiry.1h'], ['6', 'upload.expiry.6h'], ['12', 'upload.expiry.12h'],
            ['24', 'upload.expiry.1d'], ['72', 'upload.expiry.3d'], ['168', 'upload.expiry.7d'],
            ['336', 'upload.expiry.14d'], ['720', 'upload.expiry.30d'], ['0', 'upload.expiry.never'],
        ].map(([v, k]) => `<option value="${v}"${v === '24' ? ' selected' : ''}>${t(k)}</option>`).join('');

        showModal(`
            <h3>${t('hostshare.dialogTitle')}</h3>
            <div class="host-share-target">${isDir ? ICON_FOLDER : ICON_FILE}<span>${escapeHtml(entry.name)}</span></div>
            <div id="hostshare-form">
                <div class="form-group">
                    <label>${t('upload.expiry')}</label>
                    <select id="hostshare-expiry">${expiryOpts}</select>
                </div>
                <div class="form-group">
                    <label>${t('upload.password')}</label>
                    <input type="password" id="hostshare-password" autocomplete="new-password">
                </div>
                <div class="form-group">
                    <label>${t('upload.maxDownloads')}</label>
                    <input type="number" id="hostshare-max-downloads" value="0" min="0">
                </div>
                ${isDir ? '' : `
                <div class="form-group">
                    <label class="toggle-label">
                        <input type="checkbox" id="hostshare-symlink" checked>
                        <span class="toggle-switch"></span>
                        <span>${t('hostshare.symlink')}</span>
                    </label>
                </div>`}
                <div class="form-actions">
                    <button class="btn btn-primary" id="hostshare-submit">${t('hostshare.share')}</button>
                    <button class="btn btn-ghost" id="hostshare-cancel">${t('receive.cancel')}</button>
                </div>
            </div>
            <div id="hostshare-result"></div>
        `);

        document.getElementById('hostshare-cancel')?.addEventListener('click', closeModal);
        document.getElementById('hostshare-submit')?.addEventListener('click', () => submitHostShare(entry));
    }

    async function submitHostShare(entry) {
        const isDir = entry.isdir === '1';
        const submitBtn = document.getElementById('hostshare-submit');
        const body = {
            path: entry.path,
            password: document.getElementById('hostshare-password')?.value || '',
            expires_in: parseInt(document.getElementById('hostshare-expiry')?.value || '24', 10),
            max_downloads: parseInt(document.getElementById('hostshare-max-downloads')?.value || '0', 10),
        };
        if (!isDir) body.use_symlink = !!document.getElementById('hostshare-symlink')?.checked;

        if (submitBtn) { submitBtn.disabled = true; submitBtn.innerHTML = '<span class="spinner"></span>'; }
        try {
            const res = await api(isDir ? '/api/share-folder' : '/api/share-from-path', {
                method: 'POST',
                body: JSON.stringify(body),
            });
            if (!res.ok) throw new Error(await res.text());
            const result = await res.json();

            // Reuse the upload result renderer (link + QR). Folder size comes from
            // the response since the browse entry reports 0 for directories.
            document.getElementById('hostshare-form')?.remove();
            document.querySelector('.host-share-target')?.remove();
            const container = document.getElementById('hostshare-result');
            container.innerHTML = '';
            renderUploadResult(container, result, { name: entry.name, size: result.file_size || parseInt(entry.size, 10) || 0 });
            const closeRow = document.createElement('div');
            closeRow.className = 'form-actions';
            closeRow.innerHTML = `<button class="btn btn-ghost" id="hostshare-done">${t('hostshare.done')}</button>`;
            container.appendChild(closeRow);
            document.getElementById('hostshare-done')?.addEventListener('click', closeModal);
            toast(t('hostshare.success'), 'success');
        } catch (err) {
            if (submitBtn) { submitBtn.disabled = false; submitBtn.textContent = t('hostshare.share'); }
            toast(err.message || t('hostshare.error'), 'error');
        }
    }

    // ==========================================
    // Shares
    // ==========================================
    async function loadShares() {
        const listEl = document.getElementById('shares-list');
        const emptyEl = document.getElementById('shares-empty');
        const statsEl = document.getElementById('shares-stats');

        listEl.innerHTML = '<div style="text-align:center;padding:40px"><span class="spinner"></span></div>';
        emptyEl.style.display = 'none';

        try {
            const [sharesRes, statsRes, taildropRes] = await Promise.all([
                api('/api/shares'),
                api('/api/stats'),
                api('/api/taildrop/status').catch(() => null),
            ]);

            const shares = sharesRes.ok ? await sharesRes.json() : [];
            currentShares = shares || [];
            const stats = statsRes.ok ? await statsRes.json() : {};

            // Taildrop is admin-only; non-admins simply get no send button.
            if (taildropRes && taildropRes.ok) {
                const td = await taildropRes.json();
                taildropState = { available: !!td.available, devices: td.devices || [] };
            } else {
                taildropState = { available: false, devices: [] };
            }

            statsEl.innerHTML = `
                <div class="stat-item">
                    <span class="stat-value">${stats.total_shares || 0}</span>
                    <span class="stat-label">${t('stat.totalShares')}</span>
                </div>
                <div class="stat-item">
                    <span class="stat-value">${stats.total_downloads || 0}</span>
                    <span class="stat-label">${t('stat.totalDownloads')}</span>
                </div>
                <div class="stat-item">
                    <span class="stat-value">${formatSize(stats.total_size || 0)}</span>
                    <span class="stat-label">${t('stat.totalSize')}</span>
                </div>
                <div class="stat-item">
                    <span class="stat-value">${stats.protected_shares || 0}</span>
                    <span class="stat-label">${t('stat.protected')}</span>
                </div>
            `;

            if (!shares || shares.length === 0) {
                listEl.innerHTML = '';
                emptyEl.style.display = '';
                return;
            }

            listEl.innerHTML = shares.map(share => {
                const url = share.share_url || share.url || '';
                const dlCount = share.max_downloads > 0
                    ? `${share.downloads || 0}/${share.max_downloads}`
                    : `${share.downloads || 0}`;

                const isExpiringSoon = share.expires_at && (new Date(share.expires_at) - new Date()) < 3600000;

                return `
                <div class="share-card" data-id="${escapeHtml(share.id)}">
                    <input type="checkbox" class="share-checkbox" data-id="${escapeHtml(share.id)}" style="width:16px;height:16px;cursor:pointer;flex-shrink:0;margin-right:8px;accent-color:var(--primary)">
                    <div class="share-thumb">${shareThumbHTML(share)}</div>
                    <div class="share-info">
                        <div class="share-name" title="${escapeHtml(share.original_name || share.file_name || '')}">${escapeHtml(share.original_name || share.file_name || share.id)}</div>
                        <div class="share-meta">
                            <span class="share-meta-item">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                                ${dlCount}
                            </span>
                            <span class="share-meta-item">${formatSize(share.file_size || 0)}</span>
                            <span class="share-meta-item">${relativeExpiry(share.expires_at)}</span>
                            ${share.has_password ? '<span class="badge badge-password">' + escapeHtml(t('stat.protected')) + '</span>' : ''}
                            ${share.is_directory ? '<span class="badge badge-directory">' + escapeHtml(t('common.folder')) + '</span>' : ''}
                            ${isExpiringSoon ? '<span class="badge badge-expiring">' + escapeHtml(t('common.expiring')) + '</span>' : ''}
                        </div>
                    </div>
                    <div class="share-actions">
                        ${url ? `<button class="btn-icon" title="${t('common.copyLink')}" data-copy="${escapeHtml(url)}">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>
                        </button>` : ''}
                        <button class="btn-icon" title="${t('common.qr')}" data-qr="${escapeHtml(share.id)}">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="3" height="3"/></svg>
                        </button>
                        <button class="btn-icon edit-share-btn" data-id="${escapeHtml(share.id)}" title="${t('common.edit')}">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                                <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/>
                                <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/>
                            </svg>
                        </button>
                        <button class="btn-icon email-share-btn" data-id="${escapeHtml(share.id)}" title="${t('shares.sendEmail')}">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                                <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
                                <polyline points="22,6 12,13 2,6"/>
                            </svg>
                        </button>
                        <button class="btn-icon" data-activity="${escapeHtml(share.id)}" data-name="${escapeHtml(share.original_name || share.file_name || '')}" title="${t('activity.forShare')}">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                                <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
                            </svg>
                        </button>
                        ${(taildropState.available && !share.is_directory) ? `<button class="btn-icon taildrop-share-btn" data-id="${escapeHtml(share.id)}" title="${t('shares.taildrop')}">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                                <path d="M22 2L11 13"/><path d="M22 2l-7 20-4-9-9-4 20-7z"/>
                            </svg>
                        </button>` : ''}
                        <button class="btn-icon danger" title="${t('common.delete')}" data-delete="${escapeHtml(share.id)}">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>
                        </button>
                    </div>
                </div>
                `;
            }).join('');

            // Swap broken thumbnails (e.g. source file deleted) for type icons
            attachThumbFallbacks(listEl);

            // Attach events
            listEl.querySelectorAll('[data-copy]').forEach(btn => {
                btn.addEventListener('click', () => copyToClipboard(btn.dataset.copy));
            });

            listEl.querySelectorAll('[data-qr]').forEach(btn => {
                btn.addEventListener('click', () => {
                    showModal(`
                        <h3>${t('common.qr')}</h3>
                        <div style="text-align:center;padding:20px">
                            <img src="/qr/${escapeHtml(btn.dataset.qr)}" alt="QR" style="width:200px;height:200px;background:#fff;padding:12px;border-radius:8px">
                        </div>
                        <div class="modal-actions">
                            <button class="btn btn-ghost" id="modal-close-btn">${t('common.close')}</button>
                        </div>
                    `);
                    document.getElementById('modal-close-btn')?.addEventListener('click', closeModal);
                });
            });

            listEl.querySelectorAll('[data-activity]').forEach(btn => {
                btn.addEventListener('click', () => showShareActivity(btn.dataset.activity, btn.dataset.name));
            });

            listEl.querySelectorAll('[data-delete]').forEach(btn => {
                btn.addEventListener('click', async () => {
                    if (!confirm(t('shares.deleteConfirm'))) return;
                    try {
                        const res = await api(`/api/shares/${btn.dataset.delete}`, { method: 'DELETE' });
                        if (res.ok) {
                            toast(t('shares.deleted'), 'success');
                            loadShares();
                        } else {
                            toast(t('toast.error'), 'error');
                        }
                    } catch { toast(t('toast.error'), 'error'); }
                });
            });

            listEl.querySelectorAll('.edit-share-btn').forEach(btn => {
                btn.addEventListener('click', () => showEditShareModal(btn.dataset.id));
            });

            listEl.querySelectorAll('.email-share-btn').forEach(btn => {
                btn.addEventListener('click', () => showEmailShareModal(btn.dataset.id));
            });

            listEl.querySelectorAll('.taildrop-share-btn').forEach(btn => {
                btn.addEventListener('click', () => showTaildropModal(btn.dataset.id));
            });

            // Bulk checkbox logic
            const bulkBar = document.getElementById('bulk-actions');
            const selectAllCb = document.getElementById('select-all-shares');
            const selectedCountEl = document.getElementById('selected-count');

            function updateBulkBar() {
                const checkboxes = listEl.querySelectorAll('.share-checkbox');
                const checked = listEl.querySelectorAll('.share-checkbox:checked');
                if (bulkBar) {
                    bulkBar.style.display = checked.length > 0 ? 'flex' : 'none';
                }
                if (selectedCountEl) {
                    selectedCountEl.textContent = t('shares.selectedCount').replace('{n}', checked.length);
                }
                if (selectAllCb) {
                    selectAllCb.checked = checkboxes.length > 0 && checked.length === checkboxes.length;
                    selectAllCb.indeterminate = checked.length > 0 && checked.length < checkboxes.length;
                }
            }

            listEl.querySelectorAll('.share-checkbox').forEach(cb => {
                cb.addEventListener('change', updateBulkBar);
                cb.addEventListener('click', (e) => e.stopPropagation());
            });

            if (selectAllCb) {
                selectAllCb.onchange = () => {
                    listEl.querySelectorAll('.share-checkbox').forEach(cb => {
                        cb.checked = selectAllCb.checked;
                    });
                    updateBulkBar();
                };
            }

            const bulkDeleteBtn = document.getElementById('bulk-delete-btn');
            if (bulkDeleteBtn) {
                bulkDeleteBtn.onclick = async () => {
                    const ids = Array.from(listEl.querySelectorAll('.share-checkbox:checked')).map(cb => cb.dataset.id);
                    if (ids.length === 0) return;
                    if (!confirm(`${t('shares.bulkDelete')} (${ids.length})?`)) return;
                    try {
                        const res = await api('/api/shares/bulk-delete', {
                            method: 'POST',
                            body: JSON.stringify({ ids }),
                        });
                        if (res.ok) {
                            toast(`${ids.length} ${t('shares.deleted')}`, 'success');
                            loadShares();
                        } else {
                            toast(t('toast.error'), 'error');
                        }
                    } catch { toast(t('toast.error'), 'error'); }
                };
            }

        } catch {
            listEl.innerHTML = '';
            toast(t('toast.error'), 'error');
        }
    }

    // ==========================================
    // Edit Share Modal
    // ==========================================
    function showEditShareModal(shareId) {
        const share = currentShares.find(s => s.id === shareId);
        if (!share) return;

        // Sentinel: backend stores year 9999 for "unbegrenzt" shares (see
        // relativeExpiry). Detect it so an already-unlimited share opens with
        // the toggle on instead of an astronomically large hour count.
        const isUnlimited = new Date(share.expires_at).getFullYear() > 9000;
        // An unlimited share has no hour count. Showing one anyway ("24") in a
        // greyed-out field reads like a real expiry and made people think the
        // share would die tomorrow. Leave it empty and say "unbegrenzt" in the
        // placeholder; the default only appears once the toggle is switched off.
        const remaining = isUnlimited ? '' : Math.max(1, Math.ceil((new Date(share.expires_at) - Date.now()) / 3600000));
        const shareName = share.original_name || share.file_name || share.id;

        const html = `
            <h3>${t('shares.edit')}</h3>
            <p class="modal-subject" title="${escapeHtml(shareName)}">${escapeHtml(shareName)}</p>
            <div class="form-group">
                <label>${t('upload.expiry')} (${t('upload.expiry.hours')})</label>
                <input type="number" id="edit-expiry" value="${remaining}" min="1" max="8760" placeholder="${escapeHtml(t('shares.unlimited'))}"${isUnlimited ? ' disabled' : ''}>
                <label class="toggle-label" style="margin-top:10px">
                    <input type="checkbox" id="edit-unlimited"${isUnlimited ? ' checked' : ''}>
                    <span class="toggle-switch"></span>
                    <span>${t('upload.expiry.never')}</span>
                </label>
            </div>
            <div class="form-group">
                <label>${t('upload.maxDownloads')}</label>
                <input type="number" id="edit-max-downloads" value="${share.max_downloads || 0}" min="0">
            </div>
            <div class="form-group">
                <label>${t('upload.password')} (${share.has_password ? t('shares.changePassword') : t('shares.addPassword')})</label>
                <input type="password" id="edit-password" placeholder="${share.has_password ? '••••••••' : ''}">
            </div>
            <div class="modal-actions">
                <button class="btn btn-ghost" id="cancel-edit-btn">${t('receive.cancel')}</button>
                <button class="btn btn-primary" id="save-edit-btn">${t('shares.save')}</button>
            </div>
        `;

        showModal(html);

        document.getElementById('cancel-edit-btn').addEventListener('click', closeModal);

        // Unlimited toggle disables the hour input; when on we send 0, which the
        // backend maps to the "never expires" sentinel.
        const unlimitedEl = document.getElementById('edit-unlimited');
        const expiryEl = document.getElementById('edit-expiry');
        unlimitedEl.addEventListener('change', () => {
            expiryEl.disabled = unlimitedEl.checked;
            if (unlimitedEl.checked) {
                expiryEl.value = '';
            } else if (expiryEl.value === '') {
                // Switching the toggle off needs a number to edit; 24h is the
                // same default the upload form offers.
                expiryEl.value = '24';
            }
        });

        document.getElementById('save-edit-btn').onclick = async () => {
            const body = {};
            if (unlimitedEl.checked) {
                body.expires_in_hours = 0; // unbegrenzt
            } else {
                const expiry = parseInt(expiryEl.value, 10);
                if (!(expiry > 0)) {
                    // Silently omitting the field would leave the share exactly
                    // as it was while the dialog reported success — say so
                    // instead of pretending the edit landed.
                    toast(t('shares.expiryRequired'), 'error');
                    return;
                }
                body.expires_in_hours = expiry;
            }
            const maxDl = parseInt(document.getElementById('edit-max-downloads').value);
            if (!isNaN(maxDl)) body.max_downloads = maxDl;
            const pw = document.getElementById('edit-password').value;
            if (pw !== '') body.password = pw;

            try {
                const resp = await fetch('/api/shares/' + shareId, {
                    method: 'PUT',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(body)
                });
                if (!resp.ok) throw new Error(await resp.text());
                toast(t('shares.updated'), 'success');
                closeModal();
                loadShares();
            } catch(e) {
                toast(e.message || 'Failed to update share', 'error');
            }
        };
    }

    // ==========================================
    // Email Share Modal
    // ==========================================
    function showEmailShareModal(shareId) {
        const share = currentShares.find(s => s.id === shareId);
        if (!share) return;

        const html = `
            <h3>${t('shares.sendEmail')}</h3>
            <p class="modal-subject" title="${escapeHtml(share.original_name || share.file_name || share.id)}">${escapeHtml(share.original_name || share.file_name || share.id)}</p>
            <div class="form-group">
                <label>${t('email.recipientEmail')}</label>
                <input type="email" id="email-recipient" required placeholder="name@example.com">
            </div>
            <div class="form-group">
                <label>${t('email.recipientName')}</label>
                <input type="text" id="email-recipient-name" placeholder="${t('email.optional')}">
            </div>
            <div class="form-group">
                <label>${t('email.message')}</label>
                <textarea id="email-message" rows="3" placeholder="${t('email.optional')}" style="width:100%;padding:8px 12px;background:var(--bg-input);border:1px solid var(--border);border-radius:var(--radius);color:var(--text-primary);font-family:var(--font);resize:vertical"></textarea>
            </div>
            <div class="modal-actions">
                <button class="btn btn-ghost" id="cancel-email-btn">${t('receive.cancel')}</button>
                <button class="btn btn-primary" id="send-email-btn">${t('email.send')}</button>
            </div>
        `;
        showModal(html);

        document.getElementById('cancel-email-btn').addEventListener('click', closeModal);
        document.getElementById('send-email-btn').onclick = async () => {
            const recipient = document.getElementById('email-recipient').value;
            if (!recipient) { toast(t('email.enterRecipient'), 'error'); return; }
            try {
                const res = await api('/api/email/send', {
                    method: 'POST',
                    body: JSON.stringify({
                        share_id: shareId,
                        recipient_email: recipient,
                        recipient_name: document.getElementById('email-recipient-name').value,
                        message: document.getElementById('email-message').value,
                        notify_download: true,
                        lang: LANG,
                    }),
                });
                if (!res.ok) {
                    // JSON {"error"} from the send/record step, plain text from
                    // validation — show the reason, not the raw response body.
                    const body = (await res.text()).trim();
                    let detail = body;
                    try { detail = JSON.parse(body).error || body; } catch (_) { /* plain text */ }
                    throw new Error(t('email.sendFailed') + ' ' + detail);
                }
                toast(t('email.sent'), 'success');
                closeModal();
            } catch(e) { toast(e.message || t('toast.error'), 'error'); }
        };
    }

    function showTaildropModal(shareId) {
        const share = currentShares.find(s => s.id === shareId);
        if (!share) return;

        const devices = taildropState.devices || [];
        const body = devices.length === 0
            ? `<p style="color:var(--text-secondary)">${t('taildrop.noDevices')}</p>`
            : `<div class="form-group">
                   <label>${t('taildrop.device')}</label>
                   <select id="taildrop-device" style="width:100%;padding:8px 12px;background:var(--bg-input);border:1px solid var(--border);border-radius:var(--radius);color:var(--text-primary);font-family:var(--font)">
                       ${devices.map(d => `<option value="${escapeHtml(d.dnsName)}">${escapeHtml(d.name || d.dnsName)}${d.online ? '' : ' (offline)'}</option>`).join('')}
                   </select>
               </div>`;

        showModal(`
            <h3>${t('taildrop.title')}</h3>
            <p class="modal-subject" title="${escapeHtml(share.original_name || share.file_name || share.id)}">${escapeHtml(share.original_name || share.file_name || share.id)}</p>
            ${body}
            <div class="modal-actions">
                <button class="btn btn-ghost" id="taildrop-cancel-btn">${t('receive.cancel')}</button>
                ${devices.length ? `<button class="btn btn-primary" id="taildrop-send-btn">${t('taildrop.send')}</button>` : ''}
            </div>
        `);

        document.getElementById('taildrop-cancel-btn').addEventListener('click', closeModal);
        const sendBtn = document.getElementById('taildrop-send-btn');
        if (sendBtn) {
            sendBtn.addEventListener('click', async () => {
                const device = document.getElementById('taildrop-device').value;
                if (!device) { toast(t('taildrop.noDevices'), 'error'); return; }
                sendBtn.disabled = true;
                const original = sendBtn.textContent;
                sendBtn.textContent = t('taildrop.sending');
                try {
                    const res = await api('/api/taildrop/send', {
                        method: 'POST',
                        body: JSON.stringify({ shareId: shareId, device: device }),
                    });
                    if (!res.ok) throw new Error(await res.text());
                    toast(t('taildrop.sent'), 'success');
                    closeModal();
                } catch (e) {
                    toast(e.message || t('toast.error'), 'error');
                    sendBtn.disabled = false;
                    sendBtn.textContent = original;
                }
            });
        }
    }

    // ==========================================
    // Receive Links
    // ==========================================
    async function loadReceiveLinks() {
        const listEl = document.getElementById('receive-list');
        const emptyEl = document.getElementById('receive-empty');

        listEl.innerHTML = '<div style="text-align:center;padding:40px"><span class="spinner"></span></div>';
        emptyEl.style.display = 'none';

        try {
            const res = await api('/api/receive-links');
            const links = res.ok ? await res.json() : [];

            if (!links || links.length === 0) {
                listEl.innerHTML = '';
                emptyEl.style.display = '';
                return;
            }

            listEl.innerHTML = links.map(link => {
                const receiveUrl = `${window.location.origin}/r/${link.id}`;
                const uploadsText = link.max_uploads > 0
                    ? `${link.current_uploads || 0}/${link.max_uploads}`
                    : `${link.current_uploads || 0}`;

                return `
                <div class="receive-card" data-id="${escapeHtml(link.id)}">
                    <div>
                        <div class="receive-name">${escapeHtml(link.name)}</div>
                        <div class="receive-meta">
                            <span class="share-meta-item">${uploadsText} ${t('receive.uploads')}</span>
                            <span class="share-meta-item">${formatSize(link.total_size || 0)}</span>
                            ${link.expires_at ? `<span class="share-meta-item">${relativeExpiry(link.expires_at)}</span>` : ''}
                            ${link.has_password ? '<span class="badge badge-password">' + escapeHtml(t('stat.protected')) + '</span>' : ''}
                            ${link.auto_share ? '<span class="badge" style="background:var(--info-bg);color:var(--info);border:1px solid var(--info-border)">' + escapeHtml(t('common.autoShare')) + '</span>' : ''}
                        </div>
                    </div>
                    <div class="share-actions">
                        <button class="btn-icon" title="${t('common.copyLink')}" data-copy="${escapeHtml(receiveUrl)}">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>
                        </button>
                        <button class="btn-icon danger" title="${t('common.delete')}" data-delete="${escapeHtml(link.id)}">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>
                        </button>
                    </div>
                </div>
                `;
            }).join('');

            listEl.querySelectorAll('[data-copy]').forEach(btn => {
                btn.addEventListener('click', () => copyToClipboard(btn.dataset.copy));
            });

            listEl.querySelectorAll('[data-delete]').forEach(btn => {
                btn.addEventListener('click', async () => {
                    if (!confirm(t('receive.deleteConfirm'))) return;
                    try {
                        const res = await api(`/api/receive-links/${btn.dataset.delete}`, { method: 'DELETE' });
                        if (res.ok) {
                            toast(t('receive.deleted'), 'success');
                            loadReceiveLinks();
                        } else {
                            toast(t('toast.error'), 'error');
                        }
                    } catch { toast(t('toast.error'), 'error'); }
                });
            });

        } catch {
            listEl.innerHTML = '';
            toast(t('toast.error'), 'error');
        }
    }

    function initReceive() {
        const formWrapper = document.getElementById('receive-form-wrapper');
        const createBtn = document.getElementById('create-receive-btn');
        const cancelBtn = document.getElementById('receive-cancel');
        const form = document.getElementById('receive-form');

        createBtn.addEventListener('click', () => {
            formWrapper.style.display = '';
            createBtn.style.display = 'none';
        });

        cancelBtn.addEventListener('click', () => {
            formWrapper.style.display = 'none';
            createBtn.style.display = '';
            form.reset();
        });

        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            const body = {
                name: document.getElementById('receive-name').value,
                password: document.getElementById('receive-password').value || '',
                max_uploads: parseInt(document.getElementById('receive-max-uploads').value) || 0,
                max_file_size: (parseInt(document.getElementById('receive-max-file-size').value) || 0) * 1024 * 1024,
                expires_in: parseInt(document.getElementById('receive-expiry').value) || 0,
                allowed_extensions: document.getElementById('receive-extensions').value || '',
                auto_share: document.getElementById('receive-auto-share').checked,
            };

            try {
                const res = await api('/api/receive-links', {
                    method: 'POST',
                    body: JSON.stringify(body),
                });

                if (res.ok) {
                    toast(t('receive.created'), 'success');
                    formWrapper.style.display = 'none';
                    createBtn.style.display = '';
                    form.reset();
                    loadReceiveLinks();
                } else {
                    const err = await res.text();
                    toast(err || t('toast.error'), 'error');
                }
            } catch { toast(t('toast.error'), 'error'); }
        });
    }

    // ==========================================
    // Settings
    // ==========================================
    async function loadSettings() {
        await Promise.all([
            loadNetworkConfig(),
            loadFileRestrictionsConfig(),
            loadWebhookConfig(),
            loadUserManagement(),
            load2FAConfig(),
            loadAPIKeys(),
            loadSMTPConfig(),
            loadActivityLog(),
            loadSessions(),
        ]);
    }

    // ==========================================
    // Active sessions
    // ==========================================
    async function loadSessions() {
        const container = document.getElementById('sessions-config');
        if (!container) return;

        let sessions = [];
        try {
            const res = await api('/api/sessions');
            if (!res.ok) throw new Error(await res.text());
            sessions = (await res.json()).sessions || [];
        } catch (err) {
            container.innerHTML = `<p style="color:var(--danger)">${escapeHtml(err.message || t('toast.error'))}</p>`;
            return;
        }

        const isAdminView = sessions.some(s => s.user_email);
        // Current first, then newest.
        sessions.sort((a, b) => (b.current - a.current) || (new Date(b.created_at) - new Date(a.created_at)));
        const others = sessions.filter(s => !s.current).length;

        container.innerHTML = `
            <p style="font-size:var(--text-sm);color:var(--text-muted);margin-bottom:var(--space-3)">${t(isAdminView ? 'sessions.hintAdmin' : 'sessions.hint')}</p>
            <div style="overflow-x:auto">
                <table class="users-table">
                    <thead><tr>
                        <th>${t('activity.from')}</th>
                        ${isAdminView ? `<th>${t('activity.who')}</th>` : ''}
                        <th>${t('sessions.since')}</th>
                        <th></th>
                    </tr></thead>
                    <tbody>
                        ${sessions.map(s => `
                            <tr>
                                <td>
                                    <div style="font-family:var(--font-mono);font-size:var(--text-xs)">${escapeHtml(s.ip || '')}</div>
                                    <div style="font-size:var(--text-xs);color:var(--text-muted)" title="${escapeHtml(s.user_agent || '')}">${escapeHtml((s.user_agent || '').slice(0, 48))}</div>
                                </td>
                                ${isAdminView ? `<td style="font-size:var(--text-xs)">${escapeHtml(s.user_email || '')}</td>` : ''}
                                <td style="white-space:nowrap;font-size:var(--text-xs)">${escapeHtml(new Date(s.created_at).toLocaleString(LANG))}</td>
                                <td style="text-align:right">
                                    ${s.current
                                        ? `<span class="role-badge role-admin">${t('sessions.current')}</span>`
                                        : `<button class="btn btn-ghost btn-sm revoke-session-btn" data-id="${escapeHtml(s.id)}">${t('sessions.revoke')}</button>`}
                                </td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            </div>
            ${others > 0 ? `<button class="btn btn-ghost btn-sm" id="revoke-others-btn" style="margin-top:var(--space-3);color:var(--danger)">${t('sessions.revokeOthers')}</button>` : ''}
        `;

        container.querySelectorAll('.revoke-session-btn').forEach(btn => {
            btn.onclick = async () => {
                const res = await api('/api/sessions/' + encodeURIComponent(btn.dataset.id), { method: 'DELETE' });
                if (res.ok) { toast(t('sessions.revoked'), 'success'); loadSessions(); }
                else toast((await res.text()).trim() || t('toast.error'), 'error');
            };
        });
        const othersBtn = document.getElementById('revoke-others-btn');
        if (othersBtn) othersBtn.onclick = async () => {
            if (!confirm(t('sessions.confirmOthers'))) return;
            const res = await api('/api/sessions/revoke-others', { method: 'POST' });
            if (res.ok) {
                const { revoked } = await res.json();
                toast(t('sessions.revokedOthers').replace('{n}', revoked), 'success');
                loadSessions();
            } else toast(t('toast.error'), 'error');
        };
    }

    // ==========================================
    // Activity log
    // ==========================================
    const ACTIVITY_KINDS = [
        'share.created', 'share.updated', 'share.deleted', 'share.expired',
        'share.downloaded', 'share.streamed', 'receive.uploaded',
        'auth.login', 'auth.login_failed', 'auth.locked', 'auth.logout', 'auth.setup',
        'session.revoked', 'security',
    ];

    function activityKindLabel(kind) {
        const key = 'activity.kind.' + kind;
        const label = t(key);
        return label === key ? kind : label;
    }

    function activityActor(e) {
        if (e.actor_email) return escapeHtml(e.actor_email);
        if (e.actor_id) return escapeHtml(e.actor_id);
        // No request behind it (expiry sweep) vs. a request without a login.
        return `<span style="color:var(--text-muted)">${e.ip ? t('activity.anonymous') : t('activity.system')}</span>`;
    }

    // ACTIVITY_DETAIL_RULES translates the detail texts the server records for
    // sign-in and session events. The stored records stay as written (they are
    // evidence, and the CSV export keeps them verbatim); only the display is
    // translated. Every text internal/middleware writes must match one rule —
    // TestActivityDetailsAreTranslated (internal/i18n) enforces that.
    // The optional "TYPE: " prefix is what audit.AuthSink adds for event types
    // without their own kind.
    const ACTIVITY_DETAIL_RULES = [
        [/^Successful (JSON )?login \(role=(\w+)\)$/, m => t(m[1] ? 'activity.d.loginOkApi' : 'activity.d.loginOk').replace('{role}', roleLabel(m[2]))],
        [/^Failed (JSON )?login attempt (\d+)\/(\d+)$/, m => t(m[1] ? 'activity.d.loginFailedApi' : 'activity.d.loginFailed').replace('{n}', m[2]).replace('{max}', m[3])],
        [/^Account locked after max failed attempts( \(JSON\))?$/, m => t(m[1] ? 'activity.d.loginLockedApi' : 'activity.d.loginLocked')],
        [/^Admin 2FA code missing\/invalid( \(JSON\))?$/, m => t(m[1] ? 'activity.d.twoFAInvalidApi' : 'activity.d.twoFAInvalid')],
        [/^Local auth disabled \(OIDC-only\)$/, () => t('activity.d.localAuthDisabled')],
        [/^(?:CSRF_VIOLATION: )?Invalid CSRF token on login$/, () => t('activity.d.csrfInvalid')],
        [/^(?:RATE_LIMIT_HIT: )?Login rate limit exceeded$/, () => t('activity.d.rateLimited')],
        [/^User logged out$/, () => t('activity.d.loggedOut')],
        [/^Setup rejected: invalid setup token$/, () => t('activity.d.setupTokenInvalid')],
        [/^Initial admin setup completed$/, () => t('activity.d.setupDone')],
        [/^Admin 2FA enabled$/, () => t('activity.d.twoFAEnabled')],
        [/^Admin 2FA disabled$/, () => t('activity.d.twoFADisabled')],
        [/^Revoked all other sessions$/, () => t('activity.d.revokedAll')],
        [/^Revoked session (\S+) \((.*)\)$/, m => t('activity.d.revokedOne').replace('{id}', m[1]).replace('{email}', m[2])],
    ];

    function activityDetail(e) {
        const d = e.detail || '';
        const kind = e.kind || '';
        if (kind.startsWith('share.') || kind.startsWith('receive.')) {
            // File events: the detail is a file name, optionally with a size or
            // a marker. Only the marker and the size are ours to translate.
            let m = d.match(/^(.*) \((\d+) bytes\)$/);
            if (m) return m[1] + ' (' + formatSize(Number(m[2])) + ')';
            m = d.match(/^(.*) \(bulk\)$/);
            if (m) return m[1] + ' (' + t('activity.d.bulk') + ')';
            m = d.match(/^(.*) \(zip\)$/);
            if (m) return m[1] + ' (ZIP)';
            return d;
        }
        for (const [re, fmt] of ACTIVITY_DETAIL_RULES) {
            const m = d.match(re);
            if (m) return fmt(m);
        }
        return d;
    }

    function activityRowsHTML(events, { withShare = true } = {}) {
        return events.map(e => `
            <tr>
                <td title="${escapeHtml(e.at)}" style="white-space:nowrap">${escapeHtml(new Date(e.at).toLocaleString(LANG))}</td>
                <td>${escapeHtml(activityKindLabel(e.kind))}</td>
                <td>${activityActor(e)}</td>
                ${withShare ? `<td style="font-family:var(--font-mono);font-size:var(--text-xs)">${escapeHtml(e.share_id || e.link_id || '')}</td>` : ''}
                <td style="font-family:var(--font-mono);font-size:var(--text-xs)" title="${escapeHtml(e.user_agent || '')}">${escapeHtml(e.ip || '')}</td>
                <td style="color:var(--text-muted)">${escapeHtml(activityDetail(e))}</td>
            </tr>
        `).join('');
    }

    function activityTableHTML(events, opts) {
        if (events.length === 0) return `<p style="color:var(--text-muted)">${t('activity.empty')}</p>`;
        const withShare = opts?.withShare !== false;
        return `
            <div style="overflow-x:auto">
                <table class="users-table activity-table">
                    <thead><tr>
                        <th>${t('activity.when')}</th>
                        <th>${t('activity.kind')}</th>
                        <th>${t('activity.who')}</th>
                        ${withShare ? `<th>${t('activity.share')}</th>` : ''}
                        <th>${t('activity.from')}</th>
                        <th>${t('activity.detail')}</th>
                    </tr></thead>
                    <tbody>${activityRowsHTML(events, { withShare })}</tbody>
                </table>
            </div>
        `;
    }

    // The settings card: admin-only on the server, so a 403 simply hides it
    // instead of showing a dead widget to a user or viewer.
    async function loadActivityLog() {
        const card = document.getElementById('activity-card');
        const container = document.getElementById('activity-config');
        if (!card || !container) return;

        const state = { kind: '', offset: 0, limit: 50, events: [], total: 0 };

        async function fetchPage(append) {
            const params = new URLSearchParams({ limit: state.limit, offset: state.offset });
            if (state.kind) params.set('kind', state.kind);
            const res = await api('/api/events?' + params.toString());
            if (res.status === 403) { card.style.display = 'none'; return false; }
            if (!res.ok) throw new Error(await res.text());
            const page = await res.json();
            state.total = page.total;
            state.events = append ? state.events.concat(page.events) : page.events;
            return true;
        }

        function render() {
            const exportParams = state.kind ? '?kind=' + encodeURIComponent(state.kind) : '';
            container.innerHTML = `
                <p style="font-size:var(--text-sm);color:var(--text-muted);margin-bottom:var(--space-3)">${t('settings.activityHint')}</p>
                <div style="display:flex;gap:8px;align-items:center;margin-bottom:var(--space-3);flex-wrap:wrap">
                    <select id="activity-kind" style="flex:0 1 auto">
                        <option value="">${t('activity.allKinds')}</option>
                        ${ACTIVITY_KINDS.map(k => `<option value="${k}" ${k === state.kind ? 'selected' : ''}>${escapeHtml(activityKindLabel(k))}</option>`).join('')}
                    </select>
                    <span style="font-size:var(--text-xs);color:var(--text-muted)">${state.events.length} / ${state.total}</span>
                    <a class="btn btn-ghost btn-sm" href="/api/events/export${exportParams}" style="margin-left:auto">${t('activity.export')}</a>
                </div>
                <div id="activity-table">${activityTableHTML(state.events)}</div>
                ${state.events.length < state.total ? `<button class="btn btn-ghost btn-sm" id="activity-more" style="margin-top:var(--space-3)">${t('activity.loadMore')}</button>` : ''}
            `;
            document.getElementById('activity-kind').onchange = async (ev) => {
                state.kind = ev.target.value;
                state.offset = 0;
                await load(false);
            };
            document.getElementById('activity-more')?.addEventListener('click', async () => {
                state.offset = state.events.length;
                await load(true);
            });
        }

        async function load(append) {
            try {
                if (await fetchPage(append)) render();
            } catch (err) {
                container.innerHTML = `<p style="color:var(--danger)">${t('activity.loadFailed')}: ${escapeHtml(err.message || '')}</p>`;
            }
        }

        await load(false);
    }

    // Per-share activity, opened from the share list. Owner or admin.
    async function showShareActivity(shareId, shareName) {
        showModal(`
            <h3>${t('activity.forShare')} — ${escapeHtml(shareName || shareId)}</h3>
            <div id="share-activity-body" style="margin:var(--space-3) 0"><p style="color:var(--text-muted)">…</p></div>
            <div class="modal-actions">
                <button class="btn btn-ghost" id="modal-close-btn">${t('common.close')}</button>
            </div>
        `);
        document.getElementById('modal-close-btn')?.addEventListener('click', closeModal);
        const body = document.getElementById('share-activity-body');
        try {
            const res = await api(`/api/shares/${encodeURIComponent(shareId)}/events?limit=100`);
            if (!res.ok) throw new Error(await res.text());
            const page = await res.json();
            body.innerHTML = activityTableHTML(page.events, { withShare: false });
        } catch (err) {
            body.innerHTML = `<p style="color:var(--danger)">${t('activity.loadFailed')}: ${escapeHtml(err.message || '')}</p>`;
        }
    }

    async function loadNetworkConfig() {
        const container = document.getElementById('network-config');
        try {
            const res = await api('/api/network');
            if (!res.ok) {
                container.innerHTML = '<p style="color:var(--text-muted)">' + escapeHtml(t('load.networkFailed')) + '</p>';
                return;
            }
            networkConfig = await res.json();
            const nw = networkConfig.networks || {};
            const port = networkConfig.port || '8080';

            const networks = [
                { key: 'local', label: 'Local Network', urlPrefix: 'http://', urlSuffix: ':' + port },
                { key: 'cloudflare', label: 'Cloudflare Tunnel', urlPrefix: '', urlSuffix: '' },
                { key: 'tailscale', label: 'Tailscale Funnel', urlPrefix: '', urlSuffix: '' },
                { key: 'easytier', label: 'EasyTier', urlPrefix: 'http://', urlSuffix: ':' + port },
                // Free-text field for a user's own fixed domain (reverse proxy,
                // own Cloudflare/tunnel domain, WireGuard, …). Never auto-detected.
                { key: 'custom', label: 'Custom Domain / URL', urlPrefix: '', urlSuffix: '', ph: 'https://files.example.com', hint: t('settings.customHint') },
            ];

            const primary = networkConfig.primaryNetwork || 'local';

            container.innerHTML = `
                <div class="network-cards">
                    ${networks.map(n => {
                        const info = nw[n.key] || {};
                        const enabled = !!info.enabled;
                        const url = info.url || '';
                        const detected = info.detected || '';
                        const disabledClass = enabled ? '' : 'disabled';
                        const isPrimary = primary === n.key;
                        return `
                        <div class="network-card ${disabledClass}" data-network-key="${n.key}">
                            <div class="network-card-header">
                                <label class="network-enable-label">
                                    <input type="checkbox" class="network-enable-cb" data-key="${n.key}" ${enabled ? 'checked' : ''}>
                                    <span class="status-dot ${enabled && url ? 'active' : ''}"></span>
                                    <span class="network-label">${escapeHtml(n.label)}</span>
                                </label>
                                <label class="network-primary-label" title="${t('settings.primary')}">
                                    <input type="radio" name="primary-network" value="${n.key}" ${isPrimary ? 'checked' : ''} ${!enabled ? 'disabled' : ''}>
                                    <span class="primary-indicator">${t('settings.primary')}</span>
                                </label>
                            </div>
                            <div class="network-card-body">
                                <input type="text" class="network-url-input" data-key="${n.key}" value="${escapeHtml(url)}" placeholder="${escapeHtml(n.ph || t('settings.notDetected'))}" ${!enabled ? 'disabled' : ''}>
                                ${detected ? `
                                <div class="network-detected-hint">
                                    <span>${t('settings.detected')}: ${escapeHtml(detected)}</span>
                                    ${detected !== url ? `<button type="button" class="btn-use-detected" data-key="${n.key}" data-detected="${escapeHtml(detected)}" ${!enabled ? 'disabled' : ''}>${t('settings.useDetected')}</button>` : ''}
                                </div>` : ''}
                                ${n.hint ? `<div class="network-detected-hint"><span>${escapeHtml(n.hint)}</span></div>` : ''}
                            </div>
                        </div>`;
                    }).join('')}
                </div>
                <div class="form-row" style="margin-top:var(--space-4)">
                    <div class="form-group">
                        <label>${t('settings.maxFileSize')}</label>
                        <input type="number" id="max-file-size-gb" value="${networkConfig.maxFileSizeGB || 10}" min="1" max="100" style="width:120px"> <span style="font-size:var(--text-sm);color:var(--text-muted)">GB</span>
                    </div>
                </div>
                <div style="margin-top:var(--space-4)">
                    <button class="btn btn-primary btn-sm" id="save-network-btn">${t('settings.save')}</button>
                </div>
            `;

            // Checkbox toggle: enable/disable network card
            container.querySelectorAll('.network-enable-cb').forEach(cb => {
                cb.addEventListener('change', () => {
                    const key = cb.dataset.key;
                    const card = container.querySelector(`.network-card[data-network-key="${key}"]`);
                    const urlInput = card.querySelector('.network-url-input');
                    const radioBtn = card.querySelector('input[type="radio"]');
                    const useDetectedBtns = card.querySelectorAll('.btn-use-detected');
                    if (cb.checked) {
                        card.classList.remove('disabled');
                        urlInput.disabled = false;
                        radioBtn.disabled = false;
                        useDetectedBtns.forEach(b => b.disabled = false);
                    } else {
                        card.classList.add('disabled');
                        urlInput.disabled = true;
                        radioBtn.disabled = true;
                        useDetectedBtns.forEach(b => b.disabled = true);
                        // If this was primary, uncheck it
                        if (radioBtn.checked) {
                            radioBtn.checked = false;
                            // Select first enabled network as primary
                            const firstEnabled = container.querySelector('.network-enable-cb:checked');
                            if (firstEnabled) {
                                const fallbackRadio = container.querySelector(`input[type="radio"][value="${firstEnabled.dataset.key}"]`);
                                if (fallbackRadio) fallbackRadio.checked = true;
                            }
                        }
                    }
                });
            });

            // "Use detected" buttons
            container.querySelectorAll('.btn-use-detected').forEach(btn => {
                btn.addEventListener('click', () => {
                    const key = btn.dataset.key;
                    const detected = btn.dataset.detected;
                    const input = container.querySelector(`.network-url-input[data-key="${key}"]`);
                    if (input && detected) input.value = detected;
                });
            });

            // Save button
            document.getElementById('save-network-btn')?.addEventListener('click', async () => {
                const selected = document.querySelector('input[name="primary-network"]:checked')?.value;
                try {
                    const tunnelRes = await api('/api/tunnel');
                    const data = tunnelRes.ok ? await tunnelRes.json() : {};
                    const tunnelConfig = data.config || data;

                    // Collect checkbox states and URL values
                    const cbTrue = true, cbFalse = false;
                    const localCb = container.querySelector('.network-enable-cb[data-key="local"]');
                    const cfCb = container.querySelector('.network-enable-cb[data-key="cloudflare"]');
                    const tsCb = container.querySelector('.network-enable-cb[data-key="tailscale"]');
                    const etCb = container.querySelector('.network-enable-cb[data-key="easytier"]');
                    const cuCb = container.querySelector('.network-enable-cb[data-key="custom"]');

                    tunnelConfig.localEnabled = localCb?.checked ? cbTrue : cbFalse;
                    tunnelConfig.cloudflareEnabled = cfCb?.checked ? cbTrue : cbFalse;
                    tunnelConfig.tailscaleEnabled = tsCb?.checked ? cbTrue : cbFalse;
                    tunnelConfig.easytierEnabled = etCb?.checked ? cbTrue : cbFalse;
                    tunnelConfig.customEnabled = cuCb?.checked ? cbTrue : cbFalse;

                    // Collect URL values
                    tunnelConfig.localIp = container.querySelector('.network-url-input[data-key="local"]')?.value?.trim() || '';
                    tunnelConfig.cloudflareUrl = container.querySelector('.network-url-input[data-key="cloudflare"]')?.value?.trim() || '';
                    tunnelConfig.tailscaleUrl = container.querySelector('.network-url-input[data-key="tailscale"]')?.value?.trim() || '';
                    tunnelConfig.easytierIp = container.querySelector('.network-url-input[data-key="easytier"]')?.value?.trim() || '';
                    tunnelConfig.customUrl = container.querySelector('.network-url-input[data-key="custom"]')?.value?.trim() || '';

                    if (selected) tunnelConfig.primaryNetwork = selected;
                    const maxSize = parseInt(document.getElementById('max-file-size-gb')?.value);
                    if (maxSize > 0 && maxSize <= 100) tunnelConfig.maxFileSizeGB = maxSize;

                    const saveRes = await api('/api/tunnel', {
                        method: 'POST',
                        body: JSON.stringify(tunnelConfig),
                    });
                    if (saveRes.ok) {
                        toast(t('settings.saved'), 'success');
                        const saveData = await saveRes.json().catch(() => ({}));
                        if (saveData.cloudflareRotating) {
                            await waitForCloudflareRotation();
                        }
                        loadNetworkConfig();
                    } else {
                        toast(t('toast.error'), 'error');
                    }
                } catch { toast(t('toast.error'), 'error'); }
            });

        } catch {
            container.innerHTML = '<p style="color:var(--text-muted)">' + escapeHtml(t('load.networkFailed')) + '</p>';
        }
    }

    // After selecting Cloudflare as the primary network, the backend asks the
    // tunnel container to mint a fresh quick-tunnel URL. Poll /api/tunnel until
    // a new URL appears (the wrapper blanks the old one first, then writes the
    // new one), so the UI reflects the rotated link rather than a dead one.
    async function waitForCloudflareRotation() {
        let startUrl = '';
        try {
            const r = await api('/api/tunnel');
            if (r.ok) startUrl = (await r.json()).url || '';
        } catch { /* ignore */ }
        toast(t('settings.cfRotating'), 'info');
        for (let i = 0; i < 30; i++) { // ~60s budget
            await new Promise(res => setTimeout(res, 2000));
            try {
                const r = await api('/api/tunnel');
                if (!r.ok) continue;
                const url = (await r.json()).url || '';
                if (url && url !== startUrl) {
                    toast(`${t('settings.cfRotated')} ${url}`, 'success');
                    return;
                }
            } catch { /* keep polling */ }
        }
        toast(t('settings.cfRotateTimeout'), 'error');
    }

    async function loadFileRestrictionsConfig() {
        const container = document.getElementById('file-restrictions-config');
        if (!container) return;
        try {
            const res = await api('/api/tunnel');
            const config = res.ok ? await res.json() : {};

            const defaultBlocked = '.exe,.bat,.cmd,.com,.msi,.scr,.pif,.vbs,.vbe,.js,.jse,.ws,.wsf,.wsc,.wsh,.ps1,.ps1xml,.ps2,.ps2xml,.psc1,.psc2,.reg,.inf,.lnk,.hta,.cpl,.msc,.jar';

            container.innerHTML = `
                <p style="font-size:var(--text-sm);color:var(--text-muted);margin-bottom:var(--space-4)">${t('settings.fileRestrictionsHint')}</p>
                <div class="form-group">
                    <label>${t('settings.blockedExtensions')}</label>
                    <input type="text" id="blocked-extensions" value="${escapeHtml(config.blockedExtensions || defaultBlocked)}" placeholder=".exe,.bat,.cmd,...">
                </div>
                <div class="form-group">
                    <label>${t('settings.allowedExtensions')}</label>
                    <input type="text" id="allowed-extensions" value="${escapeHtml(config.allowedExtensions || '')}" placeholder="${t('settings.allowedExtensionsHint')}">
                </div>
                <div class="form-actions">
                    <button class="btn btn-primary btn-sm" id="save-file-restrictions-btn">${t('settings.save')}</button>
                </div>
            `;

            document.getElementById('save-file-restrictions-btn').addEventListener('click', async () => {
                try {
                    const tunnelRes = await api('/api/tunnel');
                    const tunnelConfig = tunnelRes.ok ? await tunnelRes.json() : {};
                    tunnelConfig.blockedExtensions = document.getElementById('blocked-extensions').value.trim();
                    tunnelConfig.allowedExtensions = document.getElementById('allowed-extensions').value.trim();

                    const saveRes = await api('/api/tunnel', {
                        method: 'POST',
                        body: JSON.stringify(tunnelConfig),
                    });
                    if (saveRes.ok) toast(t('settings.saved'), 'success');
                    else toast(t('toast.error'), 'error');
                } catch { toast(t('toast.error'), 'error'); }
            });
        } catch {
            container.innerHTML = '<p style="color:var(--text-muted)">' + escapeHtml(t('load.restrictionsFailed')) + '</p>';
        }
    }

    async function loadWebhookConfig() {
        const container = document.getElementById('webhook-config');
        try {
            const res = await api('/api/webhook');
            const config = res.ok ? await res.json() : {};

            // The secret is never sent back by the API (only whether one is
            // stored), so the field starts empty by design. Leaving it empty on
            // save must therefore KEEP the stored secret, not erase it — hence
            // the field is only included in the body when the admin typed
            // something, or ticked "clear".
            const secretSet = !!config.secret_set;

            container.innerHTML = `
                <label class="toggle-label" style="margin-bottom:var(--space-3)">
                    <input type="checkbox" id="webhook-enabled" ${config.enabled ? 'checked' : ''}>
                    <span class="toggle-switch"></span>
                    <span>${t('settings.webhookEnabled')}</span>
                </label>
                <div class="form-group">
                    <label>${t('settings.webhookUrl')}</label>
                    <input type="url" id="webhook-url" value="${escapeHtml(config.url || '')}" placeholder="https://...">
                </div>
                <div class="form-group">
                    <label>${t('settings.webhookEvents')}</label>
                    <label class="checkbox-label">
                        <input type="checkbox" id="webhook-on-download" ${config.on_download ? 'checked' : ''}>
                        <span>${t('settings.webhookOnDownload')}</span>
                    </label>
                    <label class="checkbox-label">
                        <input type="checkbox" id="webhook-on-limit" ${config.on_limit_reached ? 'checked' : ''}>
                        <span>${t('settings.webhookOnLimit')}</span>
                    </label>
                    <label class="checkbox-label">
                        <input type="checkbox" id="webhook-on-expire" ${config.on_expire ? 'checked' : ''}>
                        <span>${t('settings.webhookOnExpire')}</span>
                    </label>
                </div>
                <div class="form-group">
                    <label>${t('settings.webhookSecret')}</label>
                    <input type="password" id="webhook-secret" value="" autocomplete="new-password"
                           placeholder="${secretSet ? t('settings.webhookSecretKeep') : t('settings.webhookSecretNone')}">
                    ${secretSet ? `
                    <label class="checkbox-label">
                        <input type="checkbox" id="webhook-secret-clear">
                        <span>${t('settings.webhookSecretClear')}</span>
                    </label>` : ''}
                </div>
                <div class="form-actions">
                    <button class="btn btn-primary btn-sm" id="save-webhook-btn">${t('settings.save')}</button>
                    <button class="btn btn-ghost btn-sm" id="test-webhook-btn">${t('settings.testWebhook')}</button>
                </div>
            `;

            document.getElementById('save-webhook-btn').addEventListener('click', async () => {
                const url = document.getElementById('webhook-url').value.trim();
                const enabled = document.getElementById('webhook-enabled').checked;
                if (enabled && !url) {
                    toast(t('settings.webhookNeedsUrl'), 'error');
                    return;
                }
                const body = {
                    enabled,
                    url,
                    on_download: document.getElementById('webhook-on-download').checked,
                    on_limit_reached: document.getElementById('webhook-on-limit').checked,
                    on_expire: document.getElementById('webhook-on-expire').checked,
                };
                // Omitted `secret` means "keep what is stored" server-side.
                const typedSecret = document.getElementById('webhook-secret').value;
                const clearSecret = document.getElementById('webhook-secret-clear')?.checked;
                if (clearSecret) body.secret = '';
                else if (typedSecret) body.secret = typedSecret;
                try {
                    const saveRes = await api('/api/webhook', {
                        method: 'POST',
                        body: JSON.stringify(body),
                    });
                    if (saveRes.ok) {
                        toast(t('settings.saved'), 'success');
                        loadWebhookConfig();
                    } else {
                        toast(await saveRes.text() || t('toast.error'), 'error');
                    }
                } catch { toast(t('toast.error'), 'error'); }
            });

            document.getElementById('test-webhook-btn').addEventListener('click', async () => {
                try {
                    const testRes = await api('/api/webhook/test', { method: 'POST' });
                    if (testRes.ok) toast(t('settings.webhookSent'), 'success');
                    else toast(await testRes.text() || t('toast.error'), 'error');
                } catch { toast(t('toast.error'), 'error'); }
            });
        } catch {
            container.innerHTML = '<p style="color:var(--text-muted)">' + escapeHtml(t('load.webhookFailed')) + '</p>';
        }
    }

    async function loadUserManagement() {
        const container = document.getElementById('user-management');
        try {
            const res = await api('/api/users');
            if (!res.ok) {
                container.innerHTML = '<p style="color:var(--text-muted)">' + escapeHtml(t('load.usersUnavailable')) + '</p>';
                return;
            }
            const users = await res.json();

            container.innerHTML = `
                <table class="users-table">
                    <thead>
                        <tr>
                            <th>${t('settings.name')}</th>
                            <th>${t('settings.email')}</th>
                            <th>${t('settings.role')}</th>
                            <th>${t('settings.quota')}</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        ${(users || []).map(u => `
                            <tr>
                                <td>${escapeHtml(u.name || '-')}</td>
                                <td>${escapeHtml(u.email || '-')}</td>
                                <td><span class="role-badge role-${(u.role || 'viewer').toLowerCase()}">${escapeHtml(roleLabel(u.role || 'viewer'))}</span></td>
                                <td style="white-space:nowrap">
                                    <span style="color:var(--text-muted);font-size:var(--text-sm)">${formatSize(u.usageBytes || 0)}</span>
                                    <span style="color:var(--text-muted)"> / </span>
                                    <input type="number" min="0" step="1" class="user-quota-input"
                                           value="${u.quotaBytes ? Math.round(u.quotaBytes / 1073741824) : 0}"
                                           data-user-quota="${escapeHtml(u.id)}"
                                           title="GB, 0 = ${t('settings.unlimited')}"
                                           style="width:64px;padding:2px 6px"> GB
                                </td>
                                <td>
                                    <button class="btn-icon danger" data-delete-user="${escapeHtml(u.id)}" title="${t('common.delete')}">
                                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>
                                    </button>
                                </td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
                <div style="margin-top:var(--space-4);border-top:1px solid var(--border);padding-top:var(--space-4)">
                    <h4 style="font-size:var(--text-base);font-weight:var(--weight-semibold, 600);margin-bottom:var(--space-3)">${t('settings.createUser')}</h4>
                    <div class="form-row">
                        <div class="form-group">
                            <label>${t('settings.email')}</label>
                            <input type="email" id="new-user-email" required>
                        </div>
                        <div class="form-group">
                            <label>${t('settings.name')}</label>
                            <input type="text" id="new-user-name" required>
                        </div>
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label>${t('settings.role')}</label>
                            <select id="new-user-role">
                                <option value="admin">${t('role.admin')}</option>
                                <option value="user" selected>${t('role.user')}</option>
                                <option value="viewer">${t('role.viewer')}</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label>${t('settings.userPassword')}</label>
                            <input type="password" id="new-user-password" autocomplete="new-password">
                            <small style="color:var(--text-muted);display:block;margin-top:4px">${t('settings.userPasswordHint')}</small>
                        </div>
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label>${t('settings.quotaGB')}</label>
                            <input type="number" id="new-user-quota" value="0" min="0" step="1">
                        </div>
                    </div>
                    <button class="btn btn-primary btn-sm" id="create-user-btn">${t('settings.createUser')}</button>
                </div>
            `;

            container.querySelectorAll('[data-delete-user]').forEach(btn => {
                btn.addEventListener('click', async () => {
                    if (!confirm(t('settings.deleteUserConfirm'))) return;
                    try {
                        const delRes = await api(`/api/users/${btn.dataset.deleteUser}`, { method: 'DELETE' });
                        if (delRes.ok) {
                            toast(t('settings.userDeleted'), 'success');
                            loadUserManagement();
                        } else toast(t('toast.error'), 'error');
                    } catch { toast(t('toast.error'), 'error'); }
                });
            });

            // Inline quota edit: PUT the new limit when an admin changes the GB field.
            container.querySelectorAll('.user-quota-input').forEach(input => {
                input.addEventListener('change', async () => {
                    const gb = Math.max(0, parseInt(input.value || '0', 10));
                    input.value = gb;
                    try {
                        const putRes = await api(`/api/users/${input.dataset.userQuota}`, {
                            method: 'PUT',
                            body: JSON.stringify({ quotaBytes: gb * 1073741824 }),
                        });
                        if (putRes.ok) toast(t('settings.quotaUpdated'), 'success');
                        else toast((await putRes.text()) || t('toast.error'), 'error');
                    } catch { toast(t('toast.error'), 'error'); }
                });
            });

            document.getElementById('create-user-btn')?.addEventListener('click', async () => {
                const email = document.getElementById('new-user-email').value;
                const name = document.getElementById('new-user-name').value;
                const role = document.getElementById('new-user-role').value;
                const password = document.getElementById('new-user-password').value;
                const quotaGB = Math.max(0, parseInt(document.getElementById('new-user-quota')?.value || '0', 10));

                if (!email || !name) {
                    toast(t('settings.fillAll'), 'warning');
                    return;
                }
                // Password is optional (SSO-only accounts have none); the backend
                // enforces min. 8 chars only when one is set — mirror that here.
                if (password && password.length < 8) {
                    toast(t('settings.passwordTooShort'), 'warning');
                    return;
                }

                try {
                    const createRes = await api('/api/users', {
                        method: 'POST',
                        body: JSON.stringify({ email, name, role, password, quotaBytes: quotaGB * 1073741824 }),
                    });
                    if (createRes.ok) {
                        toast(t('settings.userCreated'), 'success');
                        loadUserManagement();
                    } else {
                        const err = await createRes.text();
                        toast(err || t('toast.error'), 'error');
                    }
                } catch { toast(t('toast.error'), 'error'); }
            });
        } catch {
            container.innerHTML = '<p style="color:var(--text-muted)">' + escapeHtml(t('load.usersUnavailable')) + '</p>';
        }
    }

    // ==========================================
    // API Keys Management
    // ==========================================
    // Two-Factor Authentication (TOTP)
    // ==========================================
    async function load2FAConfig() {
        const container = document.getElementById('twofa-config');
        if (!container) return;
        try {
            const res = await api('/api/admin/2fa');
            const data = res.ok ? await res.json() : {};
            render2FA(container, !!data.enabled);
        } catch {
            container.innerHTML = '<p style="color:var(--text-muted)">' + escapeHtml(t('load.twoFAFailed')) + '</p>';
        }
    }

    function render2FA(container, enabled) {
        const statusLabel = enabled ? t('settings.twofaEnabled') : t('settings.twofaDisabled');
        const statusClass = enabled ? 'active' : '';
        let html = `
            <p style="font-size:var(--text-sm);color:var(--text-muted);margin-bottom:var(--space-3)">${t('settings.twofaHint')}</p>
            <div class="twofa-status" style="display:flex;align-items:center;gap:8px;margin-bottom:var(--space-4)">
                <span class="status-dot ${statusClass}"></span>
                <span style="color:var(--text-muted)">${t('settings.twofaStatus')}:</span>
                <strong>${statusLabel}</strong>
            </div>`;

        if (enabled) {
            html += `
                <div class="form-group">
                    <label>${t('settings.twofaCode')}</label>
                    <input type="text" id="twofa-disable-code" inputmode="numeric" autocomplete="off" maxlength="6" placeholder="000000" style="max-width:160px;font-family:var(--font-mono);letter-spacing:2px">
                </div>
                <div class="form-actions">
                    <button class="btn btn-danger btn-sm" id="twofa-disable-btn">${t('settings.twofaDisable')}</button>
                </div>`;
        } else {
            html += `
                <div class="form-actions">
                    <button class="btn btn-primary btn-sm" id="twofa-enable-btn">${t('settings.twofaEnable')}</button>
                </div>
                <div id="twofa-setup" style="display:none;margin-top:var(--space-4)"></div>`;
        }

        container.innerHTML = html;

        if (enabled) {
            document.getElementById('twofa-disable-btn').addEventListener('click', async () => {
                const code = (document.getElementById('twofa-disable-code').value || '').trim();
                if (!code) { toast(t('settings.twofaCodeRequired'), 'error'); return; }
                const r = await api('/api/admin/2fa/disable', {
                    method: 'POST',
                    body: JSON.stringify({ code }),
                });
                if (r.ok) {
                    toast(t('settings.twofaDisabledMsg'), 'success');
                    render2FA(container, false);
                } else {
                    toast((await r.text()).trim() || t('toast.error'), 'error');
                }
            });
        } else {
            document.getElementById('twofa-enable-btn').addEventListener('click', () => start2FASetup(container));
        }
    }

    async function start2FASetup(container) {
        const setupEl = document.getElementById('twofa-setup');
        if (!setupEl) return;
        let setup;
        try {
            const res = await api('/api/admin/2fa/setup');
            if (!res.ok) { toast(t('settings.twofaSetupFailed'), 'error'); return; }
            setup = await res.json();
        } catch {
            toast(t('settings.twofaSetupFailed'), 'error');
            return;
        }

        setupEl.style.display = 'block';
        setupEl.innerHTML = `
            <p style="font-size:var(--text-sm);margin-bottom:var(--space-2)">${t('settings.twofaScan')}</p>
            <img src="${escapeHtml(setup.qr || '')}" alt="2FA QR code" class="twofa-qr" width="200" height="200" style="background:#fff;padding:8px;border-radius:var(--radius);display:block">
            <p style="font-size:var(--text-sm);margin-top:var(--space-3);margin-bottom:var(--space-2)">${t('settings.twofaManual')}</p>
            <code class="twofa-secret" style="display:inline-block;user-select:all;font-family:var(--font-mono);font-size:var(--text-sm);word-break:break-all;background:var(--bg-elevated, rgba(127,127,127,0.1));padding:6px 10px;border-radius:var(--radius);color:var(--text-primary)">${escapeHtml(setup.secret || '')}</code>
            <div class="form-group" style="margin-top:var(--space-4)">
                <label>${t('settings.twofaCode')}</label>
                <input type="text" id="twofa-enable-code" inputmode="numeric" autocomplete="off" maxlength="6" placeholder="000000" style="max-width:160px;font-family:var(--font-mono);letter-spacing:2px">
            </div>
            <div class="form-actions">
                <button class="btn btn-primary btn-sm" id="twofa-verify-btn">${t('settings.twofaVerifyEnable')}</button>
                <button class="btn btn-ghost btn-sm" id="twofa-cancel-btn">${t('settings.twofaCancel')}</button>
            </div>`;

        document.getElementById('twofa-cancel-btn').addEventListener('click', () => {
            setupEl.style.display = 'none';
            setupEl.innerHTML = '';
        });

        document.getElementById('twofa-verify-btn').addEventListener('click', async () => {
            const code = (document.getElementById('twofa-enable-code').value || '').trim();
            if (!code) { toast(t('settings.twofaCodeRequired'), 'error'); return; }
            const r = await api('/api/admin/2fa/enable', {
                method: 'POST',
                body: JSON.stringify({ secret: setup.secret, code }),
            });
            if (r.ok) {
                toast(t('settings.twofaEnabledMsg'), 'success');
                render2FA(container, true);
            } else {
                toast((await r.text()).trim() || t('toast.error'), 'error');
            }
        });
    }

    // ==========================================
    async function loadAPIKeys() {
        const container = document.getElementById('apikeys-config');
        if (!container) return;
        try {
            const res = await api('/api/api-keys');
            const keys = res.ok ? await res.json() : [];

            container.innerHTML = `
                <p style="font-size:var(--text-sm);color:var(--text-muted);margin-bottom:var(--space-3)">${t('settings.apiKeysHint')}</p>
                <div id="api-keys-list">
                    ${keys.length === 0 ? `<p style="color:var(--text-muted)">${t('settings.noApiKeys')}</p>` :
                    keys.map(k => `
                        <div class="file-item" style="margin-bottom:4px">
                            <div class="file-item-info">
                                <div><strong>${escapeHtml(k.name)}</strong></div>
                                <div style="font-size:var(--text-xs);color:var(--text-muted);font-family:var(--font-mono)">${escapeHtml(k.prefix)}</div>
                            </div>
                            <span class="role-badge role-${k.role}" style="margin-right:8px">${escapeHtml(roleLabel(k.role))}</span>
                            <button class="btn-icon danger delete-apikey-btn" data-id="${escapeHtml(k.id)}" title="${t('common.delete')}">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>
                            </button>
                        </div>
                    `).join('')}
                </div>
                <div style="margin-top:var(--space-4);display:flex;gap:8px;align-items:end">
                    <div class="form-group" style="margin:0;flex:1">
                        <label>${t('settings.name')}</label>
                        <input type="text" id="apikey-name" placeholder="${escapeHtml(t('settings.apiKeyNamePlaceholder'))}">
                    </div>
                    <button class="btn btn-primary btn-sm" id="create-apikey-btn">${t('settings.createApiKey')}</button>
                </div>
                <div id="new-key-display" style="display:none;margin-top:var(--space-3)"></div>
            `;

            // Delete handlers
            container.querySelectorAll('.delete-apikey-btn').forEach(btn => {
                btn.onclick = async () => {
                    if (!confirm(t('settings.deleteApiKeyConfirm'))) return;
                    const delRes = await api('/api/api-keys/' + btn.dataset.id, { method: 'DELETE' });
                    if (delRes.ok) { toast(t('settings.apiKeyDeleted'), 'success'); loadAPIKeys(); }
                    else toast(t('toast.error'), 'error');
                };
            });

            // Create handler
            document.getElementById('create-apikey-btn').onclick = async () => {
                const name = document.getElementById('apikey-name').value || 'API Key';
                const res = await api('/api/api-keys', {
                    method: 'POST',
                    body: JSON.stringify({ name, role: 'admin' }),
                });
                if (res.ok) {
                    const data = await res.json();
                    document.getElementById('new-key-display').style.display = 'block';
                    document.getElementById('new-key-display').innerHTML = `
                        <div style="background:var(--success-bg, rgba(34,197,94,0.1));border:1px solid var(--success-border, rgba(34,197,94,0.3));border-radius:var(--radius);padding:12px">
                            <p style="font-size:var(--text-sm);font-weight:600;color:var(--success, #22c55e);margin-bottom:8px">${t('settings.apiKeyCreated')}</p>
                            <code style="font-family:var(--font-mono);font-size:var(--text-sm);word-break:break-all;color:var(--text-primary)">${escapeHtml(data.key)}</code>
                            <p style="font-size:var(--text-xs);color:var(--text-muted);margin-top:8px">${t('settings.apiKeyOnlyOnce')}</p>
                            <button class="btn btn-ghost btn-sm" style="margin-top:8px" id="copy-new-apikey-btn">${t('common.copy')}</button>
                        </div>
                    `;
                    document.getElementById('copy-new-apikey-btn').onclick = () => {
                        navigator.clipboard.writeText(data.key);
                        document.getElementById('copy-new-apikey-btn').textContent = t('common.copied');
                    };
                    loadAPIKeys();
                } else {
                    toast(t('toast.error'), 'error');
                }
            };
        } catch {
            container.innerHTML = '<p style="color:var(--text-muted)">' + escapeHtml(t('load.apiKeysFailed')) + '</p>';
        }
    }

    // ==========================================
    // SMTP Configuration
    // ==========================================
    async function loadSMTPConfig() {
        const container = document.getElementById('smtp-config');
        if (!container) return;
        try {
            const res = await api('/api/smtp');
            const config = res.ok ? await res.json() : {};

            // Which encryption the saved config uses. A fresh config (no flags)
            // defaults to STARTTLS, the port-587 case most providers expect.
            const enc = config.use_tls ? 'ssltls' : (config.use_starttls === false ? 'none' : 'starttls');

            container.innerHTML = `
                <div class="form-row">
                    <div class="form-group"><label>${t('smtp.host')}</label><input type="text" id="smtp-host" value="${escapeHtml(config.host || '')}" placeholder="smtp.gmail.com"></div>
                    <div class="form-group"><label>${t('smtp.port')}</label><input type="number" id="smtp-port" value="${config.port || 587}" style="width:100px"></div>
                    <div class="form-group"><label>${t('smtp.encryption')}</label>
                        <select id="smtp-encryption">
                            <option value="starttls" ${enc === 'starttls' ? 'selected' : ''}>STARTTLS (587)</option>
                            <option value="ssltls" ${enc === 'ssltls' ? 'selected' : ''}>SSL/TLS (465)</option>
                            <option value="none" ${enc === 'none' ? 'selected' : ''}>${t('smtp.encryptionNone')}</option>
                        </select>
                    </div>
                </div>
                <div class="form-row">
                    <div class="form-group"><label>${t('smtp.username')}</label><input type="text" id="smtp-user" value="${escapeHtml(config.username || '')}"></div>
                    <div class="form-group"><label>${t('smtp.password')}</label><input type="password" id="smtp-pass" value="${escapeHtml(config.password || '')}"></div>
                </div>
                <div class="form-row">
                    <div class="form-group"><label>${t('smtp.fromEmail')}</label><input type="email" id="smtp-from" value="${escapeHtml(config.from_email || '')}"></div>
                    <div class="form-group"><label>${t('smtp.fromName')}</label><input type="text" id="smtp-from-name" value="${escapeHtml(config.from_name || 'CasaDrop')}"></div>
                </div>
                <label class="toggle-label" style="margin-bottom:var(--space-3)">
                    <input type="checkbox" id="smtp-enabled" ${config.enabled ? 'checked' : ''}>
                    <span class="toggle-switch"></span>
                    <span>${t('settings.smtpEnabled')}</span>
                </label>
                <div class="form-actions">
                    <button class="btn btn-primary btn-sm" id="save-smtp-btn">${t('settings.save')}</button>
                    <button class="btn btn-ghost btn-sm" id="test-smtp-btn">${t('settings.testSmtp')}</button>
                </div>
            `;

            // Switching encryption fills in the conventional port, but only when
            // the current port is a standard one — a custom port is left alone.
            document.getElementById('smtp-encryption').onchange = (e) => {
                const portEl = document.getElementById('smtp-port');
                const cur = parseInt(portEl.value);
                if (!cur || cur === 25 || cur === 465 || cur === 587 || cur === 2525) {
                    if (e.target.value === 'ssltls') portEl.value = 465;
                    else if (e.target.value === 'starttls') portEl.value = 587;
                }
            };

            document.getElementById('save-smtp-btn').onclick = async () => {
                const encryption = document.getElementById('smtp-encryption').value;
                const body = {
                    enabled: document.getElementById('smtp-enabled').checked,
                    host: document.getElementById('smtp-host').value,
                    port: parseInt(document.getElementById('smtp-port').value) || 587,
                    username: document.getElementById('smtp-user').value,
                    password: document.getElementById('smtp-pass').value,
                    from_email: document.getElementById('smtp-from').value,
                    from_name: document.getElementById('smtp-from-name').value,
                    use_tls: encryption === 'ssltls',       // implicit TLS, port 465
                    use_starttls: encryption === 'starttls', // STARTTLS, port 587
                };
                const r = await api('/api/smtp', { method: 'POST', body: JSON.stringify(body) });
                toast(r.ok ? t('settings.saved') : t('toast.error'), r.ok ? 'success' : 'error');
            };

            document.getElementById('test-smtp-btn').onclick = async () => {
                const r = await api('/api/smtp/test', { method: 'POST' });
                toast(r.ok ? t('settings.smtpTestOk') : (await r.text() || t('toast.error')), r.ok ? 'success' : 'error');
            };
        } catch {
            container.innerHTML = '<p style="color:var(--text-muted)">' + escapeHtml(t('load.smtpFailed')) + '</p>';
        }
    }

    // ==========================================
    // Mobile Support
    // ==========================================
    function initMobile() {
        // Create mobile header (always, CSS hides it on desktop)
        const header = document.createElement('div');
        header.className = 'mobile-header';
        header.innerHTML = `
            <button class="mobile-menu-btn" id="mobile-menu-btn">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
            </button>
            <span class="brand-name">CasaDrop</span>
        `;
        document.body.prepend(header);

        document.getElementById('mobile-menu-btn').addEventListener('click', () => {
            const sidebar = document.getElementById('sidebar');
            const isOpen = sidebar.classList.toggle('open');

            document.querySelectorAll('.mobile-overlay').forEach(e => e.remove());

            if (isOpen) {
                const overlay = document.createElement('div');
                overlay.className = 'mobile-overlay';
                overlay.addEventListener('click', () => {
                    sidebar.classList.remove('open');
                    overlay.remove();
                });
                document.body.appendChild(overlay);
            }
        });
    }

    // ==========================================
    // Init
    // ==========================================
    function init() {
        applyI18n();
        initMobile();
        initUpload();
        initReceive();

        // Navigation
        document.querySelectorAll('.nav-btn[data-view]').forEach(btn => {
            btn.addEventListener('click', () => showView(btn.dataset.view));
        });

        // Login form
        document.getElementById('login-form').addEventListener('submit', async (e) => {
            e.preventDefault();
            const errEl = document.getElementById('login-error');
            errEl.style.display = 'none';

            const password = document.getElementById('login-password').value;
            try {
                await doLogin(password);
            } catch (err) {
                errEl.textContent = err.message;
                errEl.style.display = '';
            }
        });

        // SSO button
        document.getElementById('sso-btn')?.addEventListener('click', () => {
            window.location.href = '/auth/oidc/login';
        });

        // Logout
        document.getElementById('logout-btn').addEventListener('click', doLogout);

        // Refresh shares
        document.getElementById('refresh-shares').addEventListener('click', loadShares);

        // Host share browser: refresh + delegated row actions
        document.getElementById('hostshare-refresh')?.addEventListener('click', () => loadHostBrowser(currentHostPath));
        const hostshareView = document.getElementById('hostshare-view');
        hostshareView?.addEventListener('click', (e) => {
            const openBtn = e.target.closest('.host-open');
            if (openBtn) { loadHostBrowser(openBtn.dataset.path); return; }
            const shareBtn = e.target.closest('.host-share-btn');
            if (shareBtn) { openHostShareDialog(shareBtn.dataset); return; }
        });

        // Start auth check
        checkAuth();
    }

    // Wait for DOM
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

})();
