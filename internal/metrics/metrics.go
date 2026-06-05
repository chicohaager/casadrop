package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// UploadsTotal counts total uploads
	UploadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zima_uploads_total",
			Help: "Total number of file uploads",
		},
		[]string{"status"}, // "success" or "failed"
	)

	// DownloadsTotal counts total downloads
	DownloadsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "zima_downloads_total",
			Help: "Total number of file downloads",
		},
	)

	// SharesCreated counts total shares created
	SharesCreated = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "zima_shares_created_total",
			Help: "Total number of shares created",
		},
	)

	// SharesDeleted counts total shares deleted
	SharesDeleted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "zima_shares_deleted_total",
			Help: "Total number of shares deleted",
		},
	)

	// SharesExpired counts total shares expired
	SharesExpired = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "zima_shares_expired_total",
			Help: "Total number of shares that expired",
		},
	)

	// ActiveShares shows current number of active shares
	ActiveShares = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "zima_active_shares",
			Help: "Number of currently active (non-expired) shares",
		},
	)

	// StorageUsedBytes shows total storage used
	StorageUsedBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "zima_storage_used_bytes",
			Help: "Total storage used in bytes",
		},
	)

	// UploadBytes counts total bytes uploaded
	UploadBytes = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "zima_upload_bytes_total",
			Help: "Total bytes uploaded",
		},
	)

	// DownloadBytes counts total bytes downloaded
	DownloadBytes = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "zima_download_bytes_total",
			Help: "Total bytes downloaded",
		},
	)

	// HTTPRequestsTotal counts HTTP requests
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zima_http_requests_total",
			Help: "Total HTTP requests by method, path and status",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration measures request duration
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "zima_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// AuthFailures counts authentication failures
	AuthFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zima_auth_failures_total",
			Help: "Total authentication failures",
		},
		[]string{"type"}, // "login", "share_password"
	)

	// RateLimitHits counts rate limit hits
	RateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zima_rate_limit_hits_total",
			Help: "Total rate limit hits",
		},
		[]string{"endpoint"},
	)
)

// RecordUpload records an upload metric
func RecordUpload(success bool, bytes int64) {
	status := "success"
	if !success {
		status = "failed"
	}
	UploadsTotal.WithLabelValues(status).Inc()
	if success {
		UploadBytes.Add(float64(bytes))
		SharesCreated.Inc()
	}
}

// RecordDownload records a download metric
func RecordDownload(bytes int64) {
	DownloadsTotal.Inc()
	DownloadBytes.Add(float64(bytes))
}

// RecordShareDeleted records a share deletion
func RecordShareDeleted() {
	SharesDeleted.Inc()
}

// RecordShareExpired records a share expiration
func RecordShareExpired() {
	SharesExpired.Inc()
}

// UpdateActiveShares updates the active shares gauge
func UpdateActiveShares(count int) {
	ActiveShares.Set(float64(count))
}

// UpdateStorageUsed updates the storage used gauge
func UpdateStorageUsed(bytes int64) {
	StorageUsedBytes.Set(float64(bytes))
}

// RecordAuthFailure records an authentication failure
func RecordAuthFailure(authType string) {
	AuthFailures.WithLabelValues(authType).Inc()
}

// RecordRateLimitHit records a rate limit hit
func RecordRateLimitHit(endpoint string) {
	RateLimitHits.WithLabelValues(endpoint).Inc()
}
