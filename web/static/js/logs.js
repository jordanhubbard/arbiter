// Log Viewer for AgentiCorp
// Displays structured logs, heartbeats, and system status

const LogViewer = {
    maxLogs: 1000,
    logs: [],
    filters: {
        level: 'all', // all, debug, info, warn, error
        source: 'all', // all, temporal, agent, provider, dispatcher, database, actions
        search: ''
    },
    autoScroll: true,

    init() {
        this.startPolling();
        this.setupEventHandlers();
    },

    async fetchLogs() {
        try {
            const response = await apiCall('/logs/recent?limit=100', { suppressToast: true });
            if (response && response.logs) {
                this.addLogs(response.logs);
            }
        } catch (error) {
            console.error('[LogViewer] Failed to fetch logs:', error);
        }
    },

    async fetchSystemMetrics() {
        try {
            const [status, agents, providers] = await Promise.all([
                apiCall('/system/status', { suppressToast: true }),
                apiCall('/agents', { suppressToast: true }),
                apiCall('/providers', { suppressToast: true })
            ]);
            
            return { status, agents, providers };
        } catch (error) {
            console.error('[LogViewer] Failed to fetch metrics:', error);
            return null;
        }
    },

    addLogs(newLogs) {
        this.logs.push(...newLogs);
        if (this.logs.length > this.maxLogs) {
            this.logs = this.logs.slice(-this.maxLogs);
        }
        this.render();
    },

    render() {
        const container = document.getElementById('logs-view');
        if (!container) return;

        const filtered = this.getFilteredLogs();
        const metricsHTML = this.renderMetrics();
        const logsHTML = this.renderLogs(filtered);

        container.innerHTML = `
            <div class="logs-container">
                ${metricsHTML}
                <div class="logs-filters">
                    ${this.renderFilters()}
                </div>
                <div class="logs-list" id="logs-list">
                    ${logsHTML}
                </div>
            </div>
        `;

        if (this.autoScroll) {
            const logsList = document.getElementById('logs-list');
            if (logsList) {
                logsList.scrollTop = logsList.scrollHeight;
            }
        }
    },

    renderMetrics() {
        // Will be populated by fetchSystemMetrics()
        return `
            <div class="metrics-panel">
                <div class="metric" id="metric-status">
                    <span class="metric-label">System:</span>
                    <span class="metric-value">Loading...</span>
                </div>
                <div class="metric" id="metric-agents">
                    <span class="metric-label">Agents:</span>
                    <span class="metric-value">Loading...</span>
                </div>
                <div class="metric" id="metric-providers">
                    <span class="metric-label">Providers:</span>
                    <span class="metric-value">Loading...</span>
                </div>
                <div class="metric" id="metric-beads">
                    <span class="metric-label">Open Beads:</span>
                    <span class="metric-value">Loading...</span>
                </div>
            </div>
        `;
    },

    renderFilters() {
        return `
            <select id="log-level-filter" onchange="LogViewer.setFilter('level', this.value)">
                <option value="all">All Levels</option>
                <option value="debug">Debug</option>
                <option value="info">Info</option>
                <option value="warn">Warning</option>
                <option value="error">Error</option>
            </select>
            
            <select id="log-source-filter" onchange="LogViewer.setFilter('source', this.value)">
                <option value="all">All Sources</option>
                <option value="temporal">Temporal</option>
                <option value="agent">Agents</option>
                <option value="provider">Providers</option>
                <option value="dispatcher">Dispatcher</option>
                <option value="database">Database</option>
                <option value="actions">Actions</option>
            </select>

            <input 
                type="text" 
                id="log-search" 
                placeholder="Search logs..."
                oninput="LogViewer.setFilter('search', this.value)"
            />

            <label>
                <input 
                    type="checkbox" 
                    ${this.autoScroll ? 'checked' : ''}
                    onchange="LogViewer.toggleAutoScroll()"
                />
                Auto-scroll
            </label>

            <button onclick="LogViewer.clearLogs()">Clear</button>
            <button onclick="LogViewer.exportLogs()">Export</button>
        `;
    },

    renderLogs(logs) {
        if (logs.length === 0) {
            return '<div class="log-empty">No logs match the current filters</div>';
        }

        return logs.map(log => {
            const levelClass = `log-${log.level || 'info'}`;
            const source = log.source || 'system';
            const sourceClass = source === 'actions' ? 'log-source-actions' : '';
            const timestamp = new Date(log.timestamp).toLocaleTimeString();
            
            const actionType = log.metadata && log.metadata.action_type
                ? `<span class="log-action-badge">${this.escapeHtml(String(log.metadata.action_type))}</span>`
                : '';
            const actionStatusValue = log.metadata && log.metadata.status
                ? String(log.metadata.status)
                : '';
            const actionStatusClass = actionStatusValue
                ? `log-action-status log-action-status-${this.escapeHtml(actionStatusValue.toLowerCase())}`
                : '';
            const actionStatus = actionStatusValue
                ? `<span class="${actionStatusClass}">${this.escapeHtml(actionStatusValue)}</span>`
                : '';

            return `
                <div class="log-entry ${levelClass} ${sourceClass}">
                    <span class="log-time">${timestamp}</span>
                    <span class="log-level">${log.level?.toUpperCase() || 'INFO'}</span>
                    <span class="log-source">[${source}]</span>
                    ${actionType}
                    ${actionStatus}
                    <span class="log-message">${this.escapeHtml(log.message)}</span>
                    ${log.metadata ? `<span class="log-metadata">${this.formatMetadata(log.metadata)}</span>` : ''}
                </div>
            `;
        }).join('');
    },

    getFilteredLogs() {
        return this.logs.filter(log => {
            if (this.filters.level !== 'all' && log.level !== this.filters.level) {
                return false;
            }
            if (this.filters.source !== 'all' && log.source !== this.filters.source) {
                return false;
            }
            if (this.filters.search && !log.message.toLowerCase().includes(this.filters.search.toLowerCase())) {
                return false;
            }
            return true;
        });
    },

    setFilter(name, value) {
        this.filters[name] = value;
        this.render();
    },

    toggleAutoScroll() {
        this.autoScroll = !this.autoScroll;
    },

    clearLogs() {
        this.logs = [];
        this.render();
    },

    exportLogs() {
        const data = JSON.stringify(this.logs, null, 2);
        const blob = new Blob([data], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `agenticorp-logs-${new Date().toISOString()}.json`;
        a.click();
        URL.revokeObjectURL(url);
    },

    formatMetadata(metadata) {
        if (typeof metadata === 'object') {
            return Object.entries(metadata)
                .map(([k, v]) => `${k}=${v}`)
                .join(', ');
        }
        return String(metadata);
    },

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    },

    setupEventHandlers() {
        // Listen for real-time log events via SSE if available
        if (window.eventSource) {
            window.eventSource.addEventListener('log', (event) => {
                try {
                    const log = JSON.parse(event.data);
                    this.addLogs([log]);
                } catch (error) {
                    console.error('[LogViewer] Failed to parse log event:', error);
                }
            });
        }
    },

    async updateMetrics() {
        const metrics = await this.fetchSystemMetrics();
        if (!metrics) return;

        const { status, agents, providers } = metrics;

        // Update system status
        const statusEl = document.getElementById('metric-status');
        if (statusEl && status) {
            const statusValue = statusEl.querySelector('.metric-value');
            statusValue.textContent = `${status.state || 'unknown'} (${status.reason || 'N/A'})`;
            statusValue.className = `metric-value status-${status.state || 'unknown'}`;
        }

        // Update agents
        const agentsEl = document.getElementById('metric-agents');
        if (agentsEl && agents) {
            const idle = agents.filter(a => a.status === 'idle').length;
            const working = agents.filter(a => a.status === 'working').length;
            const paused = agents.filter(a => a.status === 'paused').length;
            const agentsValue = agentsEl.querySelector('.metric-value');
            agentsValue.textContent = `${agents.length} total (${idle} idle, ${working} working, ${paused} paused)`;
        }

        // Update providers
        const providersEl = document.getElementById('metric-providers');
        if (providersEl && providers) {
            const healthy = providers.filter(p => p.status === 'healthy' || p.status === 'active').length;
            const providersValue = providersEl.querySelector('.metric-value');
            providersValue.textContent = `${healthy}/${providers.length} healthy`;
        }

        // Update beads (from global state if available)
        const beadsEl = document.getElementById('metric-beads');
        if (beadsEl && window.state && window.state.beads) {
            const openBeads = window.state.beads.filter(b => b.status === 'open').length;
            const beadsValue = beadsEl.querySelector('.metric-value');
            beadsValue.textContent = `${openBeads}`;
        }
    },

    startPolling() {
        // Fetch logs every 5 seconds
        setInterval(() => {
            this.fetchLogs();
            this.updateMetrics();
        }, 5000);

        // Initial fetch
        this.fetchLogs();
        this.updateMetrics();
    }
};

// Initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => LogViewer.init());
} else {
    LogViewer.init();
}
