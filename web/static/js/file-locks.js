// File Locks View

async function renderFileLocks() {
    const container = document.getElementById('file-locks-container');
    if (!container) return;

    try {
        const locks = await apiCall('/file-locks');

        if (!locks || locks.length === 0) {
            container.innerHTML = renderEmptyState(
                'No active file locks',
                'File locks appear here when agents or processes lock files for exclusive access'
            );
            return;
        }

        container.innerHTML = `
            <table class="data-table" style="margin-top: 1rem;">
                <thead>
                    <tr>
                        <th>File Path</th>
                        <th>Locked By</th>
                        <th>Lock Type</th>
                        <th>Acquired</th>
                        <th>Duration</th>
                        <th>Status</th>
                    </tr>
                </thead>
                <tbody>
                    ${locks.map(lock => renderFileLockRow(lock)).join('')}
                </tbody>
            </table>
        `;
    } catch (error) {
        container.innerHTML = '<div class="error">Failed to load file locks: ' + escapeHtml(error.message) + '</div>';
    }
}

function renderFileLockRow(lock) {
    const acquired = lock.acquired_at ? new Date(lock.acquired_at) : null;
    const now = new Date();
    const duration = acquired ? formatLockDuration(now - acquired) : 'Unknown';
    const isExpired = lock.expires_at && new Date(lock.expires_at) < now;
    const status = isExpired ? '<span style="color: var(--warning-color);">Expired</span>' :
                   '<span style="color: var(--success-color);">Active</span>';

    return `
        <tr>
            <td><code class="code-small">${escapeHtml(lock.file_path || '')}</code></td>
            <td>${escapeHtml(lock.locked_by || 'Unknown')}</td>
            <td><span class="badge">${escapeHtml(lock.lock_type || 'exclusive')}</span></td>
            <td class="small">${acquired ? acquired.toLocaleString() : 'Unknown'}</td>
            <td class="small">${duration}</td>
            <td>${status}</td>
        </tr>
    `;
}

function formatLockDuration(ms) {
    const seconds = Math.floor(ms / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return days + 'd ' + (hours % 24) + 'h';
    if (hours > 0) return hours + 'h ' + (minutes % 60) + 'm';
    if (minutes > 0) return minutes + 'm ' + (seconds % 60) + 's';
    return seconds + 's';
}

// Refresh file locks button handler
document.addEventListener('DOMContentLoaded', function() {
    var refreshBtn = document.getElementById('refresh-file-locks-btn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', renderFileLocks);
    }
});
