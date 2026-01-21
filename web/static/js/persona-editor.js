// persona-editor.js - Web-based Persona Editor

class PersonaEditor {
    constructor() {
        this.currentPersona = null;
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadPersonas();
    }

    setupEventListeners() {
        document.getElementById('new-persona-btn')?.addEventListener('click', () => this.newPersona());
        document.getElementById('save-persona-btn')?.addEventListener('click', () => this.savePersona());
        document.getElementById('delete-persona-btn')?.addEventListener('click', () => this.deletePersona());
        document.getElementById('persona-name')?.addEventListener('input', () => this.updatePreview());
        document.getElementById('persona-role')?.addEventListener('input', () => this.updatePreview());
        document.getElementById('persona-instructions')?.addEventListener('input', () => this.updatePreview());
        document.getElementById('persona-capabilities')?.addEventListener('input', () => this.updatePreview());
    }

    async loadPersonas() {
        try {
            const response = await fetch('/api/v1/personas');
            if (!response.ok) throw new Error('Failed to load personas');
            
            const personas = await response.json();
            this.renderPersonaList(personas);
        } catch (error) {
            this.showError('Failed to load personas: ' + error.message);
        }
    }

    renderPersonaList(personas) {
        const list = document.getElementById('persona-list');
        if (!list) return;

        list.innerHTML = personas.map(persona => `
            <div class="persona-item" data-id="${persona.id}" onclick="personaEditor.loadPersona('${persona.id}')">
                <div class="persona-item-name">${persona.name || persona.id}</div>
                <div class="persona-item-role">${persona.role || 'N/A'}</div>
            </div>
        `).join('');
    }

    async loadPersona(id) {
        try {
            const response = await fetch(`/api/v1/personas/${id}`);
            if (!response.ok) throw new Error('Failed to load persona');
            
            this.currentPersona = await response.json();
            this.populateForm();
            this.updatePreview();
        } catch (error) {
            this.showError('Failed to load persona: ' + error.message);
        }
    }

    populateForm() {
        if (!this.currentPersona) return;

        document.getElementById('persona-id').value = this.currentPersona.id || '';
        document.getElementById('persona-name').value = this.currentPersona.name || '';
        document.getElementById('persona-role').value = this.currentPersona.role || '';
        document.getElementById('persona-instructions').value = this.currentPersona.instructions || '';
        document.getElementById('persona-capabilities').value = 
            Array.isArray(this.currentPersona.capabilities) 
                ? this.currentPersona.capabilities.join('\n')
                : '';
    }

    newPersona() {
        this.currentPersona = {
            id: '',
            name: '',
            role: '',
            instructions: '',
            capabilities: []
        };
        this.populateForm();
        this.updatePreview();
    }

    async savePersona() {
        try {
            const persona = this.getFormData();
            
            // Validate
            const validation = this.validatePersona(persona);
            if (!validation.valid) {
                this.showError('Validation failed: ' + validation.errors.join(', '));
                return;
            }

            // Save via API
            const method = this.currentPersona?.id ? 'PUT' : 'POST';
            const url = this.currentPersona?.id 
                ? `/api/v1/personas/${this.currentPersona.id}`
                : '/api/v1/personas';

            const response = await fetch(url, {
                method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(persona)
            });

            if (!response.ok) throw new Error('Failed to save persona');

            this.showSuccess('Persona saved successfully');
            this.loadPersonas();
        } catch (error) {
            this.showError('Failed to save: ' + error.message);
        }
    }

    async deletePersona() {
        if (!this.currentPersona?.id) {
            this.showError('No persona selected');
            return;
        }

        if (!confirm(`Delete persona "${this.currentPersona.name}"?`)) {
            return;
        }

        try {
            const response = await fetch(`/api/v1/personas/${this.currentPersona.id}`, {
                method: 'DELETE'
            });

            if (!response.ok) throw new Error('Failed to delete persona');

            this.showSuccess('Persona deleted');
            this.newPersona();
            this.loadPersonas();
        } catch (error) {
            this.showError('Failed to delete: ' + error.message);
        }
    }

    getFormData() {
        const capabilities = document.getElementById('persona-capabilities').value
            .split('\n')
            .map(line => line.trim())
            .filter(line => line);

        return {
            id: document.getElementById('persona-id').value.trim(),
            name: document.getElementById('persona-name').value.trim(),
            role: document.getElementById('persona-role').value.trim(),
            instructions: document.getElementById('persona-instructions').value.trim(),
            capabilities: capabilities
        };
    }

    validatePersona(persona) {
        const errors = [];

        if (!persona.id) errors.push('ID is required');
        if (!persona.name) errors.push('Name is required');
        if (!persona.role) errors.push('Role is required');
        if (!persona.instructions) errors.push('Instructions are required');
        if (persona.capabilities.length === 0) errors.push('At least one capability required');

        // ID validation
        if (persona.id && !/^[a-z0-9-]+$/.test(persona.id)) {
            errors.push('ID must contain only lowercase letters, numbers, and hyphens');
        }

        return {
            valid: errors.length === 0,
            errors
        };
    }

    updatePreview() {
        const persona = this.getFormData();
        const preview = document.getElementById('persona-preview');
        if (!preview) return;

        const markdown = this.generateMarkdown(persona);
        preview.innerHTML = `<pre>${this.escapeHtml(markdown)}</pre>`;
    }

    generateMarkdown(persona) {
        return `# ${persona.name || 'Untitled Persona'}

## Role
${persona.role || 'N/A'}

## Instructions
${persona.instructions || 'No instructions provided'}

## Capabilities
${persona.capabilities.map(cap => `- ${cap}`).join('\n') || 'No capabilities defined'}`;
    }

    escapeHtml(text) {
        const map = {
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            '"': '&quot;',
            "'": '&#039;'
        };
        return text.replace(/[&<>"']/g, m => map[m]);
    }

    showError(message) {
        this.showNotification(message, 'error');
    }

    showSuccess(message) {
        this.showNotification(message, 'success');
    }

    showNotification(message, type) {
        const notification = document.getElementById('notification');
        if (!notification) return;

        notification.textContent = message;
        notification.className = `notification ${type} show`;

        setTimeout(() => {
            notification.classList.remove('show');
        }, 3000);
    }
}

// Initialize editor when DOM is ready
let personaEditor;
document.addEventListener('DOMContentLoaded', () => {
    personaEditor = new PersonaEditor();
});
