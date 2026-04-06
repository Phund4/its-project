package coordinatorclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"data-ingestion/internal/config"
)

type Client struct {
	base string
	cli  *http.Client
}

func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		base: strings.TrimRight(baseURL, "/"),
		cli:  &http.Client{Timeout: timeout},
	}
}

type assignmentItem struct {
	SegmentID      string `json:"segment_id"`
	CameraID       string `json:"camera_id"`
	RTSPURL        string `json:"rtsp_url"`
	MunicipalityID string `json:"municipality_id"`
}

type assignmentsResp struct {
	Items []assignmentItem `json:"items"`
}

func (c *Client) fetchAssignments(ctx context.Context, zoneID, clusterID, instanceID, sourceKind string) ([]assignmentItem, error) {
	u, err := url.Parse(c.base + "/v1/assignments")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("zone_id", zoneID)
	q.Set("cluster_id", clusterID)
	q.Set("instance_id", instanceID)
	q.Set("source_kind", sourceKind)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coordinator assignments status: %s", resp.Status)
	}
	var ar assignmentsResp
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, err
	}
	return ar.Items, nil
}

func (c *Client) FetchCameraAssignments(ctx context.Context, zoneID, clusterID, instanceID string) ([]config.Camera, error) {
	items, err := c.fetchAssignments(ctx, zoneID, clusterID, instanceID, "camera")
	if err != nil {
		return nil, err
	}
	out := make([]config.Camera, 0, len(items))
	for _, it := range items {
		if strings.TrimSpace(it.SegmentID) == "" || strings.TrimSpace(it.CameraID) == "" || strings.TrimSpace(it.RTSPURL) == "" {
			continue
		}
		out = append(out, config.Camera{
			SegmentID: it.SegmentID,
			CameraID:  it.CameraID,
			RTSPURL:   it.RTSPURL,
		})
	}
	return out, nil
}

func (c *Client) FetchTelemetryMunicipalities(ctx context.Context, zoneID, clusterID, instanceID string) ([]string, error) {
	items, err := c.fetchAssignments(ctx, zoneID, clusterID, instanceID, "telemetry")
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, it := range items {
		m := strings.TrimSpace(it.MunicipalityID)
		if m == "" {
			continue
		}
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, m)
	}
	return out, nil
}

func (c *Client) SendHeartbeat(ctx context.Context, zoneID, clusterID, instanceID string, assignments int) error {
	body := map[string]any{
		"zone_id":      zoneID,
		"cluster_id":   clusterID,
		"instance_id":  instanceID,
		"assignments":  assignments,
		"observed_at":  time.Now().UTC(),
		"load":         0.0,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/v1/workers/heartbeat", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("coordinator heartbeat status: %s", resp.Status)
	}
	return nil
}
