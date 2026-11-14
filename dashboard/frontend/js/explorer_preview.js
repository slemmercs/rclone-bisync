/**
 * Dual-Pane File Explorer Preview & Download Module for Bisync Hub Dashboard
 * Handles file preview modal (text, images, PDFs) and streaming file downloads.
 */

window.ExplorerPreviewModule = {
    isItemPreviewable(item) {
        if (!item || item.IsDir) return false;
        const name = (item.Name || '').toLowerCase();
        const previewExtensions = [
            '.txt', '.md', '.markdown', '.log', '.conf', '.ini', '.env', '.csv',
            '.json', '.yaml', '.yml', '.xml', '.html', '.htm', '.css', '.js', '.ts', '.go', '.sh', '.bash', '.py', '.rs', '.sql',
            '.png', '.jpg', '.jpeg', '.gif', '.webp', '.svg', '.bmp', '.ico', '.pdf'
        ];
        return previewExtensions.some(ext => name.endsWith(ext));
    },

    async previewFile(paneKey, item) {
        if (!item || item.IsDir) return;
        const pane = this.explorer[paneKey];
        const remote = pane.target || (paneKey === 'left' ? 'local' : '');
        let path = item.Path || item.Name;
        if (pane.path && !path.startsWith(pane.path)) {
            path = `${pane.path.replace(/\/+$/, '')}/${item.Name}`;
        }
        const profileId = this.activeProfile ? this.activeProfile.id : "";

        this.filePreviewModal = {
            show: true,
            loading: true,
            error: null,
            remote,
            path,
            name: item.Name,
            preview: null
        };

        try {
            const query = new URLSearchParams({
                remote,
                path,
                profile_id: profileId
            });
            const res = await fetch(`/api/explorer/preview?${query.toString()}`);
            if (!res.ok) {
                const errData = await res.json().catch(() => ({}));
                throw new Error(errData.error || `HTTP ${res.status}`);
            }
            const data = await res.json();
            this.filePreviewModal.preview = data;
            this.filePreviewModal.loading = false;
        } catch (err) {
            this.filePreviewModal.loading = false;
            this.filePreviewModal.error = err.message || "Failed to load preview";
        }
    },

    closePreviewModal() {
        this.filePreviewModal.show = false;
        this.filePreviewModal.preview = null;
        this.filePreviewModal.error = null;
    },

    downloadFile(paneKey, item) {
        if (!item || item.IsDir) return;
        const pane = this.explorer[paneKey];
        const remote = pane.target || (paneKey === 'left' ? 'local' : '');
        let path = item.Path || item.Name;
        if (pane.path && !path.startsWith(pane.path)) {
            path = `${pane.path.replace(/\/+$/, '')}/${item.Name}`;
        }
        const profileId = this.activeProfile ? this.activeProfile.id : "";
        const query = new URLSearchParams({
            remote,
            path,
            profile_id: profileId
        });
        const url = `/api/explorer/download?${query.toString()}`;
        const a = document.createElement('a');
        a.href = url;
        a.download = item.Name || 'download';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
    },
    downloadPreviewFile() {
        if (!this.filePreviewModal.remote || !this.filePreviewModal.path) return;
        const profileId = this.activeProfile ? this.activeProfile.id : "";
        const query = new URLSearchParams({
            remote: this.filePreviewModal.remote,
            path: this.filePreviewModal.path,
            profile_id: profileId
        });
        window.open(`/api/explorer/download?${query.toString()}`, "_blank");
    },


    openDownloadFolder(paneKey) {
        const pane = this.explorer[paneKey];
        const remote = pane.target || (paneKey === 'left' ? 'local' : '');
        const path = pane.path || '';
        const profileId = this.activeProfile ? this.activeProfile.id : "";
        const query = new URLSearchParams({
            remote,
            path,
            profile_id: profileId
        });
        const url = `/api/explorer/download?${query.toString()}`;
        const a = document.createElement('a');
        a.href = url;
        const folderName = path ? path.split('/').filter(Boolean).pop() : remote;
        a.download = `${folderName || 'folder'}.zip`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
    }
};
