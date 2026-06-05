// folder.js — public folder-share browse page (CSP-safe, no inline scripts/handlers).
// Server-rendered values arrive via data-* attributes on #folder-root.
(function () {
    var root = document.getElementById('folder-root');
    if (!root) return;

    var shareId = root.dataset.shareId || '';
    var hasPassword = root.dataset.hasPassword === 'true';
    var currentPath = '/';
    var password = '';

    function getStoredPassword() {
        var cookies = document.cookie.split(';');
        for (var i = 0; i < cookies.length; i++) {
            var c = cookies[i].trim();
            if (c.startsWith('folder_auth_' + shareId + '=')) {
                return decodeURIComponent(c.substring(('folder_auth_' + shareId + '=').length));
            }
        }
        return '';
    }

    function formatSize(bytes) {
        if (bytes === 0) return '0 B';
        var units = ['B', 'KB', 'MB', 'GB'];
        var i = Math.floor(Math.log(bytes) / Math.log(1024));
        return (bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0) + ' ' + units[i];
    }

    function handlePassword(e) {
        e.preventDefault();
        var pw = document.getElementById('folder-password').value;
        if (!pw) return false;

        fetch('/folder/' + shareId + '/contents?path=/&password=' + encodeURIComponent(pw))
            .then(function (resp) {
                if (resp.ok) {
                    password = pw;
                    document.cookie = 'folder_auth_' + shareId + '=' + encodeURIComponent(pw) + '; path=/; SameSite=Strict; max-age=86400';
                    document.getElementById('password-gate').style.display = 'none';
                    document.getElementById('folder-content').style.display = 'block';
                    return resp.json();
                } else {
                    document.getElementById('password-error').style.display = 'block';
                    return null;
                }
            })
            .then(function (data) {
                if (data) renderContents(data);
            })
            .catch(function () {
                document.getElementById('password-error').style.display = 'block';
            });
        return false;
    }

    function loadPath(path) {
        currentPath = path;
        var loading = document.getElementById('loading');
        var fileList = document.getElementById('file-list');
        var fetchError = document.getElementById('fetch-error');
        loading.style.display = 'block';
        fileList.style.display = 'none';
        fetchError.style.display = 'none';

        var url = '/folder/' + shareId + '/contents?path=' + encodeURIComponent(path);
        if (password) url += '&password=' + encodeURIComponent(password);

        fetch(url)
            .then(function (resp) {
                if (!resp.ok) throw new Error('Failed to load');
                return resp.json();
            })
            .then(function (data) {
                loading.style.display = 'none';
                fileList.style.display = 'block';
                renderContents(data);
            })
            .catch(function (err) {
                loading.style.display = 'none';
                fetchError.style.display = 'block';
                document.getElementById('fetch-error-text').textContent = 'Failed to load folder contents.';
            });
    }

    function renderBreadcrumbs() {
        var bc = document.getElementById('breadcrumbs');
        var parts = currentPath.split('/').filter(Boolean);
        var html = '<button class="breadcrumb-item' + (parts.length === 0 ? ' active' : '') + '" data-path="/">';
        html += '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>';
        html += ' Root</button>';

        var buildPath = '';
        for (var i = 0; i < parts.length; i++) {
            buildPath += '/' + parts[i];
            html += '<span class="breadcrumb-sep">/</span>';
            if (i === parts.length - 1) {
                html += '<span class="breadcrumb-item active">' + escapeHtml(parts[i]) + '</span>';
            } else {
                html += '<button class="breadcrumb-item" data-path="' + escapeAttr(buildPath) + '">' + escapeHtml(parts[i]) + '</button>';
            }
        }
        bc.innerHTML = html;
    }

    function renderContents(data) {
        renderBreadcrumbs();
        var body = document.getElementById('file-list-body');
        var items = data || [];

        if (items.length === 0) {
            body.innerHTML = '<div class="file-list-empty">This folder is empty</div>';
            return;
        }

        // Sort: directories first, then files alphabetically
        items.sort(function (a, b) {
            if (a.is_directory && !b.is_directory) return -1;
            if (!a.is_directory && b.is_directory) return 1;
            return (a.file_name || a.name || '').localeCompare(b.file_name || b.name || '');
        });

        var html = '';
        for (var i = 0; i < items.length; i++) {
            var item = items[i];
            var name = item.file_name || item.name || 'Unknown';
            var isDir = item.is_directory;
            var size = isDir ? '--' : formatSize(item.file_size || 0);
            var relPath = item.relative_path || (currentPath === '/' ? '/' + name : currentPath + '/' + name);

            if (isDir) {
                html += '<div class="file-row dir-row" data-path="' + escapeAttr(relPath) + '">';
                html += '<div class="file-row-name">';
                html += '<svg class="file-row-icon dir" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>';
                html += '<span class="file-row-label">' + escapeHtml(name) + '</span>';
                html += '</div>';
                html += '<span class="file-row-size">' + size + '</span>';
                html += '</div>';
            } else {
                var dlUrl = '/folder/' + shareId + '/download?path=' + encodeURIComponent(relPath);
                if (password) dlUrl += '&password=' + encodeURIComponent(password);
                html += '<a class="file-row" href="' + escapeAttr(dlUrl) + '">';
                html += '<div class="file-row-name">';
                html += '<svg class="file-row-icon file" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M13 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V9z"/><polyline points="13 2 13 9 20 9"/></svg>';
                html += '<span class="file-row-label">' + escapeHtml(name) + '</span>';
                html += '</div>';
                html += '<span class="file-row-size">' + size + '</span>';
                html += '</a>';
            }
        }
        body.innerHTML = html;
    }

    function downloadZip(e) {
        e.preventDefault();
        var url = '/folder/' + shareId + '/zip';
        if (password) url += '?password=' + encodeURIComponent(password);
        window.location.href = url;
    }

    function escapeHtml(str) {
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(str));
        return div.innerHTML;
    }

    function escapeAttr(str) {
        return String(str)
            .replace(/&/g, '&amp;')
            .replace(/"/g, '&quot;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;');
    }

    // Password form (only present when the share is password protected).
    var pwForm = document.getElementById('folder-password-form');
    if (pwForm) pwForm.addEventListener('submit', handlePassword);

    // ZIP download button.
    var zipBtn = document.getElementById('btn-zip');
    if (zipBtn) zipBtn.addEventListener('click', downloadZip);

    // Delegated navigation: breadcrumbs + directory rows carry data-path.
    var breadcrumbs = document.getElementById('breadcrumbs');
    if (breadcrumbs) {
        breadcrumbs.addEventListener('click', function (e) {
            var el = e.target.closest('[data-path]');
            if (el && breadcrumbs.contains(el)) loadPath(el.dataset.path);
        });
    }

    var fileListBody = document.getElementById('file-list-body');
    if (fileListBody) {
        fileListBody.addEventListener('click', function (e) {
            var el = e.target.closest('.dir-row[data-path]');
            if (el && fileListBody.contains(el)) loadPath(el.dataset.path);
        });
    }

    // Init
    if (hasPassword) {
        password = getStoredPassword();
        if (password) {
            // Try stored password
            fetch('/folder/' + shareId + '/contents?path=/&password=' + encodeURIComponent(password))
                .then(function (resp) {
                    if (resp.ok) {
                        var gate = document.getElementById('password-gate');
                        if (gate) gate.style.display = 'none';
                        document.getElementById('folder-content').style.display = 'block';
                        return resp.json();
                    } else {
                        password = '';
                        document.getElementById('loading').style.display = 'none';
                        return null;
                    }
                })
                .then(function (data) {
                    if (data) {
                        document.getElementById('loading').style.display = 'none';
                        document.getElementById('file-list').style.display = 'block';
                        renderContents(data);
                    }
                });
        }
    } else {
        loadPath('/');
    }
})();
