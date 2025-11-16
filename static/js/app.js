// App state
let currentUser = null;
let allStandards = [];
let myStandards = [];
let categories = [];
let currentPage = 1;

// Initialize app
async function init() {
    const token = API.getToken();
    
    if (token) {
        try {
            currentUser = await API.getMe();
            showMainScreen();
            await loadMyStandards();
        } catch (error) {
            console.error('Failed to load user:', error);
            API.clearToken();
            showAuthScreen();
        }
    } else {
        showAuthScreen();
    }
    
    setupEventListeners();
    checkOnlineStatus();
}

// Setup event listeners
function setupEventListeners() {
    // Auth tabs
    document.querySelectorAll('.tab').forEach(tab => {
        tab.addEventListener('click', (e) => {
            document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
            document.querySelectorAll('.auth-form').forEach(f => f.classList.remove('active'));
            e.target.classList.add('active');
            document.getElementById(`${e.target.dataset.tab}-form`).classList.add('active');
        });
    });
    
    // Auth forms
    document.getElementById('login-form').addEventListener('submit', handleLogin);
    document.getElementById('register-form').addEventListener('submit', handleRegister);
    
    // Logout
    document.getElementById('logout-btn').addEventListener('click', handleLogout);
    
    // Navigation
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const view = e.target.dataset.view;
            switchView(view);
        });
    });
    
    // Search and filter
    document.getElementById('search-input').addEventListener('input', debounce(loadAllStandards, 300));
    document.getElementById('style-filter').addEventListener('change', loadAllStandards);
    
    // Add standard/category buttons
    document.getElementById('add-standard-btn').addEventListener('click', () => {
        document.getElementById('add-standard-modal').classList.remove('hidden');
    });
    document.getElementById('add-category-btn').addEventListener('click', () => {
        document.getElementById('add-category-modal').classList.remove('hidden');
    });
    
    // Modal forms
    document.getElementById('add-standard-form').addEventListener('submit', handleAddStandard);
    document.getElementById('add-category-form').addEventListener('submit', handleAddCategory);
    
    // Modal cancel buttons
    document.querySelectorAll('.modal .cancel').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.target.closest('.modal').classList.add('hidden');
        });
    });
}

// Auth handlers
async function handleLogin(e) {
    e.preventDefault();
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;
    const errorEl = document.getElementById('login-error');
    
    try {
        debugPrint('handleLogin called for user:', username);
        currentUser = await API.login(username, password);
        debugPrint('Login successful, user:', currentUser);
        showMainScreen();
        await loadMyStandards();
    } catch (error) {
        debugPrint('Login failed with error:', error);
        errorEl.textContent = error.message;
    }
}

async function handleRegister(e) {
    e.preventDefault();
    const name = document.getElementById('register-name').value;
    const username = document.getElementById('register-username').value;
    const password = document.getElementById('register-password').value;
    const errorEl = document.getElementById('register-error');
    
    try {
        currentUser = await API.register(username, name, password);
        showMainScreen();
        await loadMyStandards();
    } catch (error) {
        errorEl.textContent = error.message;
    }
}

async function handleLogout() {
    try {
        await API.logout();
    } catch (error) {
        console.error('Logout error:', error);
    }
    currentUser = null;
    showAuthScreen();
}

// Screen switching
function showAuthScreen() {
    document.getElementById('auth-screen').classList.remove('hidden');
    document.getElementById('main-screen').classList.add('hidden');
}

function showMainScreen() {
    document.getElementById('auth-screen').classList.add('hidden');
    document.getElementById('main-screen').classList.remove('hidden');
    document.getElementById('user-name').textContent = currentUser.name;
    
    // Show/hide admin controls
    const adminElements = document.querySelectorAll('.admin-only');
    adminElements.forEach(el => {
        if (currentUser.is_admin) {
            el.classList.remove('hidden');
        } else {
            el.classList.add('hidden');
        }
    });
    
    // Load pending standards if admin
    if (currentUser.is_admin) {
        loadPendingStandards();
    }
}

function switchView(viewName) {
    document.querySelectorAll('.nav-btn').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('.view').forEach(view => view.classList.add('hidden'));
    
    document.querySelector(`[data-view="${viewName}"]`).classList.add('active');
    document.getElementById(`${viewName}-view`).classList.remove('hidden');
    
    // Load data for the view
    if (viewName === 'all-standards') {
        loadAllStandards();
    } else if (viewName === 'categories') {
        loadCategories();
    } else if (viewName === 'pending') {
        loadPendingStandards(1);
    }
}

// Load data
async function loadMyStandards() {
    try {
        const data = await API.getMyStandards();
        myStandards = data.standards || [];
        renderMyStandards(data.grouped || {});
    } catch (error) {
        console.error('Failed to load my standards:', error);
    }
}

async function loadAllStandards(page = 1) {
    const search = document.getElementById('search-input').value;
    const style = document.getElementById('style-filter').value;
    
    try {
        const data = await API.getStandards(page, 100, search, style);
        allStandards = data.standards || [];
        currentPage = page;
        renderAllStandards(data);
    } catch (error) {
        console.error('Failed to load standards:', error);
    }
}

async function loadCategories() {
    try {
        categories = await API.getCategories();
        renderCategories();
    } catch (error) {
        console.error('Failed to load categories:', error);
    }
}

async function loadPendingStandards(page = 1) {
    if (!currentUser || !currentUser.is_admin) return;
    
    try {
        const data = await API.getPendingStandards(page, 100);
        renderPendingStandards(data);
    } catch (error) {
        console.error('Failed to load pending standards:', error);
    }
}

// Render functions
function renderMyStandards(grouped) {
    const container = document.getElementById('standards-by-category');
    container.innerHTML = '';
    
    if (Object.keys(grouped).length === 0) {
        container.innerHTML = '<p style="color: white;">No standards yet. Add some from the "All Standards" tab!</p>';
        return;
    }
    
    for (const [categoryName, standards] of Object.entries(grouped)) {
        const categoryDiv = document.createElement('div');
        categoryDiv.className = 'category-group';
        
        const category = standards[0].category;
        const colorHtml = category ? `<span class="category-color" style="background: ${category.color}"></span>` : '';
        
        categoryDiv.innerHTML = `
            <h3>${colorHtml}${categoryName}</h3>
            ${standards.map(us => `
                <div class="standard-item">
                    <div class="standard-info">
                        <h4>${us.jazz_standard.title}</h4>
                        <p>${us.jazz_standard.composer} - ${us.jazz_standard.style}</p>
                        ${us.notes ? `<p><em>${us.notes}</em></p>` : ''}
                    </div>
                    <div class="standard-actions">
                        <button class="btn-remove" onclick="removeStandard(${us.jazz_standard_id})">Remove</button>
                    </div>
                </div>
            `).join('')}
        `;
        
        container.appendChild(categoryDiv);
    }
}

function renderAllStandards(data) {
    const container = document.getElementById('all-standards-list');
    const standards = data.standards || [];
    
    if (standards.length === 0) {
        container.innerHTML = '<p style="color: white;">No standards found.</p>';
        return;
    }
    
    container.innerHTML = standards.map(standard => {
        const isKnown = myStandards.some(us => us.jazz_standard_id === standard.id);
        const isPending = standard.status === 'pending';
        const isRejected = standard.status === 'rejected';
        
        let statusBadge = '';
        if (isPending) {
            statusBadge = '<span style="color: #ffc107; font-weight: bold;">⏳ Pending Approval</span>';
        } else if (isRejected) {
            statusBadge = '<span style="color: #dc3545; font-weight: bold;">❌ Rejected</span>';
        }
        
        const actionBtn = isKnown
            ? `<button class="btn-remove" onclick="removeStandard(${standard.id})">Remove</button>`
            : `<button class="btn-know" onclick="addStandard(${standard.id})">I Know This</button>`;
        
        return `
            <div class="standard-item">
                <div class="standard-info">
                    <h4>${standard.title} ${statusBadge}</h4>
                    <p>${standard.composer} - ${standard.style}</p>
                    ${standard.additional_note ? `<p><em>${standard.additional_note}</em></p>` : ''}
                    ${standard.creator ? `<p style="font-size: 0.85em; color: #aaa;">Submitted by: ${standard.creator.name}</p>` : ''}
                </div>
                <div class="standard-actions">
                    ${!isPending && !isRejected ? actionBtn : ''}
                    ${currentUser.is_admin ? `<button class="btn-remove" onclick="deleteStandard(${standard.id})">Delete</button>` : ''}
                </div>
            </div>
        `;
    }).join('');
    
    renderPagination(data);
}

function renderPagination(data) {
    const container = document.getElementById('pagination');
    const totalPages = Math.ceil(data.total / data.limit);
    
    if (totalPages <= 1) {
        container.innerHTML = '';
        return;
    }
    
    container.innerHTML = `
        <button ${data.page === 1 ? 'disabled' : ''} onclick="loadAllStandards(${data.page - 1})">Previous</button>
        <span>Page ${data.page} of ${totalPages}</span>
        <button ${data.page === totalPages ? 'disabled' : ''} onclick="loadAllStandards(${data.page + 1})">Next</button>
    `;
}

function renderPendingStandards(data) {
    const container = document.getElementById('pending-standards-list');
    if (!container) return;
    
    const standards = data.standards || [];
    
    if (standards.length === 0) {
        container.innerHTML = '<p style="color: white;">No pending standards.</p>';
        return;
    }
    
    container.innerHTML = standards.map(standard => `
        <div class="standard-item pending-item">
            <div class="standard-info">
                <h4>${standard.title}</h4>
                <p>${standard.composer} - ${standard.style}</p>
                ${standard.additional_note ? `<p><em>${standard.additional_note}</em></p>` : ''}
                <p style="font-size: 0.85em; color: #aaa;">
                    Submitted by: ${standard.creator ? standard.creator.name : 'Unknown'} 
                    on ${new Date(standard.created_at).toLocaleDateString()}
                </p>
            </div>
            <div class="standard-actions">
                <button class="btn-approve" onclick="approveStandard(${standard.id})">✓ Approve</button>
                <button class="btn-reject" onclick="rejectStandard(${standard.id})">✗ Reject</button>
                <button class="btn-remove" onclick="deleteStandard(${standard.id})">Delete</button>
            </div>
        </div>
    `).join('');
}

function renderCategories() {
    const container = document.getElementById('categories-list');
    
    if (categories.length === 0) {
        container.innerHTML = '<p style="color: white;">No categories yet. Create one!</p>';
        return;
    }
    
    container.innerHTML = categories.map(cat => `
        <div class="category-item">
            <h3>
                <span class="category-color" style="background: ${cat.color}"></span>
                ${cat.name}
            </h3>
            <button class="btn-remove" onclick="deleteCategory(${cat.id})">Delete</button>
        </div>
    `).join('');
}

// Action handlers
async function addStandard(standardId) {
    try {
        await API.addToMyStandards(standardId);
        await loadMyStandards();
        if (document.getElementById('all-standards-view').classList.contains('active')) {
            await loadAllStandards(currentPage);
        }
    } catch (error) {
        alert(error.message);
    }
}

async function removeStandard(standardId) {
    if (!confirm('Remove this standard from your list?')) return;
    
    try {
        await API.removeFromMyStandards(standardId);
        await loadMyStandards();
        if (document.getElementById('all-standards-view').classList.contains('active')) {
            await loadAllStandards(currentPage);
        }
    } catch (error) {
        alert(error.message);
    }
}

async function deleteStandard(standardId) {
    if (!confirm('Delete this standard permanently?')) return;
    
    try {
        await API.deleteStandard(standardId);
        await loadAllStandards(currentPage);
    } catch (error) {
        alert(error.message);
    }
}

async function handleAddStandard(e) {
    e.preventDefault();
    
    const title = document.getElementById('new-standard-title').value;
    const composer = document.getElementById('new-standard-composer').value;
    const style = document.getElementById('new-standard-style').value;
    const note = document.getElementById('new-standard-note').value;
    
    try {
        const response = await API.createStandard(title, composer, style, note);
        document.getElementById('add-standard-modal').classList.add('hidden');
        e.target.reset();
        
        // Show success message
        if (response.message) {
            alert(response.message);
        }
        
        await loadAllStandards(currentPage);
        
        // If admin, also reload pending standards
        if (currentUser.is_admin) {
            loadPendingStandards();
        }
    } catch (error) {
        alert(error.message);
    }
}

async function handleAddCategory(e) {
    e.preventDefault();
    
    const name = document.getElementById('new-category-name').value;
    const color = document.getElementById('new-category-color').value;
    
    try {
        await API.createCategory(name, color);
        document.getElementById('add-category-modal').classList.add('hidden');
        e.target.reset();
        await loadCategories();
    } catch (error) {
        alert(error.message);
    }
}

async function deleteCategory(categoryId) {
    if (!confirm('Delete this category? Standards will be uncategorized.')) return;
    
    try {
        await API.deleteCategory(categoryId);
        await loadCategories();
        await loadMyStandards();
    } catch (error) {
        alert(error.message);
    }
}

// Utility functions
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

function checkOnlineStatus() {
    const indicator = document.getElementById('offline-indicator');
    
    function updateStatus() {
        if (navigator.onLine) {
            indicator.classList.add('hidden');
        } else {
            indicator.classList.remove('hidden');
        }
    }
    
    window.addEventListener('online', updateStatus);
    window.addEventListener('offline', updateStatus);
    updateStatus();
}

// Admin functions for approving/rejecting standards
async function approveStandard(id) {
    if (!confirm('Approve this standard?')) return;
    
    try {
        await API.approveStandard(id);
        await loadPendingStandards();
        await loadAllStandards(currentPage);
        alert('Standard approved successfully!');
    } catch (error) {
        alert(error.message);
    }
}

async function rejectStandard(id) {
    if (!confirm('Reject this standard? It will be marked as rejected.')) return;
    
    try {
        await API.rejectStandard(id);
        await loadPendingStandards();
        await loadAllStandards(currentPage);
        alert('Standard rejected.');
    } catch (error) {
        alert(error.message);
    }
}

// Initialize on load
document.addEventListener('DOMContentLoaded', init);
