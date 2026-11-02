/**
 * Guided Rclone Bisync Flags Builder Module
 * Manages standard flag catalog, row parsing, serialization, and real-time CLI preview.
 * Strictly enforces <400 LOC modularity constraint.
 */

const RCLONE_FLAGS_CATALOG = [
    { flag: "--fast-list", label: "--fast-list", desc: "Use recursive list (faster scanning, uses more RAM)", isBool: true },
    { flag: "--drive-shared-with-me", label: "--drive-shared-with-me", desc: "Sync Google Drive items shared with you", isBool: true },
    { flag: "--checksum", label: "--checksum", desc: "Skip based on checksum comparison instead of modtime", isBool: true },
    { flag: "--ignore-checksum", label: "--ignore-checksum", desc: "Skip checksum comparison even if available", isBool: true },
    { flag: "--size-only", label: "--size-only", desc: "Skip based on size only (not modtime or checksum)", isBool: true },
    { flag: "--resilient", label: "--resilient", desc: "Allow bisync to continue on non-fatal errors", isBool: true },
    { flag: "--recover", label: "--recover", desc: "Automatically recover from state inconsistencies", isBool: true },
    { flag: "--checkers", label: "--checkers", desc: "Number of checkers to run in parallel", isBool: false, placeholder: "8" },
    { flag: "--transfers", label: "--transfers", desc: "Number of file transfers to run in parallel", isBool: false, placeholder: "4" },
    { flag: "--bwlimit", label: "--bwlimit", desc: "Bandwidth limit (e.g. 10M, 500k, 10M:off)", isBool: false, placeholder: "10M" },
    { flag: "--tpslimit", label: "--tpslimit", desc: "Limit HTTP transactions per second", isBool: false, placeholder: "10" },
    { flag: "--tpslimit-burst", label: "--tpslimit-burst", desc: "Max HTTP transactions burst", isBool: false, placeholder: "20" },
    { flag: "--buffer-size", label: "--buffer-size", desc: "In-memory buffer size per transfer (e.g. 16M, 64M)", isBool: false, placeholder: "16M" },
    { flag: "--retries", label: "--retries", desc: "Number of retries on network failure", isBool: false, placeholder: "3" },
    { flag: "--retries-sleep", label: "--retries-sleep", desc: "Sleep interval between retries", isBool: false, placeholder: "500ms" },
    { flag: "--low-level-retries", label: "--low-level-retries", desc: "Low-level retry count", isBool: false, placeholder: "10" }
];

window.FlagsBuilderModule = {
    flagsCatalog: RCLONE_FLAGS_CATALOG,
    customFlagRows: [],
    cliCopied: false,

    initFlagsBuilder(rawCustomFlags) {
        this.customFlagRows = this.parseCustomFlags(rawCustomFlags || "");
    },

    addFlagRow(defaultFlag = "") {
        const catalogItem = this.flagsCatalog.find(c => c.flag === defaultFlag);
        const isCustom = defaultFlag === "custom" || (!catalogItem && defaultFlag !== "");
        const flagName = isCustom ? (defaultFlag === "custom" ? "--" : defaultFlag) : (defaultFlag || this.flagsCatalog[0].flag);
        const finalBool = catalogItem ? catalogItem.isBool : false;

        this.customFlagRows.push({
            id: "flag_" + Math.random().toString(16).substring(2, 8),
            flag: flagName,
            value: "",
            isBool: finalBool,
            isCustom: isCustom
        });
        this.syncFlagsToCustomFlags();
    },

    removeFlagRow(index) {
        if (index >= 0 && index < this.customFlagRows.length) {
            this.customFlagRows.splice(index, 1);
            this.syncFlagsToCustomFlags();
        }
    },

    onFlagSelectChange(index, selectedValue) {
        const row = this.customFlagRows[index];
        if (!row) return;

        if (selectedValue === "custom") {
            row.isCustom = true;
            row.flag = "--";
            row.isBool = false;
            row.value = "";
        } else {
            const item = this.flagsCatalog.find(c => c.flag === selectedValue);
            row.isCustom = false;
            row.flag = selectedValue;
            row.isBool = item ? item.isBool : false;
            row.value = "";
        }
        this.syncFlagsToCustomFlags();
    },

    onFlagValueInput(index, val) {
        const row = this.customFlagRows[index];
        if (!row) return;
        row.value = val;
        this.syncFlagsToCustomFlags();
    },

    onFlagCustomNameInput(index, name) {
        const row = this.customFlagRows[index];
        if (!row) return;
        row.flag = name;
        this.syncFlagsToCustomFlags();
    },

    syncFlagsToCustomFlags() {
        const serialized = this.serializeFlagsRows(this.customFlagRows);
        if (this.profileModal && this.profileModal.editing && this.profileModal.editing.flags) {
            this.profileModal.editing.flags.custom_flags = serialized || null;
        }
    },

    parseCustomFlags(raw) {
        if (!raw || typeof raw !== "string" || !raw.trim()) return [];
        const tokens = [];
        const re = /[^\s"']+|"([^"]*)"|'([^']*)'/g;
        let match;
        while ((match = re.exec(raw)) !== null) {
            tokens.push(match[1] !== undefined ? match[1] : (match[2] !== undefined ? match[2] : match[0]));
        }

        const rows = [];
        let i = 0;
        while (i < tokens.length) {
            const tok = tokens[i];
            if (!tok.startsWith("-")) {
                i++;
                continue;
            }
            let flag = tok;
            let val = "";
            let hasEqual = false;
            if (tok.includes("=")) {
                const eqIdx = tok.indexOf("=");
                flag = tok.substring(0, eqIdx);
                val = tok.substring(eqIdx + 1);
                hasEqual = true;
            }
            const known = this.flagsCatalog.find(c => c.flag === flag);
            const isBool = known ? known.isBool : (hasEqual ? (val === "true" || val === "false") : false);
            const isCustom = !known;

            if (!hasEqual && !isBool && i + 1 < tokens.length && !tokens[i + 1].startsWith("-")) {
                val = tokens[i + 1];
                i++;
            }
            rows.push({
                id: "flag_" + Math.random().toString(16).substring(2, 8),
                flag: flag,
                value: val,
                isBool: isBool,
                isCustom: isCustom
            });
            i++;
        }
        return rows;
    },

    serializeFlagsRows(rows) {
        if (!Array.isArray(rows) || rows.length === 0) return "";
        const parts = [];
        for (const r of rows) {
            const flag = (r.flag || "").trim();
            if (!flag || !flag.startsWith("-")) continue;
            if (r.isBool) {
                if (r.value === "false" || r.value === false) {
                    parts.push(`${flag}=false`);
                } else {
                    parts.push(flag);
                }
            } else {
                const val = (r.value || "").trim();
                if (val === "") {
                    parts.push(flag);
                } else if (val.includes(" ") || val.includes('"')) {
                    parts.push(`${flag} "${val.replace(/"/g, '\\"')}"`);
                } else {
                    parts.push(`${flag} ${val}`);
                }
            }
        }
        return parts.join(" ");
    },

    generateCLIPreview() {
        const p = this.profileModal?.editing;
        if (!p) return "";
        const globals = this.profilesData?.globalDefaults || {};
        const flags = p.flags || {};
        const parts = ["rclone", "bisync"];

        const remote = p.remote ? (p.remote_path ? `${p.remote.replace(/:$/, "")}:${p.remote_path}` : `${p.remote.replace(/:$/, "")}:`) : "<remote:path>";
        const local = p.local_path || "<local-path>";
        parts.push(remote, local);

        if (flags.dry_run === true) parts.push("--dry-run");
        if (flags.resync === true) parts.push("--resync");
        if (flags.force === true) parts.push("--force");

        const conflictResolve = flags.conflict_resolve || globals.conflict_resolve || "newer";
        parts.push("--conflict-resolve", conflictResolve);

        const maxDelete = flags.max_delete !== null && flags.max_delete !== undefined ? flags.max_delete : (globals.max_delete !== undefined ? globals.max_delete : 0);
        parts.push("--max-delete", String(maxDelete));

        const canary = flags.canary_validation !== null && flags.canary_validation !== undefined ? flags.canary_validation : (globals.canary_validation !== undefined ? globals.canary_validation : true);
        if (canary) parts.push("--check-access", "--check-filename", "RCLONE_TEST");

        const compareMode = flags.compare_mode || globals.compare_mode || "size,modtime";
        parts.push("--compare", compareMode);

        if (Array.isArray(p.exclude_patterns)) {
            for (const pat of p.exclude_patterns) {
                if (pat && pat.trim()) parts.push("--exclude", `"${pat.trim()}"`);
            }
        }

        const removeEmpty = flags.remove_empty_dirs !== null && flags.remove_empty_dirs !== undefined ? flags.remove_empty_dirs : (globals.remove_empty_dirs !== undefined ? globals.remove_empty_dirs : true);
        if (removeEmpty) parts.push("--remove-empty-dirs");

        const createEmpty = flags.create_empty_dirs !== null && flags.create_empty_dirs !== undefined ? flags.create_empty_dirs : (globals.create_empty_dirs !== undefined ? globals.create_empty_dirs : true);
        if (createEmpty) parts.push("--create-empty-src-dirs");

        if (flags.filters_file_path && flags.filters_file_path.trim()) {
            parts.push("--filters-file", `"${flags.filters_file_path.trim()}"`);
        }

        const po = flags.provider_options || {};
        const addBool = (key, flagName) => {
            const v = String(po[key] || "").trim().toLowerCase();
            if (v === "true" || v === "1") parts.push(flagName);
            else if (v === "false" || v === "0") parts.push(`${flagName}=false`);
        };
        addBool("drive_skip_shortcuts", "--drive-skip-shortcuts");
        addBool("drive_use_trash", "--drive-use-trash");
        addBool("drive_skip_gdocs", "--drive-skip-gdocs");
        addBool("drive_shared_with_me", "--drive-shared-with-me");
        addBool("drive_acknowledge_abuse", "--drive-acknowledge-abuse");
        if (po.drive_export_formats && po.drive_export_formats.trim()) parts.push("--drive-export-formats", po.drive_export_formats.trim());
        if (po.drive_import_formats && po.drive_import_formats.trim()) parts.push("--drive-import-formats", po.drive_import_formats.trim());
        addBool("onedrive_no_versions", "--onedrive-no-versions");
        if (po.onedrive_chunk_size && po.onedrive_chunk_size.trim()) parts.push("--onedrive-chunk-size", po.onedrive_chunk_size.trim());
        if (po.s3_chunk_size && po.s3_chunk_size.trim()) parts.push("--s3-chunk-size", po.s3_chunk_size.trim());
        if (po.s3_storage_class && po.s3_storage_class.trim()) parts.push("--s3-storage-class", po.s3_storage_class.trim());
        if (po.dropbox_chunk_size && po.dropbox_chunk_size.trim()) parts.push("--dropbox-chunk-size", po.dropbox_chunk_size.trim());

        if (flags.custom_flags && flags.custom_flags.trim()) {
            parts.push(flags.custom_flags.trim());
        }

        return parts.join(" ");
    },

    copyCLIPreview() {
        const text = this.generateCLIPreview();
        if (!text) return;
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(() => {
                this.cliCopied = true;
                setTimeout(() => { this.cliCopied = false; }, 2000);
                if (typeof this.showToast === "function") {
                    this.showToast("CLI command copied to clipboard", "success");
                }
            }).catch(() => {});
        }
    }
};
