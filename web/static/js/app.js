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
            'nav.upload': 'Upload',
            'nav.shares': 'Shares',
            'nav.receive': 'Receive',
            'nav.settings': 'Settings',
            'nav.logout': 'Logout',
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
            'upload.maxDownloads': 'Max downloads (0 = unlimited)',
            'upload.submit': 'Upload',
            'upload.uploading': 'Uploading...',
            'upload.complete': 'Upload complete',
            'upload.error': 'Upload failed',
            'upload.another': 'Upload More',
            'shares.title': 'Active Shares',
            'shares.empty': 'No active shares yet',
            'shares.downloads': 'downloads',
            'shares.expires': 'expires',
            'shares.expired': 'expired',
            'shares.never': 'never',
            'shares.unlimited': 'unlimited',
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
            'settings.enabled': 'Enabled',
            'settings.useDetected': 'Use detected',
            'settings.save': 'Save',
            'settings.saved': 'Settings saved',
            'settings.testWebhook': 'Test Webhook',
            'settings.webhookUrl': 'Webhook URL',
            'settings.webhookSecret': 'Webhook Secret',
            'settings.webhookSent': 'Webhook sent',
            'settings.createUser': 'Create User',
            'settings.email': 'Email',
            'settings.name': 'Name',
            'settings.role': 'Role',
            'settings.userPassword': 'Password',
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
            'shares.selectAll': 'Select All',
            'shares.bulkDelete': 'Delete Selected',
            'shares.selected': 'selected',
            'email.recipientEmail': 'Recipient Email',
            'email.recipientName': 'Recipient Name',
            'email.message': 'Message',
            'email.optional': 'Optional',
            'email.send': 'Send',
            'email.sent': 'Email sent successfully',
            'email.enterRecipient': 'Enter recipient email',
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
        },
        de: {
            'nav.upload': 'Hochladen',
            'nav.shares': 'Freigaben',
            'nav.receive': 'Empfangen',
            'nav.settings': 'Einstellungen',
            'nav.logout': 'Abmelden',
            'login.subtitle': 'Selbst-gehostetes Filesharing',
            'login.password': 'Passwort',
            'login.submit': 'Anmelden',
            'login.or': 'oder',
            'login.sso': 'Mit SSO anmelden',
            'upload.title': 'Dateien hochladen',
            'upload.dropText': 'Dateien hierher ziehen oder klicken',
            'upload.hint': 'Mehrere Dateien moeglich',
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
            'upload.maxDownloads': 'Max Downloads (0 = unbegrenzt)',
            'upload.submit': 'Hochladen',
            'upload.uploading': 'Wird hochgeladen...',
            'upload.complete': 'Upload abgeschlossen',
            'upload.error': 'Upload fehlgeschlagen',
            'upload.another': 'Weitere hochladen',
            'shares.title': 'Aktive Freigaben',
            'shares.empty': 'Noch keine Freigaben',
            'shares.downloads': 'Downloads',
            'shares.expires': 'laeuft ab',
            'shares.expired': 'abgelaufen',
            'shares.never': 'nie',
            'shares.unlimited': 'unbegrenzt',
            'shares.deleteConfirm': 'Freigabe loeschen?',
            'shares.copied': 'Link kopiert!',
            'shares.size': 'Groesse',
            'shares.deleted': 'Freigabe geloescht',
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
            'receive.maxFileSize': 'Max Dateigroesse (MB, 0 = unbegrenzt)',
            'receive.expiry': 'Ablauf in (Stunden, 0 = nie)',
            'receive.extensions': 'Erlaubte Endungen (z.B. .pdf,.doc)',
            'receive.autoShare': 'Hochgeladene Dateien automatisch teilen',
            'receive.submit': 'Erstellen',
            'receive.cancel': 'Abbrechen',
            'receive.empty': 'Noch keine Empfangslinks',
            'receive.uploads': 'Uploads',
            'receive.deleteConfirm': 'Empfangslink loeschen?',
            'receive.created': 'Empfangslink erstellt',
            'receive.deleted': 'Empfangslink geloescht',
            'settings.title': 'Einstellungen',
            'settings.network': 'Netzwerkkonfiguration',
            'settings.webhook': 'Webhook',
            'settings.users': 'Benutzerverwaltung',
            'settings.primary': 'Primaer',
            'settings.detected': 'Erkannt',
            'settings.notDetected': 'Nicht erkannt',
            'settings.enabled': 'Aktiviert',
            'settings.useDetected': 'Erkannten Wert verwenden',
            'settings.save': 'Speichern',
            'settings.saved': 'Einstellungen gespeichert',
            'settings.testWebhook': 'Webhook testen',
            'settings.webhookUrl': 'Webhook URL',
            'settings.webhookSecret': 'Webhook Secret',
            'settings.webhookSent': 'Webhook gesendet',
            'settings.createUser': 'Benutzer erstellen',
            'settings.email': 'E-Mail',
            'settings.name': 'Name',
            'settings.role': 'Rolle',
            'settings.userPassword': 'Passwort',
            'settings.deleteUserConfirm': 'Benutzer loeschen?',
            'settings.userCreated': 'Benutzer erstellt',
            'settings.userDeleted': 'Benutzer geloescht',
            'settings.fillAll': 'Alle Felder ausfuellen',
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
            'stat.totalSize': 'Gesamtgroesse',
            'stat.protected': 'Geschuetzt',
            'shares.sendEmail': 'Per E-Mail senden',
            'shares.selectAll': 'Alle auswählen',
            'shares.bulkDelete': 'Ausgewählte löschen',
            'shares.selected': 'ausgewählt',
            'email.recipientEmail': 'Empfänger E-Mail',
            'email.recipientName': 'Empfängername',
            'email.message': 'Nachricht',
            'email.optional': 'Optional',
            'email.send': 'Senden',
            'email.sent': 'E-Mail erfolgreich gesendet',
            'email.enterRecipient': 'Empfänger E-Mail eingeben',
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
        },
        fr: {
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
            'upload.another': 'Envoyer d\'autres fichiers',
            'shares.title': 'Partages actifs',
            'shares.empty': 'Aucun partage actif',
            'shares.downloads': 'téléchargements',
            'shares.expires': 'expire',
            'shares.expired': 'expiré',
            'shares.never': 'jamais',
            'shares.unlimited': 'illimité',
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
        },
        es: {
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
            'upload.another': 'Subir más',
            'shares.title': 'Compartidos activos',
            'shares.empty': 'Aún no hay compartidos',
            'shares.downloads': 'descargas',
            'shares.expires': 'expira',
            'shares.expired': 'expirado',
            'shares.never': 'nunca',
            'shares.unlimited': 'ilimitado',
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
        },
        it: {
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
            'upload.another': 'Carica altri file',
            'shares.title': 'Condivisioni attive',
            'shares.empty': 'Nessuna condivisione attiva',
            'shares.downloads': 'download',
            'shares.expires': 'scade',
            'shares.expired': 'scaduto',
            'shares.never': 'mai',
            'shares.unlimited': 'illimitato',
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
        },
        pt: {
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
            'upload.another': 'Enviar mais',
            'shares.title': 'Compartilhamentos ativos',
            'shares.empty': 'Nenhum compartilhamento ativo',
            'shares.downloads': 'downloads',
            'shares.expires': 'expira',
            'shares.expired': 'expirado',
            'shares.never': 'nunca',
            'shares.unlimited': 'ilimitado',
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
        },
        nl: {
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
            'upload.another': 'Meer uploaden',
            'shares.title': 'Actieve delingen',
            'shares.empty': 'Nog geen delingen',
            'shares.downloads': 'downloads',
            'shares.expires': 'verloopt',
            'shares.expired': 'verlopen',
            'shares.never': 'nooit',
            'shares.unlimited': 'onbeperkt',
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
        },
        pl: {
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
            'upload.another': 'Wyślij więcej',
            'shares.title': 'Aktywne udostępnienia',
            'shares.empty': 'Brak aktywnych udostępnień',
            'shares.downloads': 'pobrania',
            'shares.expires': 'wygasa',
            'shares.expired': 'wygasło',
            'shares.never': 'nigdy',
            'shares.unlimited': 'bez limitu',
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
        },
        ru: {
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
            'upload.another': 'Загрузить ещё',
            'shares.title': 'Активные ссылки',
            'shares.empty': 'Пока нет общих файлов',
            'shares.downloads': 'скачивания',
            'shares.expires': 'истекает',
            'shares.expired': 'истёк',
            'shares.never': 'никогда',
            'shares.unlimited': 'без ограничений',
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
        },
        ja: {
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
            'upload.another': '追加アップロード',
            'shares.title': '有効な共有',
            'shares.empty': '共有はまだありません',
            'shares.downloads': 'ダウンロード',
            'shares.expires': '期限',
            'shares.expired': '期限切れ',
            'shares.never': 'なし',
            'shares.unlimited': '無制限',
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
        },
        zh: {
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
            'upload.another': '继续上传',
            'shares.title': '活跃共享',
            'shares.empty': '暂无共享',
            'shares.downloads': '次下载',
            'shares.expires': '过期',
            'shares.expired': '已过期',
            'shares.never': '永不',
            'shares.unlimited': '不限',
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
        },
        ko: {
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
            'upload.another': '추가 업로드',
            'shares.title': '활성 공유',
            'shares.empty': '공유가 아직 없습니다',
            'shares.downloads': '다운로드',
            'shares.expires': '만료',
            'shares.expired': '만료됨',
            'shares.never': '없음',
            'shares.unlimited': '무제한',
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
        },
        tr: {
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
            'upload.another': 'Daha fazla yükle',
            'shares.title': 'Aktif Paylaşımlar',
            'shares.empty': 'Henüz paylaşım yok',
            'shares.downloads': 'indirme',
            'shares.expires': 'sona erer',
            'shares.expired': 'sona erdi',
            'shares.never': 'asla',
            'shares.unlimited': 'sınırsız',
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
        },
        // NOTE: Arabic (ar) is an RTL language. Full RTL layout support (CSS direction: rtl) is not yet implemented.
        // The translations below are correct but the UI will display LTR until RTL CSS support is added.
        ar: {
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
            'upload.another': 'رفع المزيد',
            'shares.title': 'المشاركات النشطة',
            'shares.empty': 'لا توجد مشاركات بعد',
            'shares.downloads': 'تنزيلات',
            'shares.expires': 'تنتهي',
            'shares.expired': 'منتهية',
            'shares.never': 'أبدًا',
            'shares.unlimited': 'غير محدود',
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
        }
    };

    function t(key) {
        return I18N[LANG]?.[key] || I18N['en']?.[key] || key;
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

    function timeAgo(dateStr) {
        const date = new Date(dateStr);
        const now = new Date();
        const diffMs = date - now;
        const diffMin = Math.round(diffMs / 60000);
        const diffHr = Math.round(diffMs / 3600000);
        const diffDay = Math.round(diffMs / 86400000);

        if (diffMs < 0) {
            const ago = Math.abs(diffMin);
            if (ago < 1) return LANG === 'de' ? 'gerade eben' : 'just now';
            if (ago < 60) return LANG === 'de' ? `vor ${ago}m` : `${ago}m ago`;
            const hAgo = Math.abs(diffHr);
            if (hAgo < 24) return LANG === 'de' ? `vor ${hAgo}h` : `${hAgo}h ago`;
            const dAgo = Math.abs(diffDay);
            return LANG === 'de' ? `vor ${dAgo}d` : `${dAgo}d ago`;
        }
        if (diffMin < 60) return `${diffMin}m`;
        if (diffHr < 24) return `${diffHr}h`;
        return `${diffDay}d`;
    }

    function relativeExpiry(dateStr) {
        if (!dateStr) return t('shares.never');
        const date = new Date(dateStr);
        // Sentinel: backend stores 9999-12-31 for "unbegrenzt" shares because
        // the ExpiresAt column is NOT NULL. Any date past year 9000 is never.
        if (date.getFullYear() > 9000) return t('shares.never');
        const now = new Date();
        if (date <= now) return t('shares.expired');
        return t('shares.expires') + ' ' + timeAgo(dateStr);
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

    function escapeHtml(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
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
        if (res.status === 401 || res.status === 403) {
            showLogin();
            throw new Error('Unauthorized');
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
            <button class="toast-close" aria-label="Close">&times;</button>
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
    let networkConfig = null;

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
    }

    function renderUserInfo() {
        const el = document.getElementById('user-info');
        if (!el) return;

        const name = currentUser?.name || 'Admin';
        const role = currentUser?.role || 'admin';
        const initial = name.charAt(0).toUpperCase();

        el.innerHTML = `
            <div class="user-avatar">${escapeHtml(initial)}</div>
            <div>
                <div class="user-name">${escapeHtml(name)}</div>
                <div class="user-role">${escapeHtml(role)}</div>
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
            ? '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg> <span>Light Mode</span>'
            : '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg> <span>Dark Mode</span>';
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

    function renderFileList() {
        const container = document.getElementById('file-list');
        container.innerHTML = selectedFiles.map((f, i) => {
            const info = fileTypeInfo(f.name);
            return `
            <div class="file-item">
                <div class="file-item-info">
                    <div class="file-item-icon ${info.cls}">${info.label}</div>
                    <span class="file-item-name" title="${escapeHtml(f.name)}">${escapeHtml(f.name)}</span>
                </div>
                <span class="file-item-size">${formatSize(f.size)}</span>
                <button class="file-item-remove" data-index="${i}" title="Remove">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                </button>
            </div>`;
        }).join('');

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
                updateProgress(progressId, 100, false, true);
                toast(`${file.name}: ${err.message}`, 'error');
            }
        }

        selectedFiles = [];
        renderFileList();
        document.getElementById('upload-options').style.display = 'none';
        uploadBtn.disabled = false;
        uploadBtn.textContent = t('upload.submit');
    }

    function updateProgress(id, percent, complete = false, error = false) {
        const wrapper = document.getElementById(id);
        if (!wrapper) return;
        const bar = wrapper.querySelector('.progress-bar-inner');
        const pct = wrapper.querySelector('.progress-percent');
        bar.style.width = percent + '%';
        pct.textContent = Math.round(percent) + '%';
        if (complete) bar.classList.add('complete');
        if (error) bar.classList.add('error');
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
                    reject(new Error(xhr.responseText || 'Upload failed'));
                }
            });

            xhr.addEventListener('error', () => reject(new Error('Network error')));
            xhr.send(formData);
        });
    }

    async function uploadChunked(file, password, expiresIn, maxDownloads, progressId) {
        const totalChunks = Math.ceil(file.size / CHUNK_SIZE);

        const initRes = await api('/api/upload/chunk/init', {
            method: 'POST',
            body: JSON.stringify({
                fileName: file.name,
                totalSize: file.size,
                totalChunks: totalChunks,
            }),
        });

        if (!initRes.ok) throw new Error(await initRes.text());
        const { uploadId } = await initRes.json();

        for (let i = 0; i < totalChunks; i++) {
            const start = i * CHUNK_SIZE;
            const end = Math.min(start + CHUNK_SIZE, file.size);
            const chunk = file.slice(start, end);

            const chunkRes = await api(`/api/upload/chunk/${uploadId}?index=${i}`, {
                method: 'POST',
                body: chunk,
                headers: { 'Content-Type': 'application/octet-stream' },
            });

            if (!chunkRes.ok) throw new Error(await chunkRes.text());
            updateProgress(progressId, ((i + 1) / totalChunks) * 95);
        }

        const finalRes = await api(`/api/upload/chunk/${uploadId}/finalize`, {
            method: 'POST',
            body: JSON.stringify({
                password: password || '',
                expires_in: expiresIn,
                max_downloads: maxDownloads,
            }),
        });

        if (!finalRes.ok) throw new Error(await finalRes.text());
        return await finalRes.json();
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
                    Copy
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
    // Shares
    // ==========================================
    async function loadShares() {
        const listEl = document.getElementById('shares-list');
        const emptyEl = document.getElementById('shares-empty');
        const statsEl = document.getElementById('shares-stats');

        listEl.innerHTML = '<div style="text-align:center;padding:40px"><span class="spinner"></span></div>';
        emptyEl.style.display = 'none';

        try {
            const [sharesRes, statsRes] = await Promise.all([
                api('/api/shares'),
                api('/api/stats'),
            ]);

            const shares = sharesRes.ok ? await sharesRes.json() : [];
            currentShares = shares || [];
            const stats = statsRes.ok ? await statsRes.json() : {};

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
                            ${share.has_password ? '<span class="badge badge-password">Protected</span>' : ''}
                            ${share.is_directory ? '<span class="badge badge-directory">Folder</span>' : ''}
                            ${isExpiringSoon ? '<span class="badge badge-expiring">Expiring</span>' : ''}
                        </div>
                    </div>
                    <div class="share-actions">
                        ${url ? `<button class="btn-icon" title="Copy link" data-copy="${escapeHtml(url)}">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>
                        </button>` : ''}
                        <button class="btn-icon" title="QR Code" data-qr="${escapeHtml(share.id)}">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="3" height="3"/></svg>
                        </button>
                        <button class="btn-icon edit-share-btn" data-id="${escapeHtml(share.id)}" title="Edit">
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
                        <button class="btn-icon danger" title="Delete" data-delete="${escapeHtml(share.id)}">
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
                        <h3>QR Code</h3>
                        <div style="text-align:center;padding:20px">
                            <img src="/qr/${escapeHtml(btn.dataset.qr)}" alt="QR" style="width:200px;height:200px;background:#fff;padding:12px;border-radius:8px">
                        </div>
                        <div class="modal-actions">
                            <button class="btn btn-ghost" id="modal-close-btn">Close</button>
                        </div>
                    `);
                    document.getElementById('modal-close-btn')?.addEventListener('click', closeModal);
                });
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
                    selectedCountEl.textContent = checked.length + ' ' + t('shares.selected');
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

        const remaining = Math.max(1, Math.ceil((new Date(share.expires_at) - Date.now()) / 3600000));

        const html = `
            <h3>${t('shares.edit')}</h3>
            <p style="color:var(--text-secondary);margin-bottom:16px;font-size:0.9rem">${escapeHtml(share.original_name || share.file_name || share.id)}</p>
            <div class="form-group">
                <label>${t('upload.expiry')}</label>
                <input type="number" id="edit-expiry" value="${remaining}" min="1" max="8760">
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

        document.getElementById('save-edit-btn').onclick = async () => {
            const body = {};
            const expiry = parseInt(document.getElementById('edit-expiry').value);
            if (expiry > 0) body.expires_in_hours = expiry;
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
            <p style="color:var(--text-secondary);margin-bottom:16px;font-size:var(--text-sm)">${escapeHtml(share.original_name || share.file_name || share.id)}</p>
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
                    }),
                });
                if (!res.ok) throw new Error(await res.text());
                toast(t('email.sent'), 'success');
                closeModal();
            } catch(e) { toast(e.message || t('toast.error'), 'error'); }
        };
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
                            ${link.has_password ? '<span class="badge badge-password">Protected</span>' : ''}
                            ${link.auto_share ? '<span class="badge" style="background:var(--info-bg);color:var(--info);border:1px solid var(--info-border)">Auto-share</span>' : ''}
                        </div>
                    </div>
                    <div class="share-actions">
                        <button class="btn-icon" title="Copy link" data-copy="${escapeHtml(receiveUrl)}">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>
                        </button>
                        <button class="btn-icon danger" title="Delete" data-delete="${escapeHtml(link.id)}">
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
        ]);
    }

    async function loadNetworkConfig() {
        const container = document.getElementById('network-config');
        try {
            const res = await api('/api/network');
            if (!res.ok) {
                container.innerHTML = '<p style="color:var(--text-muted)">Could not load network config</p>';
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
                { key: 'custom', label: 'Custom URL', urlPrefix: '', urlSuffix: '' },
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
                                <input type="text" class="network-url-input" data-key="${n.key}" value="${escapeHtml(url)}" placeholder="${t('settings.notDetected')}" ${!enabled ? 'disabled' : ''}>
                                ${detected ? `
                                <div class="network-detected-hint">
                                    <span>${t('settings.detected')}: ${escapeHtml(detected)}</span>
                                    ${detected !== url ? `<button type="button" class="btn-use-detected" data-key="${n.key}" data-detected="${escapeHtml(detected)}" ${!enabled ? 'disabled' : ''}>${t('settings.useDetected')}</button>` : ''}
                                </div>` : ''}
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
                        loadNetworkConfig();
                    } else {
                        toast(t('toast.error'), 'error');
                    }
                } catch { toast(t('toast.error'), 'error'); }
            });

        } catch {
            container.innerHTML = '<p style="color:var(--text-muted)">Could not load network config</p>';
        }
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
            container.innerHTML = '<p style="color:var(--text-muted)">Could not load file restrictions</p>';
        }
    }

    async function loadWebhookConfig() {
        const container = document.getElementById('webhook-config');
        try {
            const res = await api('/api/webhook');
            const config = res.ok ? await res.json() : {};

            container.innerHTML = `
                <div class="form-group">
                    <label>${t('settings.webhookUrl')}</label>
                    <input type="url" id="webhook-url" value="${escapeHtml(config.url || config.webhook_url || '')}" placeholder="https://...">
                </div>
                <div class="form-group">
                    <label>${t('settings.webhookSecret')}</label>
                    <input type="password" id="webhook-secret" value="${escapeHtml(config.secret || config.webhook_secret || '')}" placeholder="">
                </div>
                <div class="form-actions">
                    <button class="btn btn-primary btn-sm" id="save-webhook-btn">${t('settings.save')}</button>
                    <button class="btn btn-ghost btn-sm" id="test-webhook-btn">${t('settings.testWebhook')}</button>
                </div>
            `;

            document.getElementById('save-webhook-btn').addEventListener('click', async () => {
                const body = {
                    url: document.getElementById('webhook-url').value,
                    webhook_url: document.getElementById('webhook-url').value,
                    secret: document.getElementById('webhook-secret').value,
                    webhook_secret: document.getElementById('webhook-secret').value,
                };
                try {
                    const saveRes = await api('/api/webhook', {
                        method: 'POST',
                        body: JSON.stringify(body),
                    });
                    if (saveRes.ok) toast(t('settings.saved'), 'success');
                    else toast(t('toast.error'), 'error');
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
            container.innerHTML = '<p style="color:var(--text-muted)">Could not load webhook config</p>';
        }
    }

    async function loadUserManagement() {
        const container = document.getElementById('user-management');
        try {
            const res = await api('/api/users');
            if (!res.ok) {
                container.innerHTML = '<p style="color:var(--text-muted)">User management not available</p>';
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
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        ${(users || []).map(u => `
                            <tr>
                                <td>${escapeHtml(u.name || '-')}</td>
                                <td>${escapeHtml(u.email || '-')}</td>
                                <td><span class="role-badge role-${(u.role || 'viewer').toLowerCase()}">${escapeHtml(u.role || 'viewer')}</span></td>
                                <td>
                                    <button class="btn-icon danger" data-delete-user="${escapeHtml(u.id)}" title="Delete">
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
                                <option value="admin">Admin</option>
                                <option value="user" selected>User</option>
                                <option value="viewer">Viewer</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label>${t('settings.userPassword')}</label>
                            <input type="password" id="new-user-password" required>
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

            document.getElementById('create-user-btn')?.addEventListener('click', async () => {
                const email = document.getElementById('new-user-email').value;
                const name = document.getElementById('new-user-name').value;
                const role = document.getElementById('new-user-role').value;
                const password = document.getElementById('new-user-password').value;

                if (!email || !name || !password) {
                    toast(t('settings.fillAll'), 'warning');
                    return;
                }

                try {
                    const createRes = await api('/api/users', {
                        method: 'POST',
                        body: JSON.stringify({ email, name, role, password }),
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
            container.innerHTML = '<p style="color:var(--text-muted)">User management not available</p>';
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
            container.innerHTML = '<p style="color:var(--text-muted)">Could not load 2FA config</p>';
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
                            <span class="role-badge role-${k.role}" style="margin-right:8px">${escapeHtml(k.role)}</span>
                            <button class="btn-icon danger delete-apikey-btn" data-id="${escapeHtml(k.id)}" title="Delete">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>
                            </button>
                        </div>
                    `).join('')}
                </div>
                <div style="margin-top:var(--space-4);display:flex;gap:8px;align-items:end">
                    <div class="form-group" style="margin:0;flex:1">
                        <label>${t('settings.name')}</label>
                        <input type="text" id="apikey-name" placeholder="My API Key">
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
                            <button class="btn btn-ghost btn-sm" style="margin-top:8px" id="copy-new-apikey-btn">Copy</button>
                        </div>
                    `;
                    document.getElementById('copy-new-apikey-btn').onclick = () => {
                        navigator.clipboard.writeText(data.key);
                        document.getElementById('copy-new-apikey-btn').textContent = 'Copied!';
                    };
                    loadAPIKeys();
                } else {
                    toast(t('toast.error'), 'error');
                }
            };
        } catch {
            container.innerHTML = '<p style="color:var(--text-muted)">Could not load API keys</p>';
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

            container.innerHTML = `
                <div class="form-row">
                    <div class="form-group"><label>SMTP Host</label><input type="text" id="smtp-host" value="${escapeHtml(config.host || '')}" placeholder="smtp.gmail.com"></div>
                    <div class="form-group"><label>Port</label><input type="number" id="smtp-port" value="${config.port || 587}" style="width:100px"></div>
                </div>
                <div class="form-row">
                    <div class="form-group"><label>Username</label><input type="text" id="smtp-user" value="${escapeHtml(config.username || '')}"></div>
                    <div class="form-group"><label>Password</label><input type="password" id="smtp-pass" value="${escapeHtml(config.password || '')}"></div>
                </div>
                <div class="form-row">
                    <div class="form-group"><label>From Email</label><input type="email" id="smtp-from" value="${escapeHtml(config.from_email || '')}"></div>
                    <div class="form-group"><label>From Name</label><input type="text" id="smtp-from-name" value="${escapeHtml(config.from_name || 'CasaDrop')}"></div>
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

            document.getElementById('save-smtp-btn').onclick = async () => {
                const body = {
                    enabled: document.getElementById('smtp-enabled').checked,
                    host: document.getElementById('smtp-host').value,
                    port: parseInt(document.getElementById('smtp-port').value) || 587,
                    username: document.getElementById('smtp-user').value,
                    password: document.getElementById('smtp-pass').value,
                    from_email: document.getElementById('smtp-from').value,
                    from_name: document.getElementById('smtp-from-name').value,
                    use_starttls: true,
                };
                const r = await api('/api/smtp', { method: 'POST', body: JSON.stringify(body) });
                toast(r.ok ? t('settings.saved') : t('toast.error'), r.ok ? 'success' : 'error');
            };

            document.getElementById('test-smtp-btn').onclick = async () => {
                const r = await api('/api/smtp/test', { method: 'POST' });
                toast(r.ok ? t('settings.smtpTestOk') : (await r.text() || t('toast.error')), r.ok ? 'success' : 'error');
            };
        } catch {
            container.innerHTML = '<p style="color:var(--text-muted)">Could not load SMTP config</p>';
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
