package models

import (
	"encoding/json"
	"testing"
)

func TestCoreModelsSerialization(t *testing.T) {
	// 1. BandwidthLimitRequest
	req := BandwidthLimitRequest{Rate: "10M"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal BandwidthLimitRequest: %v", err)
	}
	var parsedReq BandwidthLimitRequest
	if err := json.Unmarshal(data, &parsedReq); err != nil {
		t.Fatalf("failed to unmarshal BandwidthLimitRequest: %v", err)
	}
	if parsedReq.Rate != "10M" {
		t.Errorf("expected rate 10M, got %s", parsedReq.Rate)
	}

	// 2. RcloneBwlimitRaw
	rawBw := `{"bytesPerSecond": 10485760, "rate": "10M"}`
	var bwRaw RcloneBwlimitRaw
	if err := json.Unmarshal([]byte(rawBw), &bwRaw); err != nil {
		t.Fatalf("failed to unmarshal RcloneBwlimitRaw: %v", err)
	}
	if bwRaw.BytesPerSecond != 10485760 || bwRaw.Rate != "10M" {
		t.Errorf("unexpected bwRaw values: %+v", bwRaw)
	}

	// 3. RcloneStatsRaw
	rawStats := `{
		"bytes": 52428800,
		"speed": 1048576.5,
		"transfers": 2,
		"checks": 42,
		"errors": 0,
		"fatalErrors": false,
		"transferring": [
			{
				"name": "backup.tar.gz",
				"size": 104857600,
				"bytes": 52428800,
				"percentage": 50,
				"speed": 1048576.5,
				"speedAvg": 980000.0,
				"eta": 50
			}
		]
	}`
	var statsRaw RcloneStatsRaw
	if err := json.Unmarshal([]byte(rawStats), &statsRaw); err != nil {
		t.Fatalf("failed to unmarshal RcloneStatsRaw: %v", err)
	}
	if statsRaw.Bytes != 52428800 || len(statsRaw.Transferring) != 1 {
		t.Fatalf("unexpected statsRaw: %+v", statsRaw)
	}
	if statsRaw.Transferring[0].ETA == nil || *statsRaw.Transferring[0].ETA != 50 {
		t.Errorf("expected ETA 50, got %v", statsRaw.Transferring[0].ETA)
	}
}
