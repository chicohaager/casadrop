// share.js — public share page logic (CSP-safe, no inline scripts/handlers).
// Server-rendered values arrive via data-* attributes on #share-root.
(function () {
    var root = document.getElementById('share-root');
    if (!root) return;

    var shareId = root.dataset.shareId || '';
    var hasPassword = root.dataset.hasPassword === 'true';

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

    function handlePassword(e) {
        e.preventDefault();
        var pw = document.getElementById('share-password').value;
        if (!pw) return false;

        fetch('/d/' + shareId + '?password=' + encodeURIComponent(pw), { method: 'HEAD' })
            .then(function (resp) {
                if (resp.ok) {
                    document.cookie = 'share_auth_' + shareId + '=' + encodeURIComponent(pw) + '; path=/; SameSite=Strict; max-age=86400';
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

    // On page load, if we have a stored password, append it to download link.
    if (hasPassword) {
        var pw = getPassword();
        if (pw) {
            var btn = document.getElementById('download-btn');
            if (btn) {
                btn.href = '/d/' + shareId + '?password=' + encodeURIComponent(pw);
            }
        }
    }
})();
