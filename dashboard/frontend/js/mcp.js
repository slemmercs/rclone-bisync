/**
 * Model Context Protocol (MCP) Dashboard Inspector Module
 * Provides inspection state, client configuration snippets, and clipboard utilities.
 */
window.McpModule = {
    showMcpModal: false,
    mcpModalTab: "overview", // 'overview' | 'config' | 'tools'
    mcpModalCopiedConfig: false,
    mcpModalCopiedEndpoint: false,

    openMcpModal(tab = "overview") {
        this.mcpModalTab = tab;
        this.showMcpModal = true;
    },

    closeMcpModal() {
        this.showMcpModal = false;
        this.mcpModalCopiedConfig = false;
        this.mcpModalCopiedEndpoint = false;
    },

    get mcpEndpointUrl() {
        const m = this.telemetry?.mcp_status;
        if (!m || m.transport !== "sse") {
            return "stdio (direct subprocess pipe)";
        }
        const host = m.host || "127.0.0.1";
        const port = m.port || 8081;
        return `http://${host}:${port}/sse`;
    },

    get mcpClientSnippet() {
        const m = this.telemetry?.mcp_status;
        const host = m?.host || "127.0.0.1";
        const port = m?.port || 8081;
        const url = `http://${host}:${port}/sse`;

        return JSON.stringify({
            mcpServers: {
                "rclone-bisync": {
                    url: url
                }
            }
        }, null, 2);
    },

    async copyMcpEndpoint() {
        try {
            await navigator.clipboard.writeText(this.mcpEndpointUrl);
            this.mcpModalCopiedEndpoint = true;
            setTimeout(() => { this.mcpModalCopiedEndpoint = false; }, 2000);
        } catch (_) {
            if (typeof this.showToast === "function") {
                this.showToast("Failed to copy to clipboard", "error");
            }
        }
    },

    async copyMcpConfig() {
        try {
            await navigator.clipboard.writeText(this.mcpClientSnippet);
            this.mcpModalCopiedConfig = true;
            setTimeout(() => { this.mcpModalCopiedConfig = false; }, 2000);
        } catch (_) {
            if (typeof this.showToast === "function") {
                this.showToast("Failed to copy to clipboard", "error");
            }
        }
    }
};
