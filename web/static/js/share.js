// share.js — public share page logic (CSP-safe, no inline scripts/handlers).
// Server-rendered values arrive via data-* attributes on #share-root.
(function () {
    var root = document.getElementById('share-root');
    if (!root) return;

    var shareId = root.dataset.shareId || '';
    var hasPassword = root.dataset.hasPassword === 'true';

    // Whether a media `error` event may be believed yet — see showMediaError.
    // Set where the media element is wired, and again by revealContent.
    var mediaArmed = false;

    function getPassword() {
        var cookies = document.cookie.split(';');
        for (var i = 0; i < cookies.length; i++) {
            var c = cookies[i].trim();
            if (c.startsWith('share_auth_' + shareId + '=')) {
                return decodeURIComponent(c.substring(('share_auth_' + shareId + '=').length));
            }
        }
        return '';
    }

    // Unlock the page: hide the password gate, show the file content, and wire
    // the validated password into the download link. Preview resources
    // (/thumbnail, /stream) authenticate via the share_auth cookie set below.
    function revealContent(pw) {
        var content = document.getElementById('share-content');
        if (content) {
            // Activate deferred preview sources now that the password is valid
            // (the cookie set by the caller authenticates /thumbnail and /stream).
            var deferred = content.querySelectorAll('[data-src]');
            for (var i = 0; i < deferred.length; i++) {
                deferred[i].src = deferred[i].getAttribute('data-src');
                deferred[i].removeAttribute('data-src');
            }
            // The deferred sources have just been given a real src. Clear any
            // note the empty-source errors left behind, then arm the handler so
            // a genuine decode failure from here on is still reported.
            hideMediaError();
            mediaArmed = true;
            var media = content.querySelectorAll('video, audio');
            for (var j = 0; j < media.length; j++) {
                media[j].load();
            }
            content.style.display = '';
        }
        var gate = document.getElementById('password-gate');
        if (gate) gate.style.display = 'none';
        var btn = document.getElementById('download-btn');
        if (btn) {
            btn.href = '/d/' + shareId + '?password=' + encodeURIComponent(pw);
        }
    }

    function handlePassword(e) {
        e.preventDefault();
        var pw = document.getElementById('share-password').value;
        if (!pw) return false;

        fetch('/d/' + shareId + '?password=' + encodeURIComponent(pw), { method: 'HEAD' })
            .then(function (resp) {
                if (resp.ok) {
                    document.cookie = 'share_auth_' + shareId + '=' + encodeURIComponent(pw) + '; path=/; SameSite=Strict; max-age=86400';
                    // Reload so preview resources fetch with the cookie already
                    // present; the on-load handler below then reveals the content.
                    location.reload();
                } else {
                    var err = document.getElementById('password-error');
                    err.style.display = 'block';
                    document.getElementById('share-password').value = '';
                    document.getElementById('share-password').focus();
                }
            })
            .catch(function () {
                document.getElementById('password-error').style.display = 'block';
            });
        return false;
    }

    function toggleQR() {
        var overlay = document.getElementById('qr-overlay');
        if (overlay) overlay.classList.toggle('active');
    }

    // Password form
    var pwForm = document.getElementById('password-form');
    if (pwForm) {
        pwForm.addEventListener('submit', handlePassword);
    }

    // Image preview: click to load full resolution (replaces inline onclick).
    var previewImg = document.getElementById('preview-image');
    if (previewImg) {
        previewImg.addEventListener('click', function onPreviewClick() {
            previewImg.src = '/stream/' + shareId;
            previewImg.style.cursor = 'default';
            previewImg.removeEventListener('click', onPreviewClick);
        });
    }

    // Media that cannot be decoded must say so. Left alone, <video> renders a
    // black rectangle and <audio> stays silent — a failure the visitor can only
    // interpret as "the file is broken". Both the element-level error and the
    // <source>-level error are wired: the former fires when the media itself is
    // undecodable, the latter when no listed source could be used at all.
    //
    // But on a password-protected share the <source> elements carry data-src and
    // no src at all (see revealContent), and the browser fires `error` on such a
    // source the moment it parses the media element — before the visitor has even
    // seen the password prompt. That is not a decode failure, so the listener
    // stays disarmed until the real sources are in place; otherwise every locked
    // audio/video share showed "your browser cannot play this file" sitting above
    // a player that was working perfectly.
    function showMediaError() {
        if (!mediaArmed) return;
        var note = document.getElementById('media-error');
        if (note) note.style.display = '';
    }

    function hideMediaError() {
        var note = document.getElementById('media-error');
        if (note) note.style.display = 'none';
    }

    var media = document.getElementById('media-preview');
    if (media) {
        // Armed right away when nothing is deferred (a share with no password
        // already has real src attributes); revealContent arms it otherwise.
        mediaArmed = media.querySelectorAll('source[data-src]').length === 0;
        media.addEventListener('error', showMediaError);
        var sources = media.getElementsByTagName('source');
        for (var s = 0; s < sources.length; s++) {
            sources[s].addEventListener('error', showMediaError);
        }
    }

    // QR toggle button + overlay.
    var qrBtn = document.getElementById('qr-toggle-btn');
    if (qrBtn) qrBtn.addEventListener('click', toggleQR);

    var qrOverlay = document.getElementById('qr-overlay');
    if (qrOverlay) {
        qrOverlay.addEventListener('click', toggleQR);
        var qrModal = qrOverlay.querySelector('.qr-modal');
        if (qrModal) {
            qrModal.addEventListener('click', function (e) { e.stopPropagation(); });
        }
    }

    // On page load with a stored password (returning visitor or post-reload),
    // re-validate it server-side before unlocking. A valid password reveals the
    // content; a stale one (e.g. the share password was changed) is cleared so
    // the gate is shown again instead of leaving the user stuck.
    if (hasPassword) {
        var pw = getPassword();
        if (pw) {
            fetch('/d/' + shareId + '?password=' + encodeURIComponent(pw), { method: 'HEAD' })
                .then(function (resp) {
                    if (resp.ok) {
                        revealContent(pw);
                    } else {
                        document.cookie = 'share_auth_' + shareId + '=; path=/; max-age=0';
                    }
                })
                .catch(function () { /* leave the gate visible on network error */ });
        }
    }
})();
