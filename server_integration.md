# Server Integration Guide

## 📊 JSON Payload Structure

**Updated Structure** - The middleware now provides optimized JSON with common fields moved to top-level for better analytics performance:

### Key Changes
- **`duration_ns`**: Added for precise aggregation (nanoseconds)
- **Top-level fields**: `service`, `environment`, `version`, `region` moved from `tags`
- **Smaller tags**: Only custom/additional tags remain in `tags` object
- **Better indexing**: Common fields at top-level enable faster queries

### Single Request Metric
```json
{
  "request_id": "req_1671234567890123456_987654321",
  "method": "POST",
  "path": "/api/users/:id",
  "status_code": 201,
  "duration": "45.123ms",
  "duration_ns": 45123000,
  "request_size": 1024,
  "response_size": 512,
  "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
  "client_ip": "192.168.1.100",
  "headers": {
    "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
    "Content-Type": "application/json",
    "Accept": "application/json"
  },
  "timestamp": "2024-06-13T12:30:45.123456Z",
  "service": "user-api",
  "environment": "production",
  "version": "1.2.3",
  "region": "us-west-2",
  "tags": {
    "team": "backend",
    "component": "auth"
  }
}
```

### Batch Metrics (Recommended)
```json
{
  "metrics": [
    {
      "request_id": "req_1671234567890123456_987654321",
      "method": "GET",
      "path": "/api/users",
      "status_code": 200,
      "duration": "23.456ms",
      "duration_ns": 23456000,
      "request_size": 0,
      "response_size": 2048,
      "user_agent": "curl/7.68.0",
      "client_ip": "10.0.0.1",
      "timestamp": "2024-06-13T12:30:45.100Z",
      "service": "user-api",
      "environment": "production"
    },
    {
      "request_id": "req_1671234567890123457_987654322",
      "method": "POST", 
      "path": "/api/users",
      "status_code": 201,
      "duration": "89.234ms",
      "duration_ns": 89234000,
      "request_size": 1024,
      "response_size": 256,
      "user_agent": "MyApp/1.0",
      "client_ip": "10.0.0.2",
      "timestamp": "2024-06-13T12:30:46.200Z",
      "service": "user-api",
      "environment": "production",
      "tags": {
        "team": "frontend",
        "feature": "user-registration"
      }
    }
  ],
  "metadata": {
    "batch_id": "batch_1671234567890",
    "source": "pprofio-middleware",
    "version": "1.0.0",
    "sent_at": "2024-06-13T12:30:47.000Z",
    "batch_size": 2
  }
}
```

## 🛣️ Recommended HTTP Routes

### For pprofio.io Platform
```
POST /api/v1/http-metrics              # Single metric ingestion
POST /api/v1/http-metrics/batch        # Batch metrics ingestion
```

### For Custom Analytics Platforms
```
POST /api/v1/metrics/ingest
POST /api/v1/observability/http-metrics
POST /api/v1/telemetry/requests
POST /internal/metrics/http
```

## 🔧 Integration Implementation

### Option 1: Real-time Streaming (Low Latency)
```go
collector.SetMetricsHandler(func(metrics *RequestMetrics) {
    // Send each metric immediately
    sendToServer("/api/v1/http-metrics", metrics)
})
```

### Option 2: Batched (Recommended for Production)
```go
type BatchingHandler struct {
    buffer   []*RequestMetrics
    mu       sync.Mutex
    ticker   *time.Ticker
    endpoint string
}

func (bh *BatchingHandler) AddMetric(metrics *RequestMetrics) {
    bh.mu.Lock()
    defer bh.mu.Unlock()
    
    bh.buffer = append(bh.buffer, metrics)
    
    // Send when buffer is full (e.g., 100 metrics)
    if len(bh.buffer) >= 100 {
        bh.flush()
    }
}

func (bh *BatchingHandler) flush() {
    if len(bh.buffer) == 0 {
        return
    }
    
    batch := map[string]interface{}{
        "metrics": bh.buffer,
        "metadata": map[string]interface{}{
            "batch_id":   generateBatchID(),
            "source":     "pprofio-middleware",
            "sent_at":    time.Now().Format(time.RFC3339),
            "batch_size": len(bh.buffer),
        },
    }
    
    sendBatchToServer(bh.endpoint, batch)
    bh.buffer = bh.buffer[:0] // Clear buffer
}

// Use with middleware
batcher := &BatchingHandler{endpoint: "/api/v1/http-metrics/batch"}
collector.SetMetricsHandler(batcher.AddMetric)

// Flush every 30 seconds
batcher.ticker = time.NewTicker(30 * time.Second)
go func() {
    for range batcher.ticker.C {
        batcher.mu.Lock()
        batcher.flush()
        batcher.mu.Unlock()
    }
}()
```

## 📡 HTTP Client Configuration

### Headers
```go
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer " + apiKey)
req.Header.Set("User-Agent", "pprofio-agent/1.0.0")
req.Header.Set("X-Source", "http-middleware")
```

### Recommended Settings
- **Timeout**: 10 seconds
- **Retries**: 3 attempts with exponential backoff
- **Compression**: gzip encoding
- **Keepalive**: Enabled for batching scenarios

## 🔒 Authentication

### API Key (Recommended)
```
X-API-Key: your-api-key-here
X-Service-Token: service-specific-token
```

## 📊 Server Response Format

### Success Response
```json
{
  "status": "success",
  "received": 150,
  "processed": 150,
  "batch_id": "batch_1671234567890",
  "processing_time_ms": 12
}
```

### Error Response
```json
{
  "status": "error",
  "error_code": "INVALID_PAYLOAD",
  "message": "Invalid timestamp format in metric 15",
  "details": {
    "field": "timestamp",
    "metric_index": 15,
    "expected_format": "RFC3339"
  }
}
```

## 🔄 Field Mapping & Migration

### Automatic Field Promotion
The middleware automatically moves common fields from `tags` to top-level:

```go
// When you set these tags:
adapter.WithTags(map[string]string{
    "service":     "user-api",     // ➜ moves to metrics.Service
    "environment": "production",   // ➜ moves to metrics.Environment  
    "version":     "1.2.3",       // ➜ moves to metrics.Version
    "region":      "us-west-2",   // ➜ moves to metrics.Region
    "team":        "backend",     // ➜ stays in metrics.Tags["team"]
    "component":   "auth",        // ➜ stays in metrics.Tags["component"]
})
```

### Benefits of New Structure
1. **Faster Queries**: Top-level fields indexed separately from tags
2. **Smaller Payloads**: Reduced nesting and redundancy
3. **Better Aggregation**: `duration_ns` enables precise time-based analytics
4. **Consistent Schema**: Standard fields always at predictable locations
5. **Custom Tags**: Preserved for domain-specific metadata

### Database Schema Recommendations
```sql
-- Optimized table structure for new JSON format
CREATE TABLE http_metrics (
    request_id VARCHAR(64) PRIMARY KEY,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,
    status_code INTEGER NOT NULL,
    duration_ns BIGINT NOT NULL,      -- NEW: for fast aggregations
    request_size BIGINT,
    response_size BIGINT,
    service VARCHAR(100),             -- NEW: top-level field
    environment VARCHAR(50),          -- NEW: top-level field
    version VARCHAR(50),             -- NEW: top-level field  
    region VARCHAR(50),              -- NEW: top-level field
    user_agent TEXT,
    client_ip VARCHAR(45),
    headers JSONB,
    tags JSONB,                      -- Remaining custom tags
    timestamp TIMESTAMPTZ NOT NULL,
    
    -- Optimized indexes
    INDEX idx_service_env (service, environment),
    INDEX idx_duration_ns (duration_ns),
    INDEX idx_timestamp (timestamp)
);
```

## 💾 Data Sizes

- **Single metric**: ~500-800 bytes
- **Batch of 100**: ~60-80KB
- **Compression**: ~70% reduction with gzip
- **Recommended batch size**: 50-200 metrics

## ⚡ Performance Recommendations

1. **Use batching** for production (30-60 second intervals)
2. **Enable compression** for payloads >1KB
3. **Implement circuit breaker** to handle server outages
4. **Queue metrics locally** if server is unreachable
5. **Monitor upload success rate** and alert on failures

## 🚀 Quick Start Integration

```go
// 1. Create HTTP client with retries
httpClient := &http.Client{
    Timeout: 10 * time.Second,
}

// 2. Create metrics handler
handler := func(metrics *RequestMetrics) {
    jsonData, _ := json.Marshal(metrics)
    
    req, _ := http.NewRequest("POST", 
        "https://api.pprofio.io/api/v1/http-metrics", 
        bytes.NewBuffer(jsonData))
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer " + apiKey)
    
    resp, err := httpClient.Do(req)
    if err != nil {
        log.Printf("Failed to send metrics: %v", err)
        return
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        log.Printf("Server returned status: %d", resp.StatusCode)
    }
}

// 3. Integrate with middleware
collector.SetMetricsHandler(handler)
``` 