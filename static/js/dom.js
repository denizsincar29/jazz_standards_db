// Delegated DOM handlers for dynamic elements
export default {
    attachDelegation() {
        document.body.addEventListener('click', (e) => {
            const btn = e.target.closest('button');
            if (!btn) return;
            const action = btn.getAttribute('data-action');
            const id = btn.getAttribute('data-id');

            if (!action) return;

            switch (action) {
                case 'add':
                    window.addStandard && window.addStandard(Number(id));
                    break;
                case 'remove':
                    window.removeStandard && window.removeStandard(Number(id));
                    break;
                case 'delete':
                    window.deleteStandard && window.deleteStandard(Number(id));
                    break;
                case 'approve':
                    window.approveStandard && window.approveStandard(Number(id));
                    break;
                case 'reject':
                    window.rejectStandard && window.rejectStandard(Number(id));
                    break;
                case 'delete-category':
                    window.deleteCategory && window.deleteCategory(Number(id));
                    break;
                case 'delete-passkey':
                    window.deletePassKey && window.deletePassKey(Number(id));
                    break;
                case 'paginate':
                    if (typeof id === 'string') {
                        // some pagination buttons use data-page instead
                    }
                    const page = btn.getAttribute('data-page');
                    if (page) window.loadAllStandards && window.loadAllStandards(Number(page));
                    break;
                case 'remove-my':
                    window.removeStandard && window.removeStandard(Number(id));
                    break;
                default:
                    break;
            }
        });
    }
};
