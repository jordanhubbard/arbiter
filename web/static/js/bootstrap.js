// Bootstrap Project functionality

async function showBootstrapProjectModal() {
    const modalHTML = `
        <div class="modal" id="bootstrap-modal" role="dialog" aria-labelledby="bootstrap-title" aria-modal="true">
            <div class="modal-content">
                <div class="modal-header">
                    <h2 id="bootstrap-title">Create New Project</h2>
                    <button type="button" class="modal-close" onclick="closeBootstrapModal()" aria-label="Close">&times;</button>
                </div>
                <div class="modal-body">
                    <form id="bootstrap-form">
                        <div class="form-group">
                            <label for="bootstrap-github-url">GitHub Repository URL *</label>
                            <input type="text" id="bootstrap-github-url" name="github_url" required
                                   placeholder="https://github.com/username/repo" />
                            <small>The repository will be cloned/initialized. Can be empty or contain LICENSE/README.</small>
                        </div>

                        <div class="form-group">
                            <label for="bootstrap-name">Project Name *</label>
                            <input type="text" id="bootstrap-name" name="name" required
                                   placeholder="My Awesome Project" />
                        </div>

                        <div class="form-group">
                            <label for="bootstrap-branch">Branch</label>
                            <input type="text" id="bootstrap-branch" name="branch" value="main" required />
                        </div>

                        <div class="form-group">
                            <label>Product Requirements Document (PRD) *</label>
                            <div class="prd-input-tabs">
                                <button type="button" class="prd-tab active" data-tab="text" onclick="switchPRDTab('text')">Enter Text</button>
                                <button type="button" class="prd-tab" data-tab="file" onclick="switchPRDTab('file')">Upload File</button>
                            </div>

                            <div id="prd-text-input" class="prd-input-panel active">
                                <textarea id="bootstrap-prd-text" name="prd_text" rows="12"
                                          placeholder="Describe your project requirements...

Example:
# Project: Task Manager App

## Overview
A simple task management web application for personal productivity.

## Features
- User authentication (email/password)
- Create, edit, delete tasks
- Mark tasks as complete
- Filter by status (all, active, completed)
- Responsive design for mobile and desktop

## Technical Requirements
- Frontend: React with TypeScript
- Backend: Node.js with Express
- Database: PostgreSQL
- Authentication: JWT tokens"></textarea>
                            </div>

                            <div id="prd-file-input" class="prd-input-panel">
                                <input type="file" id="bootstrap-prd-file" name="prd_file"
                                       accept=".md,.txt,.pdf,.doc,.docx" />
                                <small>Supported formats: Markdown (.md), Text (.txt), PDF, Word docs</small>
                            </div>
                        </div>

                        <div class="form-actions">
                            <button type="button" class="secondary" onclick="closeBootstrapModal()">Cancel</button>
                            <button type="submit" class="primary">Create Project</button>
                        </div>
                    </form>

                    <div id="bootstrap-status" class="bootstrap-status" style="display: none;">
                        <div class="status-icon">⏳</div>
                        <div class="status-text">Bootstrapping project...</div>
                        <div class="status-details"></div>
                    </div>
                </div>
            </div>
        </div>
    `;

    // Add modal to page
    const existingModal = document.getElementById('bootstrap-modal');
    if (existingModal) {
        existingModal.remove();
    }

    document.body.insertAdjacentHTML('beforeend', modalHTML);

    // Setup form submission
    const form = document.getElementById('bootstrap-form');
    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        await submitBootstrapForm(form);
    });

    // Show modal
    setTimeout(() => {
        document.getElementById('bootstrap-modal').classList.add('show');
    }, 10);
}

function closeBootstrapModal() {
    const modal = document.getElementById('bootstrap-modal');
    if (modal) {
        modal.classList.remove('show');
        setTimeout(() => modal.remove(), 300);
    }
}

function switchPRDTab(tab) {
    // Update tab buttons
    document.querySelectorAll('.prd-tab').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.tab === tab);
    });

    // Update panels
    document.getElementById('prd-text-input').classList.toggle('active', tab === 'text');
    document.getElementById('prd-file-input').classList.toggle('active', tab === 'file');

    // Clear the inactive input
    if (tab === 'text') {
        document.getElementById('bootstrap-prd-file').value = '';
    } else {
        document.getElementById('bootstrap-prd-text').value = '';
    }
}

async function submitBootstrapForm(form) {
    const formData = new FormData(form);

    // Get PRD content
    let prdText = formData.get('prd_text') || '';
    const prdFile = document.getElementById('bootstrap-prd-file').files[0];

    // Validate
    if (!prdText && !prdFile) {
        showToast('Please provide a PRD (either text or file)', 'error');
        return;
    }

    // If file is provided, read it
    if (prdFile) {
        prdText = await readFileAsText(prdFile);
    }

    // Build request payload
    const payload = {
        github_url: formData.get('github_url'),
        name: formData.get('name'),
        branch: formData.get('branch') || 'main',
        prd_text: prdText
    };

    // Show status
    document.getElementById('bootstrap-form').style.display = 'none';
    const statusDiv = document.getElementById('bootstrap-status');
    statusDiv.style.display = 'block';

    try {
        updateBootstrapStatus('Validating inputs...', '⏳');

        const response = await apiCall('/projects/bootstrap', {
            method: 'POST',
            body: JSON.stringify(payload)
        });

        updateBootstrapStatus('Project created!', '✅');

        // Update details
        const details = document.querySelector('.status-details');
        details.innerHTML = `
            <p><strong>Project ID:</strong> ${response.project_id}</p>
            <p><strong>Status:</strong> ${response.status}</p>
            <p><strong>Initial Bead:</strong> ${response.initial_bead_id || 'Creating...'}</p>
            <p>The Project Manager will now expand your PRD and create work breakdown.</p>
        `;

        // Auto-close after delay
        setTimeout(() => {
            closeBootstrapModal();
            showToast('Project bootstrapped successfully!', 'success');

            // Reload projects list
            if (typeof loadProjects === 'function') {
                loadProjects();
            }
            if (typeof render === 'function') {
                render();
            }
        }, 3000);

    } catch (error) {
        updateBootstrapStatus('Bootstrap failed', '❌');
        const details = document.querySelector('.status-details');
        details.innerHTML = `<p class="error">${error.message || 'Unknown error occurred'}</p>`;

        // Re-show form
        setTimeout(() => {
            document.getElementById('bootstrap-form').style.display = 'block';
            statusDiv.style.display = 'none';
        }, 3000);
    }
}

function updateBootstrapStatus(text, icon) {
    const statusDiv = document.getElementById('bootstrap-status');
    statusDiv.querySelector('.status-icon').textContent = icon;
    statusDiv.querySelector('.status-text').textContent = text;
}

function readFileAsText(file) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = (e) => resolve(e.target.result);
        reader.onerror = (e) => reject(new Error('Failed to read file'));
        reader.readAsText(file);
    });
}

// Modify the existing "Add Project" button to show options
function showProjectOptionsMenu() {
    const menuHTML = `
        <div class="context-menu" id="project-options-menu" style="position: fixed; z-index: 10000;">
            <div class="context-menu-item" onclick="showCreateProjectModal(); closeProjectOptionsMenu();">
                <span>📂</span> Join Existing Project
            </div>
            <div class="context-menu-item" onclick="showBootstrapProjectModal(); closeProjectOptionsMenu();">
                <span>✨</span> New Project (Bootstrap)
            </div>
        </div>
    `;

    // Remove existing menu
    const existing = document.getElementById('project-options-menu');
    if (existing) {
        existing.remove();
        return;
    }

    // Add menu
    document.body.insertAdjacentHTML('beforeend', menuHTML);

    // Position near button
    const button = document.getElementById('project-view-add');
    if (button) {
        const rect = button.getBoundingClientRect();
        const menu = document.getElementById('project-options-menu');
        menu.style.top = (rect.bottom + 5) + 'px';
        menu.style.left = rect.left + 'px';
    }

    // Close on click outside
    setTimeout(() => {
        document.addEventListener('click', closeProjectOptionsMenu);
    }, 100);
}

function closeProjectOptionsMenu() {
    const menu = document.getElementById('project-options-menu');
    if (menu) {
        menu.remove();
        document.removeEventListener('click', closeProjectOptionsMenu);
    }
}
