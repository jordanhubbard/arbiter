// Workflow System JavaScript

// Initialize Mermaid
mermaid.initialize({
    startOnLoad: false,
    theme: 'default',
    flowchart: {
        useMaxWidth: true,
        htmlLabels: true,
        curve: 'basis'
    }
});

// State management
let currentWorkflow = null;
let workflows = [];
let executions = [];

// Tab switching
document.querySelectorAll('.workflow-tab').forEach(tab => {
    tab.addEventListener('click', () => {
        const tabName = tab.dataset.tab;
        switchTab(tabName);
    });
});

function switchTab(tabName) {
    // Update tab buttons
    document.querySelectorAll('.workflow-tab').forEach(t => t.classList.remove('active'));
    document.querySelector(`[data-tab="${tabName}"]`).classList.add('active');

    // Update tab content
    document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
    document.getElementById(`${tabName}-tab`).classList.add('active');

    // Load data for the tab
    if (tabName === 'workflows') {
        loadWorkflows();
    } else if (tabName === 'executions') {
        loadExecutions();
    } else if (tabName === 'analytics') {
        loadAnalytics();
    } else if (tabName === 'history') {
        loadHistory();
    }
}

// Load workflows
async function loadWorkflows() {
    const loading = document.getElementById('workflows-loading');
    const error = document.getElementById('workflows-error');
    const list = document.getElementById('workflow-list');

    loading.style.display = 'block';
    error.style.display = 'none';
    list.innerHTML = '';

    try {
        const response = await fetch('/api/v1/workflows');
        if (!response.ok) throw new Error('Failed to load workflows');

        const data = await response.json();
        workflows = data.workflows || [];

        loading.style.display = 'none';

        if (workflows.length === 0) {
            list.innerHTML = '<p>No workflows found</p>';
            return;
        }

        workflows.forEach(workflow => {
            const card = createWorkflowCard(workflow);
            list.appendChild(card);
        });

    } catch (err) {
        loading.style.display = 'none';
        error.textContent = 'Error loading workflows: ' + err.message;
        error.style.display = 'block';
    }
}

function createWorkflowCard(workflow) {
    const card = document.createElement('div');
    card.className = 'workflow-card';
    card.onclick = () => showWorkflowDetail(workflow.id);

    const typeClass = workflow.workflow_type || 'bug';
    const nodeCount = workflow.nodes ? workflow.nodes.length : 0;
    const edgeCount = workflow.edges ? workflow.edges.length : 0;

    card.innerHTML = `
        <span class="workflow-type ${typeClass}">${workflow.workflow_type || 'unknown'}</span>
        <h3>${workflow.name}</h3>
        <p>${workflow.description || 'No description'}</p>
        <div class="workflow-stats">
            <span><strong>${nodeCount}</strong> nodes</span>
            <span><strong>${edgeCount}</strong> edges</span>
            ${workflow.is_default ? '<span>✓ Default</span>' : ''}
        </div>
    `;

    return card;
}

async function showWorkflowDetail(workflowId) {
    const list = document.getElementById('workflow-list');
    const detail = document.getElementById('workflow-detail');

    try {
        const response = await fetch(`/api/v1/workflows/${workflowId}`);
        if (!response.ok) throw new Error('Failed to load workflow details');

        const workflow = await response.json();
        currentWorkflow = workflow;

        list.style.display = 'none';
        detail.style.display = 'block';

        renderWorkflowDetail(workflow);

    } catch (err) {
        alert('Error loading workflow: ' + err.message);
    }
}

function renderWorkflowDetail(workflow) {
    const detail = document.getElementById('workflow-detail');

    const mermaidGraph = generateMermaidDiagram(workflow);

    detail.innerHTML = `
        <div class="workflow-detail">
            <button class="back-button" onclick="backToWorkflowList()">← Back to Workflows</button>

            <h2>${workflow.name}</h2>
            <span class="workflow-type ${workflow.workflow_type}">${workflow.workflow_type}</span>

            <p>${workflow.description || 'No description'}</p>

            <h3>Workflow Diagram</h3>
            <div class="workflow-diagram">
                <div class="mermaid" id="workflow-mermaid">${mermaidGraph}</div>
            </div>

            <h3>Nodes (${workflow.nodes ? workflow.nodes.length : 0})</h3>
            <div class="node-list">
                ${renderNodes(workflow.nodes)}
            </div>

            <h3>Edges (${workflow.edges ? workflow.edges.length : 0})</h3>
            <div class="edge-list">
                ${renderEdges(workflow.edges)}
            </div>
        </div>
    `;

    // Render Mermaid diagram
    mermaid.run({
        querySelector: '#workflow-mermaid'
    });
}

function generateMermaidDiagram(workflow) {
    if (!workflow.nodes || !workflow.edges) {
        return 'graph TD\n    A[No workflow data]';
    }

    let graph = 'graph TD\n';

    // Add start node
    graph += '    START([Start])\n';

    // Add all nodes
    workflow.nodes.forEach(node => {
        const shape = getNodeShape(node.node_type);
        const label = `${node.node_key}\\n[${node.node_type}]`;
        graph += `    ${node.node_key}${shape[0]}${label}${shape[1]}\n`;
    });

    // Add end node
    graph += '    END([End])\n';

    // Add edges
    workflow.edges.forEach(edge => {
        const from = edge.from_node_key || 'START';
        const to = edge.to_node_key || 'END';
        const condition = edge.condition || 'success';
        graph += `    ${from} -->|${condition}| ${to}\n`;
    });

    // Add styling
    graph += '    classDef taskNode fill:#e3f2fd,stroke:#1565c0,stroke-width:2px\n';
    graph += '    classDef approvalNode fill:#fff3e0,stroke:#ef6c00,stroke-width:2px\n';
    graph += '    classDef commitNode fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px\n';

    workflow.nodes.forEach(node => {
        if (node.node_type === 'task') {
            graph += `    class ${node.node_key} taskNode\n`;
        } else if (node.node_type === 'approval') {
            graph += `    class ${node.node_key} approvalNode\n`;
        } else if (node.node_type === 'commit') {
            graph += `    class ${node.node_key} commitNode\n`;
        }
    });

    return graph;
}

function generateMermaidDiagramWithHighlight(workflow, currentNodeKey) {
    if (!workflow.nodes || !workflow.edges) {
        return 'graph TD\n    A[No workflow data]';
    }

    let graph = 'graph TD\n';

    // Add start node
    graph += '    START([Start])\n';

    // Add all nodes
    workflow.nodes.forEach(node => {
        const shape = getNodeShape(node.node_type);
        const label = `${node.node_key}\\n[${node.node_type}]`;
        graph += `    ${node.node_key}${shape[0]}${label}${shape[1]}\n`;
    });

    // Add end node
    graph += '    END([End])\n';

    // Add edges
    workflow.edges.forEach(edge => {
        const from = edge.from_node_key || 'START';
        const to = edge.to_node_key || 'END';
        const condition = edge.condition || 'success';
        graph += `    ${from} -->|${condition}| ${to}\n`;
    });

    // Add styling
    graph += '    classDef taskNode fill:#e3f2fd,stroke:#1565c0,stroke-width:2px\n';
    graph += '    classDef approvalNode fill:#fff3e0,stroke:#ef6c00,stroke-width:2px\n';
    graph += '    classDef commitNode fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px\n';
    graph += '    classDef currentNode fill:#ffeb3b,stroke:#f57c00,stroke-width:4px\n';

    workflow.nodes.forEach(node => {
        if (node.node_key === currentNodeKey) {
            graph += `    class ${node.node_key} currentNode\n`;
        } else if (node.node_type === 'task') {
            graph += `    class ${node.node_key} taskNode\n`;
        } else if (node.node_type === 'approval') {
            graph += `    class ${node.node_key} approvalNode\n`;
        } else if (node.node_type === 'commit') {
            graph += `    class ${node.node_key} commitNode\n`;
        }
    });

    return graph;
}

function getNodeShape(nodeType) {
    switch (nodeType) {
        case 'approval':
            return ['{', '}'];  // Hexagon
        case 'commit':
            return ['[[', ']]'];  // Subroutine
        case 'verify':
            return ['[/', '/]'];  // Parallelogram
        default:
            return ['[', ']'];  // Rectangle
    }
}

function renderNodes(nodes) {
    if (!nodes || nodes.length === 0) {
        return '<p>No nodes defined</p>';
    }

    return nodes.map(node => `
        <div class="node-item">
            <h4>${node.node_key}</h4>
            <div class="node-meta">
                <span class="meta-item"><span class="meta-label">Type:</span> ${node.node_type}</span>
                ${node.role_required ? `<span class="meta-item"><span class="meta-label">Role:</span> ${node.role_required}</span>` : ''}
                ${node.max_attempts > 0 ? `<span class="meta-item"><span class="meta-label">Max Attempts:</span> ${node.max_attempts}</span>` : ''}
                ${node.timeout_minutes > 0 ? `<span class="meta-item"><span class="meta-label">Timeout:</span> ${node.timeout_minutes}m</span>` : ''}
            </div>
            ${node.instructions ? `<p style="margin-top:10px;color:#666;">${node.instructions}</p>` : ''}
        </div>
    `).join('');
}

function renderEdges(edges) {
    if (!edges || edges.length === 0) {
        return '<p>No edges defined</p>';
    }

    return edges.map(edge => `
        <div class="edge-item">
            <h4>${edge.from_node_key || 'START'} → ${edge.to_node_key || 'END'}</h4>
            <div class="edge-meta">
                <span class="meta-item"><span class="meta-label">Condition:</span> ${edge.condition}</span>
                <span class="meta-item"><span class="meta-label">Priority:</span> ${edge.priority}</span>
            </div>
        </div>
    `).join('');
}

function backToWorkflowList() {
    const list = document.getElementById('workflow-list');
    const detail = document.getElementById('workflow-detail');

    list.style.display = 'grid';
    detail.style.display = 'none';
    currentWorkflow = null;
}

// Load executions
async function loadExecutions() {
    const loading = document.getElementById('executions-loading');
    const error = document.getElementById('executions-error');
    const list = document.getElementById('execution-list');

    loading.style.display = 'block';
    error.style.display = 'none';
    list.innerHTML = '';

    try {
        // For now, show a message since we don't have a full list endpoint
        loading.style.display = 'none';
        list.innerHTML = `
            <div class="workflow-detail">
                <h3>Active Executions</h3>
                <p>To view workflow execution for a specific bead, enter the bead ID below:</p>
                <div style="margin-top:20px;">
                    <input type="text" id="bead-id-input" placeholder="Enter bead ID (e.g., ac-1234)" style="padding:10px; width:300px; border:1px solid #ddd; border-radius:4px;">
                    <button onclick="loadBeadWorkflow()" style="padding:10px 20px; margin-left:10px; background:#2196F3; color:white; border:none; border-radius:4px; cursor:pointer;">View Workflow</button>
                </div>
                <div id="bead-workflow-result" style="margin-top:30px;"></div>
            </div>
        `;

    } catch (err) {
        loading.style.display = 'none';
        error.textContent = 'Error loading executions: ' + err.message;
        error.style.display = 'block';
    }
}

async function loadBeadWorkflow() {
    const beadId = document.getElementById('bead-id-input').value.trim();
    const result = document.getElementById('bead-workflow-result');

    if (!beadId) {
        result.innerHTML = '<div class="error">Please enter a bead ID</div>';
        return;
    }

    result.innerHTML = '<div class="loading">Loading workflow for bead ' + beadId + '...</div>';

    try {
        const response = await fetch(`/api/v1/beads/workflow?bead_id=${beadId}`);
        if (!response.ok) throw new Error('Failed to load workflow');

        const data = await response.json();

        if (data.message) {
            result.innerHTML = `<p>${data.message}</p>`;
            return;
        }

        renderBeadWorkflow(data);

    } catch (err) {
        result.innerHTML = `<div class="error">Error: ${err.message}</div>`;
    }
}

function renderBeadWorkflow(data) {
    const result = document.getElementById('bead-workflow-result');

    const statusClass = data.execution.status || 'active';
    const mermaidGraph = generateMermaidDiagramWithHighlight(data.workflow, data.execution.current_node_key);

    let html = `
        <div class="execution-card">
            <h3>Workflow Execution for Bead ${data.bead_id}</h3>
            <span class="execution-status ${statusClass}">${data.execution.status}</span>

            <div style="margin-top:20px;">
                <h4>Workflow: ${data.workflow.name}</h4>
                <p><strong>Current Node:</strong> ${data.execution.current_node_key || 'Not started'}</p>
                <p><strong>Cycle Count:</strong> ${data.execution.cycle_count}</p>
                <p><strong>Node Attempts:</strong> ${data.execution.node_attempt_count}</p>
                <p><strong>Started:</strong> ${new Date(data.execution.started_at).toLocaleString()}</p>
            </div>

            <div class="workflow-diagram" style="margin-top:20px;">
                <h4>Workflow Progress</h4>
                <div class="mermaid" id="execution-mermaid">${mermaidGraph}</div>
            </div>

            ${data.current_node ? `
                <div style="margin-top:20px; padding:15px; background:#f5f5f5; border-radius:4px;">
                    <h4>Current Node: ${data.current_node.node_key}</h4>
                    <p><strong>Type:</strong> ${data.current_node.node_type}</p>
                    <p><strong>Role Required:</strong> ${data.current_node.role_required || 'None'}</p>
                    ${data.current_node.instructions ? `<p><strong>Instructions:</strong> ${data.current_node.instructions}</p>` : ''}
                </div>
            ` : ''}

            ${data.history && data.history.length > 0 ? `
                <div class="history-timeline">
                    <h4>Execution History</h4>
                    ${data.history.map(h => `
                        <div class="history-item">
                            <div class="history-timestamp">${new Date(h.created_at).toLocaleString()}</div>
                            <p><strong>Node:</strong> ${h.node_key || 'START'}</p>
                            <p><strong>Condition:</strong> ${h.condition}</p>
                            <p><strong>Agent:</strong> ${h.agent_id}</p>
                            ${h.attempt_number > 0 ? `<p><strong>Attempt:</strong> ${h.attempt_number}</p>` : ''}
                        </div>
                    `).join('')}
                </div>
            ` : ''}
        </div>
    `;

    result.innerHTML = html;

    // Render Mermaid diagram
    setTimeout(() => {
        mermaid.run({
            querySelector: '#execution-mermaid'
        });
    }, 100);
}

// Load analytics
async function loadAnalytics() {
    const loading = document.getElementById('analytics-loading');
    const error = document.getElementById('analytics-error');
    const content = document.getElementById('analytics-content');

    loading.style.display = 'block';
    error.style.display = 'none';
    content.innerHTML = '';

    try {
        const response = await fetch('/api/v1/workflows/analytics');
        if (!response.ok) throw new Error('Failed to load analytics');

        const data = await response.json();

        loading.style.display = 'none';

        renderAnalytics(data);

    } catch (err) {
        loading.style.display = 'none';
        error.textContent = 'Error loading analytics: ' + err.message;
        error.style.display = 'block';
    }
}

function renderAnalytics(data) {
    const content = document.getElementById('analytics-content');

    const statusActive = data.status_counts.active || 0;
    const statusCompleted = data.status_counts.completed || 0;
    const statusEscalated = data.status_counts.escalated || 0;
    const statusFailed = data.status_counts.failed || 0;

    const typeBug = data.type_counts.bug || 0;
    const typeFeature = data.type_counts.feature || 0;
    const typeUI = data.type_counts.ui || 0;

    content.innerHTML = `
        <div class="workflow-detail">
            <h2>Workflow Analytics Dashboard</h2>

            <div class="analytics-grid">
                <div class="metric-card">
                    <h3>Total Executions</h3>
                    <div class="metric-value">${data.total_executions || 0}</div>
                    <div class="status-breakdown">
                        <span class="status-badge execution-status active">${statusActive} active</span>
                        <span class="status-badge execution-status completed">${statusCompleted} completed</span>
                        <span class="status-badge execution-status escalated">${statusEscalated} escalated</span>
                        ${statusFailed > 0 ? `<span class="status-badge execution-status failed">${statusFailed} failed</span>` : ''}
                    </div>
                </div>

                <div class="metric-card">
                    <h3>Escalation Rate</h3>
                    <div class="metric-value">${data.escalation_rate.toFixed(1)}%</div>
                    <div class="metric-label">${data.escalated_count} of ${data.total_executions} escalated</div>
                </div>

                <div class="metric-card">
                    <h3>Average Cycles</h3>
                    <div class="metric-value">${data.average_cycles.toFixed(2)}</div>
                    <div class="metric-label">Max: ${data.max_cycles} cycles</div>
                </div>

                <div class="metric-card">
                    <h3>Workflow Types</h3>
                    <div class="status-breakdown" style="margin-top: 15px;">
                        <span class="status-badge workflow-type bug">${typeBug} bug</span>
                        <span class="status-badge workflow-type feature">${typeFeature} feature</span>
                        <span class="status-badge workflow-type ui">${typeUI} ui</span>
                    </div>
                </div>
            </div>

            <div class="recent-executions">
                <h3>Recent Workflow Executions</h3>
                ${data.recent_executions && data.recent_executions.length > 0 ? `
                    <table>
                        <thead>
                            <tr>
                                <th>Bead ID</th>
                                <th>Workflow</th>
                                <th>Current Node</th>
                                <th>Status</th>
                                <th>Cycles</th>
                                <th>Started</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${data.recent_executions.map(exec => `
                                <tr onclick="viewExecution('${exec.bead_id}')" style="cursor: pointer;">
                                    <td><strong>${exec.bead_id}</strong></td>
                                    <td>${exec.workflow_name}</td>
                                    <td>${exec.current_node_key || '(start)'}</td>
                                    <td><span class="execution-status ${exec.status}">${exec.status}</span></td>
                                    <td>${exec.cycle_count}</td>
                                    <td>${new Date(exec.started_at).toLocaleString()}</td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                ` : '<p>No recent executions</p>'}
            </div>
        </div>
    `;
}

function viewExecution(beadId) {
    // Switch to executions tab and load this bead
    switchTab('executions');
    setTimeout(() => {
        const input = document.getElementById('bead-id-input');
        if (input) {
            input.value = beadId;
            loadBeadWorkflow();
        }
    }, 100);
}

// Load history
async function loadHistory() {
    const loading = document.getElementById('history-loading');
    const content = document.getElementById('history-content');

    loading.style.display = 'block';
    content.innerHTML = '';

    // Placeholder for history view
    setTimeout(() => {
        loading.style.display = 'none';
        content.innerHTML = `
            <div class="workflow-detail">
                <h3>Workflow Execution History</h3>
                <p>Full history view coming soon. Use the "Active Executions" tab to view individual bead workflows.</p>
            </div>
        `;
    }, 500);
}

// Real-time updates via SSE
let eventSource = null;

function connectToEventStream() {
    if (eventSource) {
        eventSource.close();
    }

    eventSource = new EventSource('/api/v1/events/stream');

    eventSource.addEventListener('bead.status_change', (e) => {
        const data = JSON.parse(e.data);
        console.log('[Workflow] Bead status changed:', data);

        // Refresh execution view if we're viewing this bead
        const beadIdInput = document.getElementById('bead-id-input');
        if (beadIdInput && beadIdInput.value === data.bead_id) {
            loadBeadWorkflow();
        }
    });

    eventSource.addEventListener('workflow.advanced', (e) => {
        const data = JSON.parse(e.data);
        console.log('[Workflow] Workflow advanced:', data);

        // Refresh execution view if we're viewing this bead
        const beadIdInput = document.getElementById('bead-id-input');
        if (beadIdInput && beadIdInput.value === data.bead_id) {
            loadBeadWorkflow();
        }
    });

    eventSource.onerror = (error) => {
        console.error('[Workflow] EventSource error:', error);
        // Reconnect after 5 seconds
        setTimeout(connectToEventStream, 5000);
    };
}

// Auto-refresh for active execution view
let autoRefreshInterval = null;

function startAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
    }

    // Refresh every 5 seconds if viewing an execution
    autoRefreshInterval = setInterval(() => {
        const beadIdInput = document.getElementById('bead-id-input');
        const result = document.getElementById('bead-workflow-result');
        if (beadIdInput && beadIdInput.value && result.innerHTML) {
            loadBeadWorkflow();
        }
    }, 5000);
}

function stopAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
    }
}

// Initialize on page load
loadWorkflows();
connectToEventStream();
startAutoRefresh();
