/**
 * Rclone Bisync Dashboard — Centralized API Client
 * Provides consistent HTTP request handling, automatic JSON serialization,
 * CSRF protection headers ("X-Dashboard-Request: 1"), and normalized error propagation.
 */

window.API = {
    async request(endpoint, options = {}) {
        const headers = {
            "Content-Type": "application/json",
            "X-Dashboard-Request": "1",
            ...(options.headers || {})
        };

        const config = {
            ...options,
            headers
        };

        const res = await fetch(endpoint, config);
        const contentType = res.headers.get("content-type") || "";
        let data = null;

        if (contentType.includes("application/json")) {
            try {
                data = await res.json();
            } catch (_) {
                data = null;
            }
        } else {
            try {
                data = await res.text();
            } catch (_) {
                data = null;
            }
        }

        if (!res.ok) {
            const message = (data && typeof data === "object" && (data.error || data.message))
                || (typeof data === "string" && data.trim())
                || `HTTP ${res.status}: ${res.statusText}`;
            const err = new Error(message);
            err.status = res.status;
            err.data = data;
            throw err;
        }

        return data;
    },

    get(endpoint, options = {}) {
        return this.request(endpoint, { ...options, method: "GET" });
    },

    post(endpoint, body, options = {}) {
        return this.request(endpoint, {
            ...options,
            method: "POST",
            body: body !== undefined ? JSON.stringify(body) : undefined
        });
    },

    put(endpoint, body, options = {}) {
        return this.request(endpoint, {
            ...options,
            method: "PUT",
            body: body !== undefined ? JSON.stringify(body) : undefined
        });
    },

    delete(endpoint, options = {}) {
        return this.request(endpoint, { ...options, method: "DELETE" });
    }
};
