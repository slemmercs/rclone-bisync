package models

// BandwidthLimitRequest specifies a target rate limit for Rclone (e.g. "10M", "500k", "off").
type BandwidthLimitRequest struct {
	Rate string `json:"rate"`
}

// BandwidthLimitResponse represents the active bandwidth throttle setting.
type BandwidthLimitResponse struct {
	BytesPerSecond int64  `json:"bytes_per_second"`
	RateFormatted  string `json:"rate_formatted"`
	Status         string `json:"status,omitempty"`
}

// RcloneBwlimitRaw represents the raw JSON returned by Rclone's core/bwlimit RC endpoint.
type RcloneBwlimitRaw struct {
	BytesPerSecond int64  `json:"bytesPerSecond"`
	Rate           string `json:"rate"`
}

// CoreStatsResponse holds aggregate transfer metrics and active file transfers from core/stats.
type CoreStatsResponse struct {
	Bytes        int64            `json:"bytes"`
	Speed        float64          `json:"speed"`
	Transfers    int              `json:"transfers"`
	Checks       int              `json:"checks"`
	Errors       int              `json:"errors"`
	FatalErrors  bool             `json:"fatal_errors"`
	Transferring []ActiveTransfer `json:"transferring,omitempty"`
}

// ActiveTransfer represents an in-flight file transfer reported by Rclone.
type ActiveTransfer struct {
	Name       string  `json:"name"`
	Size       int64   `json:"size"`
	Bytes      int64   `json:"bytes"`
	Percentage int     `json:"percentage"`
	Speed      float64 `json:"speed"`
	SpeedAvg   float64 `json:"speed_avg,omitempty"`
	ETA        int64   `json:"eta"`
}

// RcloneStatsRaw represents the raw JSON returned by Rclone's core/stats RC endpoint.
type RcloneStatsRaw struct {
	Bytes        int64                     `json:"bytes"`
	Speed        float64                   `json:"speed"`
	Transfers    int                       `json:"transfers"`
	Checks       int                       `json:"checks"`
	Errors       int                       `json:"errors"`
	FatalErrors  bool                      `json:"fatalErrors"`
	Transferring []RcloneActiveTransferRaw `json:"transferring"`
}

// RcloneActiveTransferRaw represents raw transfer items in Rclone stats.
type RcloneActiveTransferRaw struct {
	Name       string  `json:"name"`
	Size       int64   `json:"size"`
	Bytes      int64   `json:"bytes"`
	Percentage int     `json:"percentage"`
	Speed      float64 `json:"speed"`
	SpeedAvg   float64 `json:"speedAvg"`
	ETA        *int64  `json:"eta"`
}
