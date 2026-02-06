import axios, { AxiosInstance } from 'axios';

export interface Message {
    role: 'user' | 'assistant' | 'system';
    content: string;
}

export interface ChatCompletionRequest {
    messages: Message[];
    model?: string;
    temperature?: number;
    max_tokens?: number;
}

export interface ChatCompletionResponse {
    id: string;
    choices: Array<{
        message: Message;
        finish_reason: string;
    }>;
    usage?: {
        prompt_tokens: number;
        completion_tokens: number;
        total_tokens: number;
    };
}

export class LoomClient {
    private client: AxiosInstance;
    private apiKey: string;

    constructor(apiEndpoint: string, apiKey: string) {
        this.apiKey = apiKey;
        this.client = axios.create({
            baseURL: apiEndpoint,
            timeout: 60000,
            headers: {
                'Content-Type': 'application/json',
                ...(apiKey && { 'Authorization': `Bearer ${apiKey}` })
            }
        });
    }

    updateConfig(apiEndpoint: string, apiKey: string) {
        this.apiKey = apiKey;
        this.client = axios.create({
            baseURL: apiEndpoint,
            timeout: 60000,
            headers: {
                'Content-Type': 'application/json',
                ...(apiKey && { 'Authorization': `Bearer ${apiKey}` })
            }
        });
    }

    async sendMessage(messages: Message[], model?: string): Promise<ChatCompletionResponse> {
        try {
            const request: ChatCompletionRequest = {
                messages,
                model: model || 'default',
                temperature: 0.7,
                max_tokens: 2000
            };

            const response = await this.client.post<ChatCompletionResponse>(
                '/api/v1/chat/completions',
                request
            );

            return response.data;
        } catch (error: any) {
            if (error.response) {
                throw new Error(`Loom API error: ${error.response.status} - ${error.response.data?.message || error.response.statusText}`);
            } else if (error.request) {
                throw new Error('Loom API is not reachable. Please check your connection and API endpoint.');
            } else {
                throw new Error(`Request error: ${error.message}`);
            }
        }
    }

    async healthCheck(): Promise<boolean> {
        try {
            await this.client.get('/health');
            return true;
        } catch {
            return false;
        }
    }
}
