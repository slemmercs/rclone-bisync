/**
 * Rclone Bisync Dashboard — Declarative Target Provider Options Engine
 * Single-source metadata schema for Tier 1 cloud target provider options.
 */

const PROVIDER_OPTIONS_SCHEMA = [
    {
        provider: "drive",
        key: "drive_skip_shortcuts",
        flag: "--drive-skip-shortcuts",
        type: "bool",
        label: "Skip Shortcuts",
        default: "true",
        desc: "Skip shortcut files to prevent synchronization loops."
    },
    {
        provider: "drive",
        key: "drive_use_trash",
        flag: "--drive-use-trash",
        type: "bool",
        label: "Use Trash",
        default: "true",
        desc: "Move deleted files to Google Drive trash instead of permanently deleting."
    },
    {
        provider: "drive",
        key: "drive_shared_with_me",
        flag: "--drive-shared-with-me",
        type: "bool",
        label: "Shared With Me",
        default: "false",
        desc: "Instructs Google Drive to display items in 'Shared with me' directory."
    },
    {
        provider: "drive",
        key: "drive_skip_gdocs",
        flag: "--drive-skip-gdocs",
        type: "bool",
        label: "Skip GDocs",
        default: "false",
        desc: "Skip Google Docs/Sheets/Slides during sync to avoid export failures."
    },
    {
        provider: "drive",
        key: "drive_acknowledge_abuse",
        flag: "--drive-acknowledge-abuse",
        type: "bool",
        label: "Acknowledge Abuse",
        default: "false",
        desc: "Allow download of files flagged by Google as abusive."
    },
    {
        provider: "drive",
        key: "drive_export_formats",
        flag: "--drive-export-formats",
        type: "text",
        label: "Export Formats",
        placeholder: "e.g. docx,xlsx,pptx",
        desc: "Comma-separated list of formats to export Google Docs to."
    },
    {
        provider: "drive",
        key: "drive_import_formats",
        flag: "--drive-import-formats",
        type: "text",
        label: "Import Formats",
        placeholder: "e.g. docx,xlsx",
        desc: "Comma-separated list of formats to convert on upload to Google Drive."
    },
    {
        provider: "onedrive",
        key: "onedrive_no_versions",
        flag: "--onedrive-no-versions",
        type: "bool",
        label: "No Versions (Prevent Duplicates)",
        default: "true",
        desc: "Removes old versions after upload on OneDrive Personal."
    },
    {
        provider: "onedrive",
        key: "onedrive_delta",
        flag: "--onedrive-delta",
        type: "bool",
        label: "Delta Enumeration (--onedrive-delta)",
        default: "false",
        desc: "Use delta change enumeration for rapid change detection via Graph API."
    },
    {
        provider: "onedrive",
        key: "onedrive_chunk_size",
        flag: "--onedrive-chunk-size",
        type: "text",
        label: "Upload Chunk Size",
        placeholder: "e.g. 10M, 50M (multiple of 320k)",
        desc: "Chunk size for OneDrive uploads. Must be multiple of 320k."
    },
    {
        provider: "s3",
        key: "s3_storage_class",
        flag: "--s3-storage-class",
        type: "select",
        label: "Storage Class",
        default: "STANDARD",
        options: [
            "STANDARD",
            "STANDARD_IA",
            "ONEZONE_IA",
            "INTELLIGENT_TIERING",
            "GLACIER",
            "DEEP_ARCHIVE",
            "GLACIER_IR"
        ],
        desc: "Storage class to use when storing objects in S3."
    },
    {
        provider: "s3",
        key: "s3_chunk_size",
        flag: "--s3-chunk-size",
        type: "text",
        label: "Part / Chunk Size",
        default: "16M",
        placeholder: "16M",
        desc: "Chunk size to use for uploading multipart files to S3."
    },
    {
        provider: "dropbox",
        key: "dropbox_chunk_size",
        flag: "--dropbox-chunk-size",
        type: "text",
        label: "Upload Chunk Size",
        default: "48M",
        placeholder: "48M",
        desc: "Chunk size for uploading large files to Dropbox."
    }
];

window.PROVIDER_OPTIONS_SCHEMA = PROVIDER_OPTIONS_SCHEMA;

window.ProviderOptionsModule = {
    providerOptionsSchema: PROVIDER_OPTIONS_SCHEMA,

    isTier1Provider() {
        const type = typeof this.getEditingRemoteType === "function" ? this.getEditingRemoteType() : "";
        return PROVIDER_OPTIONS_SCHEMA.some(o => o.provider === type);
    },

    getProviderOptionsList() {
        const prov = typeof this.getEditingRemoteType === "function" ? this.getEditingRemoteType() : "";
        return PROVIDER_OPTIONS_SCHEMA.filter(o => o.provider === prov);
    },

    getProviderBoolOptionsList() {
        const prov = typeof this.getEditingRemoteType === "function" ? this.getEditingRemoteType() : "";
        return PROVIDER_OPTIONS_SCHEMA.filter(o => o.provider === prov && o.type === "bool");
    },

    getProviderNonBoolOptionsList() {
        const prov = typeof this.getEditingRemoteType === "function" ? this.getEditingRemoteType() : "";
        return PROVIDER_OPTIONS_SCHEMA.filter(o => o.provider === prov && o.type !== "bool");
    },

    isProviderOptionChecked(opt) {
        if (!opt) return false;
        const def = opt.default === "true";
        return this.getProviderBoolOption(opt.key, def);
    },

    onProviderOptionToggle(opt, isChecked) {
        if (!opt) return;
        this.setProviderBoolOption(opt.key, Boolean(isChecked));
    },

    getProviderOptionValue(opt) {
        if (!opt) return "";
        return this.getProviderOption(opt.key, opt.default || "");
    },

    setProviderOptionValue(opt, val) {
        if (!opt) return;
        this.setProviderOption(opt.key, val);
    }
};
