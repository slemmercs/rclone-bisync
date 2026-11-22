package models

import (
	"encoding/json"
	"testing"
)

func TestHasherWrapRequest_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name       string
		payload    string
		wantHashes string
		wantErr    bool
	}{
		{
			name:       "string hashes comma separated",
			payload:    `{"name":"remote_hasher","remote":"remote:","hashes":"md5,sha1"}`,
			wantHashes: "md5,sha1",
			wantErr:    false,
		},
		{
			name:       "string hashes mixed case and dashes",
			payload:    `{"name":"remote_hasher","remote":"remote:","hashes":"MD5,SHA-1,QuickXorHash,CRC-32"}`,
			wantHashes: "md5,sha1,quickxor,crc32",
			wantErr:    false,
		},
		{
			name:       "array hashes with uppercase display labels",
			payload:    `{"name":"remote_hasher","remote":"remote:","hashes":["MD5","SHA-1"]}`,
			wantHashes: "md5,sha1",
			wantErr:    false,
		},
		{
			name:       "array hashes with multiple providers",
			payload:    `{"name":"remote_hasher","remote":"remote:","hashes":["DropboxHash","CRC-32","Whirlpool"]}`,
			wantHashes: "dropbox,crc32,whirlpool",
			wantErr:    false,
		},
		{
			name:       "string auto hashes",
			payload:    `{"name":"remote_hasher","remote":"remote:","hashes":"auto"}`,
			wantHashes: "auto",
			wantErr:    false,
		},
		{
			name:       "empty hashes field",
			payload:    `{"name":"remote_hasher","remote":"remote:"}`,
			wantHashes: "",
			wantErr:    false,
		},
		{
			name:       "empty array hashes",
			payload:    `{"name":"remote_hasher","remote":"remote:","hashes":[]}`,
			wantHashes: "",
			wantErr:    false,
		},
		{
			name:       "malformed json",
			payload:    `{"name":`,
			wantHashes: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req HasherWrapRequest
			err := json.Unmarshal([]byte(tt.payload), &req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && req.Hashes != tt.wantHashes {
				t.Errorf("req.Hashes = %q, want %q", req.Hashes, tt.wantHashes)
			}
		})
	}
}

func TestHasherCommandRequestJSON(t *testing.T) {
	req := HasherCommandRequest{
		Remote:  "drive_hasher",
		Command: "dump",
		Args:    []string{"md5"},
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal HasherCommandRequest: %v", err)
	}

	var parsed HasherCommandRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal HasherCommandRequest: %v", err)
	}
	if parsed.Remote != req.Remote || parsed.Command != req.Command {
		t.Errorf("mismatch: got %+v, want %+v", parsed, req)
	}
}
