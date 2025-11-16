// Debug mode - set to true to enable console logging
window.DEBUG = false;

// Debug print function
function debugPrint(...args) {
    if (window.DEBUG) {
        console.log('[DEBUG]', ...args);
    }
}

// API helper functions
const API = {
    baseURL: (() => {
        // Get base path from the <base> tag
        const baseElement = document.querySelector('base');
        const basePath = baseElement ? baseElement.getAttribute('href') : '/';
        // Remove trailing slash and add /api
        const normalizedBase = basePath.endsWith('/') ? basePath.slice(0, -1) : basePath;
        const apiBase = normalizedBase + '/api';
        debugPrint('API baseURL initialized:', apiBase);
        return apiBase;
    })(),
    
    getToken() {
        return localStorage.getItem('token');
    },
    
    setToken(token) {
        localStorage.setItem('token', token);
    },
    
    clearToken() {
        localStorage.removeItem('token');
    },
    
    async request(endpoint, options = {}) {
        const token = this.getToken();
        const headers = {
            'Content-Type': 'application/json',
            ...options.headers,
        };
        
        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }
        
        const url = `${this.baseURL}${endpoint}`;
        debugPrint('Making request to:', url, 'with options:', options);
        
        const response = await fetch(url, {
            ...options,
            headers,
            credentials: 'include', // Always include cookies
        });
        
        debugPrint('Response status:', response.status, response.statusText);
        
        if (!response.ok) {
            let errorMessage = 'Request failed';
            try {
                const error = await response.json();
                errorMessage = error.message || error.error || errorMessage;
                debugPrint('Error response:', error);
            } catch (parseError) {
                // If JSON parsing fails, try to get text
                const text = await response.text();
                debugPrint('Failed to parse error JSON. Response text:', text);
                errorMessage = `Request failed with status ${response.status}`;
            }
            throw new Error(errorMessage);
        }
        
        try {
            const data = await response.json();
            debugPrint('Response data:', data);
            return data;
        } catch (parseError) {
            const text = await response.text();
            debugPrint('Failed to parse response JSON. Response text:', text);
            throw new Error(`JSON.parse: unexpected character at line 1 column 1 of the JSON data`);
        }
    },
    
    // Auth
    async register(username, name, password) {
        const data = await this.request('/register', {
            method: 'POST',
            body: JSON.stringify({ username, name, password }),
            credentials: 'include', // Include cookies
        });
        // Token is set in cookie by server, extract from response if present
        if (data.token) {
            this.setToken(data.token);
        }
        return data;
    },
    
    async login(username, password) {
        debugPrint('Attempting login for user:', username);
        const data = await this.request('/login', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
            credentials: 'include', // Include cookies
        });
        // Token is set in cookie by server, extract from response if present
        if (data.token) {
            debugPrint('Login successful, token received');
            this.setToken(data.token);
        }
        return data;
    },
    
    async logout() {
        await this.request('/logout', { method: 'POST' });
        this.clearToken();
    },
    
    async getMe() {
        return this.request('/users/me');
    },
    
    // Standards
    async getStandards(page = 1, limit = 100, search = '', style = '') {
        const params = new URLSearchParams({ page, limit });
        if (search) params.append('search', search);
        if (style) params.append('style', style);
        return this.request(`/jazz_standards?${params}`);
    },
    
    async createStandard(title, composer, style, additional_note = '') {
        return this.request('/jazz_standards', {
            method: 'POST',
            body: JSON.stringify({ title, composer, style, additional_note }),
        });
    },
    
    async deleteStandard(id) {
        return this.request(`/jazz_standards/${id}`, { method: 'DELETE' });
    },
    
    // User Standards
    async getMyStandards() {
        return this.request('/users/me/standards');
    },
    
    async addToMyStandards(standardId, categoryId = null, notes = '') {
        return this.request(`/users/me/standards/${standardId}`, {
            method: 'POST',
            body: JSON.stringify({ category_id: categoryId, notes }),
        });
    },
    
    async updateMyStandard(standardId, categoryId = null, notes = '') {
        const body = {};
        if (categoryId !== null) body.category_id = categoryId;
        if (notes !== null) body.notes = notes;
        return this.request(`/users/me/standards/${standardId}`, {
            method: 'PUT',
            body: JSON.stringify(body),
        });
    },
    
    async removeFromMyStandards(standardId) {
        return this.request(`/users/me/standards/${standardId}`, { method: 'DELETE' });
    },
    
    // Categories
    async getCategories() {
        return this.request('/users/me/categories');
    },
    
    async createCategory(name, color) {
        return this.request('/users/me/categories', {
            method: 'POST',
            body: JSON.stringify({ name, color }),
        });
    },
    
    async updateCategory(id, name = null, color = null) {
        const body = {};
        if (name !== null) body.name = name;
        if (color !== null) body.color = color;
        return this.request(`/users/me/categories/${id}`, {
            method: 'PUT',
            body: JSON.stringify(body),
        });
    },
    
    async deleteCategory(id) {
        return this.request(`/users/me/categories/${id}`, { method: 'DELETE' });
    },
    
    // Admin: Pending Standards
    async getPendingStandards(page = 1, limit = 100) {
        const params = new URLSearchParams({ page, limit });
        return this.request(`/jazz_standards/pending?${params}`);
    },
    
    async approveStandard(id) {
        return this.request(`/jazz_standards/${id}/approve`, { method: 'POST' });
    },
    
    async rejectStandard(id) {
        return this.request(`/jazz_standards/${id}/reject`, { method: 'POST' });
    },
};
