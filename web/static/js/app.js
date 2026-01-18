// Configuration
const API_BASE = '/api/v1';
const REFRESH_INTERVAL = 5000; // 5 seconds

// State
let state = {
    beads: [],
    agents: [],
    projects: [],
    personas: [],
    decisions: []
};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadAll();
    startAutoRefresh();
});

// Auto-refresh
function startAutoRefresh() {
    setInterval(() => {
        loadAll();
    }, REFRESH_INTERVAL);
}

// Load all data
async function loadAll() {
    await Promise.all([
        loadBeads(),
        loadAgents(),
        loadProjects(),
        loadPersonas(),
        loadDecisions()
    ]);
    render();
}

// API calls
async function apiCall(endpoint, options = {}) {
    try {
        const response = await fetch(`${API_BASE}${endpoint}`, {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            }
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'API request failed');
        }
        
        if (response.status === 204) {
            return null;
        }
        
        return await response.json();
    } catch (error) {
        console.error('API Error:', error);
        alert(`Error: ${error.message}`);
        throw error;
    }
}

async function loadBeads() {
    state.beads = await apiCall('/beads');
}

async function loadAgents() {
    state.agents = await apiCall('/agents');
}

async function loadProjects() {
    state.projects = await apiCall('/projects');
}

async function loadPersonas() {
    state.personas = await apiCall('/personas');
}

async function loadDecisions() {
    state.decisions = await apiCall('/decisions');
}

// Render functions
function render() {
    renderKanban();
    renderAgents();
    renderProjects();
    renderPersonas();
    renderDecisions();
}

function renderKanban() {
    const openBeads = state.beads.filter(b => b.status === 'open');
    const inProgressBeads = state.beads.filter(b => b.status === 'in_progress');
    const closedBeads = state.beads.filter(b => b.status === 'closed');
    
    document.getElementById('open-beads').innerHTML = openBeads.map(renderBeadCard).join('');
    document.getElementById('in-progress-beads').innerHTML = inProgressBeads.map(renderBeadCard).join('');
    document.getElementById('closed-beads').innerHTML = closedBeads.map(renderBeadCard).join('');
}

function renderBeadCard(bead) {
    const priorityClass = `priority-${bead.priority}`;
    const typeClass = bead.type === 'decision' ? 'decision' : '';
    
    return `
        <div class="bead-card ${priorityClass} ${typeClass}" onclick="viewBead('${bead.id}')">
            <div class="bead-title">${escapeHtml(bead.title)}</div>
            <div class="bead-meta">
                <span>P${bead.priority}</span>
                <span>${bead.type}</span>
                ${bead.assigned_to ? `<span>👤 ${bead.assigned_to.substring(0, 8)}</span>` : ''}
            </div>
        </div>
    `;
}

function renderAgents() {
    const html = state.agents.map(agent => {
        const statusClass = agent.status;
        return `
            <div class="agent-card ${statusClass}">
                <div class="agent-header">
                    <span class="agent-name">${escapeHtml(agent.name)}</span>
                    <span class="agent-status ${statusClass}">${agent.status}</span>
                </div>
                <div>
                    <strong>Persona:</strong> ${escapeHtml(agent.persona_name)}<br>
                    <strong>Project:</strong> ${agent.project_id.substring(0, 12)}...<br>
                    ${agent.current_bead ? `<strong>Working on:</strong> ${agent.current_bead}` : ''}
                </div>
                <div style="margin-top: 1rem;">
                    <button class="danger" onclick="stopAgent('${agent.id}')">Stop Agent</button>
                </div>
            </div>
        `;
    }).join('');
    
    document.getElementById('agent-list').innerHTML = html || '<p>No active agents</p>';
}

function renderProjects() {
    const html = state.projects.map(project => `
        <div class="project-card">
            <h3>${escapeHtml(project.name)}</h3>
            <div>
                <strong>Branch:</strong> ${escapeHtml(project.branch)}<br>
                <strong>Repo:</strong> ${escapeHtml(project.git_repo)}<br>
                <strong>Agents:</strong> ${project.agents ? project.agents.length : 0}
            </div>
        </div>
    `).join('');
    
    document.getElementById('project-list').innerHTML = html || '<p>No projects configured</p>';
}

function renderPersonas() {
    const html = state.personas.map(persona => `
        <div class="persona-card" onclick="editPersona('${escapeHtml(persona.name)}')">
            <h3>🎭 ${escapeHtml(persona.name)}</h3>
            <div>
                <strong>Autonomy:</strong> ${escapeHtml(persona.autonomy_level || 'N/A')}<br>
                <strong>Character:</strong> ${escapeHtml((persona.character || '').substring(0, 100))}...
            </div>
        </div>
    `).join('');
    
    document.getElementById('persona-list').innerHTML = html || '<p>No personas available</p>';
}

function renderDecisions() {
    const html = state.decisions.map(decision => {
        const p0Class = decision.priority === 0 ? 'p0' : '';
        return `
            <div class="decision-card ${p0Class}">
                <div class="decision-question">${escapeHtml(decision.question)}</div>
                <div>
                    <strong>Priority:</strong> P${decision.priority}<br>
                    <strong>Requester:</strong> ${decision.requester_id}<br>
                    ${decision.recommendation ? `<strong>Recommendation:</strong> ${escapeHtml(decision.recommendation)}` : ''}
                </div>
                <div class="decision-actions">
                    <button onclick="claimDecision('${decision.id}')">Claim</button>
                    ${decision.status === 'in_progress' ? `<button class="secondary" onclick="makeDecision('${decision.id}')">Decide</button>` : ''}
                </div>
            </div>
        `;
    }).join('');
    
    document.getElementById('decision-list').innerHTML = html || '<p>No pending decisions</p>';
}

// Actions
function showSpawnAgentModal() {
    // Populate persona and project dropdowns
    const personaSelect = document.getElementById('agent-persona');
    const projectSelect = document.getElementById('agent-project');
    
    personaSelect.innerHTML = state.personas.map(p => 
        `<option value="${escapeHtml(p.name)}">${escapeHtml(p.name)}</option>`
    ).join('');
    
    projectSelect.innerHTML = state.projects.map(p => 
        `<option value="${p.id}">${escapeHtml(p.name)}</option>`
    ).join('');
    
    document.getElementById('spawn-agent-modal').style.display = 'block';
}

function closeSpawnAgentModal() {
    document.getElementById('spawn-agent-modal').style.display = 'none';
}

document.getElementById('spawn-agent-form')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const formData = new FormData(e.target);
    const data = {
        name: formData.get('name'),
        persona_name: formData.get('persona_name'),
        project_id: formData.get('project_id')
    };
    
    try {
        await apiCall('/agents', {
            method: 'POST',
            body: JSON.stringify(data)
        });
        
        closeSpawnAgentModal();
        loadAll();
    } catch (error) {
        // Error already handled
    }
});

async function stopAgent(agentId) {
    if (!confirm('Are you sure you want to stop this agent?')) {
        return;
    }
    
    try {
        await apiCall(`/agents/${agentId}`, {
            method: 'DELETE'
        });
        
        loadAll();
    } catch (error) {
        // Error already handled
    }
}

function viewBead(beadId) {
    const bead = state.beads.find(b => b.id === beadId);
    if (!bead) return;
    
    alert(`Bead: ${bead.title}\n\n${bead.description || 'No description'}`);
}

async function claimDecision(decisionId) {
    const userId = prompt('Enter your user ID (or agent ID):');
    if (!userId) return;
    
    try {
        await apiCall(`/beads/${decisionId}/claim`, {
            method: 'POST',
            body: JSON.stringify({ agent_id: userId })
        });
        
        loadAll();
    } catch (error) {
        // Error already handled
    }
}

async function makeDecision(decisionId) {
    const decision = prompt('Enter your decision:');
    if (!decision) return;
    
    const rationale = prompt('Enter rationale:');
    if (!rationale) return;
    
    const userId = prompt('Enter your user ID:');
    if (!userId) return;
    
    try {
        await apiCall(`/decisions/${decisionId}/decide`, {
            method: 'POST',
            body: JSON.stringify({
                decider_id: userId,
                decision: decision,
                rationale: rationale
            })
        });
        
        loadAll();
    } catch (error) {
        // Error already handled
    }
}

function editPersona(personaName) {
    alert(`Persona editor for "${personaName}" coming soon!\n\nFor now, edit ${personaName}/PERSONA.md and ${personaName}/AI_START_HERE.md files directly.`);
}

function closePersonaModal() {
    document.getElementById('persona-modal').style.display = 'none';
}

// Utilities
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Close modals when clicking outside
window.onclick = function(event) {
    const spawnModal = document.getElementById('spawn-agent-modal');
    const personaModal = document.getElementById('persona-modal');
    
    if (event.target === spawnModal) {
        closeSpawnAgentModal();
    }
    if (event.target === personaModal) {
        closePersonaModal();
    }
}
