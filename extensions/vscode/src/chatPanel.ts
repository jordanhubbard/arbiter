import * as vscode from 'vscode';
import { LoomClient, Message } from './client';

export class ChatPanelProvider implements vscode.WebviewViewProvider {
    private _view?: vscode.WebviewView;
    private _conversationHistory: Message[] = [];

    constructor(
        private readonly _extensionUri: vscode.Uri,
        private readonly _client: LoomClient
    ) {}

    public resolveWebviewView(
        webviewView: vscode.WebviewView,
        context: vscode.WebviewViewResolveContext,
        _token: vscode.CancellationToken
    ) {
        this._view = webviewView;

        webviewView.webview.options = {
            enableScripts: true,
            localResourceRoots: [this._extensionUri]
        };

        webviewView.webview.html = this._getHtmlForWebview(webviewView.webview);

        webviewView.webview.onDidReceiveMessage(async data => {
            switch (data.type) {
                case 'sendMessage':
                    await this.handleUserMessage(data.message);
                    break;
                case 'clearHistory':
                    this._conversationHistory = [];
                    this._view?.webview.postMessage({ type: 'historyCleared' });
                    break;
            }
        });
    }

    public sendMessage(question: string, codeContext?: string) {
        let message = question;
        
        if (codeContext) {
            const config = vscode.workspace.getConfiguration('loom');
            const autoContext = config.get('autoContext', true);
            
            if (autoContext) {
                message = `${question}\n\n\`\`\`\n${codeContext}\n\`\`\``;
            }
        }

        this._view?.webview.postMessage({
            type: 'addUserMessage',
            message: message
        });

        this.handleUserMessage(message);
    }

    private async handleUserMessage(userMessage: string) {
        try {
            // Add user message to history
            this._conversationHistory.push({
                role: 'user',
                content: userMessage
            });

            // Show loading indicator
            this._view?.webview.postMessage({ type: 'showLoading' });

            // Get response from Loom
            const response = await this._client.sendMessage(this._conversationHistory);

            if (response.choices && response.choices.length > 0) {
                const assistantMessage = response.choices[0].message;
                
                // Add assistant message to history
                this._conversationHistory.push(assistantMessage);

                // Send response to webview
                this._view?.webview.postMessage({
                    type: 'addAssistantMessage',
                    message: assistantMessage.content
                });
            } else {
                throw new Error('No response from Loom');
            }
        } catch (error: any) {
            vscode.window.showErrorMessage(`Loom error: ${error.message}`);
            this._view?.webview.postMessage({
                type: 'showError',
                message: error.message
            });
        } finally {
            this._view?.webview.postMessage({ type: 'hideLoading' });
        }
    }

    private _getHtmlForWebview(webview: vscode.Webview): string {
        return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Loom Chat</title>
    <style>
        body {
            padding: 10px;
            color: var(--vscode-foreground);
            font-family: var(--vscode-font-family);
            font-size: var(--vscode-font-size);
        }
        #chat-container {
            display: flex;
            flex-direction: column;
            height: calc(100vh - 100px);
        }
        #messages {
            flex: 1;
            overflow-y: auto;
            padding: 10px;
            margin-bottom: 10px;
        }
        .message {
            margin-bottom: 15px;
            padding: 10px;
            border-radius: 5px;
        }
        .user-message {
            background-color: var(--vscode-input-background);
            border-left: 3px solid var(--vscode-focusBorder);
        }
        .assistant-message {
            background-color: var(--vscode-editor-background);
            border-left: 3px solid var(--vscode-charts-green);
        }
        .error-message {
            background-color: var(--vscode-inputValidation-errorBackground);
            border-left: 3px solid var(--vscode-inputValidation-errorBorder);
            color: var(--vscode-inputValidation-errorForeground);
        }
        .message-role {
            font-weight: bold;
            margin-bottom: 5px;
            opacity: 0.8;
        }
        .message-content {
            white-space: pre-wrap;
            word-wrap: break-word;
        }
        .message-content code {
            background-color: var(--vscode-textCodeBlock-background);
            padding: 2px 4px;
            border-radius: 3px;
            font-family: var(--vscode-editor-font-family);
        }
        .message-content pre {
            background-color: var(--vscode-textCodeBlock-background);
            padding: 10px;
            border-radius: 5px;
            overflow-x: auto;
        }
        #input-container {
            display: flex;
            gap: 5px;
        }
        #message-input {
            flex: 1;
            padding: 8px;
            background-color: var(--vscode-input-background);
            color: var(--vscode-input-foreground);
            border: 1px solid var(--vscode-input-border);
            border-radius: 3px;
        }
        button {
            padding: 8px 15px;
            background-color: var(--vscode-button-background);
            color: var(--vscode-button-foreground);
            border: none;
            border-radius: 3px;
            cursor: pointer;
        }
        button:hover {
            background-color: var(--vscode-button-hoverBackground);
        }
        .loading {
            display: none;
            text-align: center;
            padding: 10px;
            opacity: 0.7;
        }
        .loading.active {
            display: block;
        }
        #clear-btn {
            background-color: var(--vscode-button-secondaryBackground);
            color: var(--vscode-button-secondaryForeground);
        }
    </style>
</head>
<body>
    <div id="chat-container">
        <div id="messages"></div>
        <div class="loading" id="loading">Thinking...</div>
        <div id="input-container">
            <textarea id="message-input" rows="3" placeholder="Ask Loom anything..."></textarea>
            <div style="display: flex; flex-direction: column; gap: 5px;">
                <button id="send-btn">Send</button>
                <button id="clear-btn">Clear</button>
            </div>
        </div>
    </div>

    <script>
        const vscode = acquireVsCodeApi();
        const messagesDiv = document.getElementById('messages');
        const messageInput = document.getElementById('message-input');
        const sendBtn = document.getElementById('send-btn');
        const clearBtn = document.getElementById('clear-btn');
        const loading = document.getElementById('loading');

        function addMessage(role, content, isError = false) {
            const messageDiv = document.createElement('div');
            messageDiv.className = \`message \${isError ? 'error-message' : role + '-message'}\`;
            
            const roleDiv = document.createElement('div');
            roleDiv.className = 'message-role';
            roleDiv.textContent = isError ? 'Error' : (role === 'user' ? 'You' : 'Loom');
            
            const contentDiv = document.createElement('div');
            contentDiv.className = 'message-content';
            contentDiv.textContent = content;
            
            messageDiv.appendChild(roleDiv);
            messageDiv.appendChild(contentDiv);
            messagesDiv.appendChild(messageDiv);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }

        function sendMessage() {
            const message = messageInput.value.trim();
            if (!message) return;

            addMessage('user', message);
            messageInput.value = '';

            vscode.postMessage({
                type: 'sendMessage',
                message: message
            });
        }

        sendBtn.addEventListener('click', sendMessage);
        clearBtn.addEventListener('click', () => {
            if (confirm('Clear conversation history?')) {
                messagesDiv.innerHTML = '';
                vscode.postMessage({ type: 'clearHistory' });
            }
        });

        messageInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && e.ctrlKey) {
                sendMessage();
            }
        });

        window.addEventListener('message', event => {
            const message = event.data;
            
            switch (message.type) {
                case 'addUserMessage':
                    // Message already added by sendMessage
                    break;
                case 'addAssistantMessage':
                    addMessage('assistant', message.message);
                    break;
                case 'showError':
                    addMessage('error', message.message, true);
                    break;
                case 'showLoading':
                    loading.classList.add('active');
                    break;
                case 'hideLoading':
                    loading.classList.remove('active');
                    break;
                case 'historyCleared':
                    // Already cleared by button handler
                    break;
            }
        });
    </script>
</body>
</html>`;
    }
}
