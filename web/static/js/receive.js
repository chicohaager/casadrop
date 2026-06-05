// receive.js — public receive-link upload page (CSP-safe, no inline scripts/handlers).
// Server-rendered values arrive via data-* attributes on #receive-root.
(function () {
    var root = document.getElementById('receive-root');
    if (!root) return;

    var linkId = root.dataset.linkId || '';
    var maxFileSize = parseInt(root.dataset.maxFileSize, 10) || 0;
    var maxUploads = parseInt(root.dataset.maxUploads, 10) || 0;
    var currentUploads = parseInt(root.dataset.currentUploads, 10) || 0;
    var selectedFile = null;

    // Format file size
    function formatSize(bytes) {
        if (bytes === 0) return '0 B';
        var units = ['B', 'KB', 'MB', 'GB'];
        var i = Math.floor(Math.log(bytes) / Math.log(1024));
        return (bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0) + ' ' + units[i];
    }

    // Set max size tag text
    var maxSizeEl = document.getElementById('max-size-text');
    if (maxSizeEl && maxFileSize > 0) {
        maxSizeEl.textContent = 'Max ' + formatSize(maxFileSize);
    }

    // Set counter bar
    if (maxUploads > 0) {
        var fillInit = document.getElementById('counter-fill');
        if (fillInit) {
            fillInit.style.width = Math.min((currentUploads / maxUploads) * 100, 100) + '%';
        }
    }

    // Drag & drop
    var dropZone = document.getElementById('drop-zone');
    var fileInput = document.getElementById('file-input');

    if (dropZone) {
        dropZone.addEventListener('dragover', function (e) {
            e.preventDefault();
            dropZone.classList.add('dragover');
        });
        dropZone.addEventListener('dragleave', function () {
            dropZone.classList.remove('dragover');
        });
        dropZone.addEventListener('drop', function (e) {
            e.preventDefault();
            dropZone.classList.remove('dragover');
            if (e.dataTransfer.files.length > 0) {
                selectFile(e.dataTransfer.files[0]);
            }
        });
    }

    if (fileInput) {
        fileInput.addEventListener('change', function () {
            if (fileInput.files.length > 0) {
                selectFile(fileInput.files[0]);
            }
        });
    }

    function selectFile(file) {
        selectedFile = file;
        document.getElementById('selected-name').textContent = file.name;
        document.getElementById('selected-size').textContent = formatSize(file.size);
        document.getElementById('selected-file').classList.add('visible');
        document.getElementById('btn-upload').classList.add('visible');

        // Hide previous statuses
        document.getElementById('status-success').classList.remove('visible');
        document.getElementById('status-error').classList.remove('visible');
    }

    function clearFile() {
        selectedFile = null;
        if (fileInput) fileInput.value = '';
        document.getElementById('selected-file').classList.remove('visible');
        document.getElementById('btn-upload').classList.remove('visible');
    }

    function uploadFile() {
        if (!selectedFile) return;

        // Client-side size check
        if (maxFileSize > 0 && selectedFile.size > maxFileSize) {
            showError('File is too large. Maximum size is ' + formatSize(maxFileSize) + '.');
            return;
        }

        var formData = new FormData();
        formData.append('file', selectedFile);

        var pwInput = document.getElementById('receive-password');
        if (pwInput && pwInput.value) {
            formData.append('password', pwInput.value);
        }

        var btn = document.getElementById('btn-upload');
        btn.disabled = true;
        btn.innerHTML = 'Uploading...';

        var progress = document.getElementById('progress-area');
        var progressFill = document.getElementById('progress-fill');
        var progressText = document.getElementById('progress-text');
        progress.classList.add('visible');

        var xhr = new XMLHttpRequest();
        xhr.open('POST', '/r/' + linkId + '/upload');

        xhr.upload.addEventListener('progress', function (e) {
            if (e.lengthComputable) {
                var pct = Math.round((e.loaded / e.total) * 100);
                progressFill.style.width = pct + '%';
                progressText.textContent = 'Uploading... ' + pct + '%';
            }
        });

        xhr.addEventListener('load', function () {
            progress.classList.remove('visible');
            if (xhr.status >= 200 && xhr.status < 300) {
                showSuccess(selectedFile.name);
                clearFile();
                currentUploads++;
                if (maxUploads > 0) {
                    var fill = document.getElementById('counter-fill');
                    if (fill) fill.style.width = Math.min((currentUploads / maxUploads) * 100, 100) + '%';
                    var counterText = document.querySelector('.counter-text');
                    if (counterText) counterText.innerHTML = '<strong>' + currentUploads + '</strong> / ' + maxUploads + ' uploads used';
                }
            } else {
                var msg = 'Upload failed.';
                try { msg = JSON.parse(xhr.responseText).error || msg; } catch (e) {}
                showError(msg);
            }
            btn.disabled = false;
            btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg> Upload File';
        });

        xhr.addEventListener('error', function () {
            progress.classList.remove('visible');
            showError('Network error. Please try again.');
            btn.disabled = false;
            btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg> Upload File';
        });

        xhr.send(formData);
    }

    function showSuccess(filename) {
        document.getElementById('success-detail').textContent = filename + ' has been uploaded.';
        document.getElementById('status-success').classList.add('visible');
        document.getElementById('status-error').classList.remove('visible');
    }

    function showError(msg) {
        document.getElementById('error-detail').textContent = msg;
        document.getElementById('status-error').classList.add('visible');
        document.getElementById('status-success').classList.remove('visible');
    }

    // Wire up the remove + upload buttons (replaces inline onclick handlers).
    var removeBtn = document.getElementById('selected-file-remove');
    if (removeBtn) removeBtn.addEventListener('click', clearFile);

    var uploadBtn = document.getElementById('btn-upload');
    if (uploadBtn) uploadBtn.addEventListener('click', uploadFile);
})();
