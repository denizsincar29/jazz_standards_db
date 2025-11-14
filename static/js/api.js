// API helper functions
const API = {
    baseURL: '/api',
    
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
        
        const response = await fetch(`${this.baseURL}${endpoint}`, {
            ...options,
            headers,
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.message || error.error || 'Request failed');
        }
        
        return response.json();
    },
    
    // Auth
    async register(username, name, password) {
        const data = await this.request('/register', {
            method: 'POST',
            body: JSON.stringify({ username, name, password }),
        });
        this.setToken(data.token);
        return data;
    },
    
    async login(username, password) {
        const data = await this.request('/login', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
        });
        this.setToken(data.token);
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
};
