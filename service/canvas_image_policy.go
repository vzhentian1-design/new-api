package service

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

// A supplier account debit is separate from this gateway's quota and cash paid.
// This read-only lookup is enabled only for the verified Qidong image account.
func VerifyCanvasImageSupplierBill(c *gin.Context) {
	if !IsCanvasImageRequest(c) || c.GetString("channel_name") != "canvas-qidong-flare" {
		return
	}
	id := c.GetString(common.UpstreamRequestIdKey)
	key := common.GetContextKeyString(c, constant.ContextKeyChannelKey)
	if id == "" || key == "" {
		return
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, "https://api.qidong.autos/api/log/token", nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+key)
	client := &http.Client{Timeout: 4 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return
	}
	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			RequestID string `json:"request_id"`
			Type      int    `json:"type"`
			Quota     int64  `json:"quota"`
			Model     string `json:"model_name"`
		} `json:"data"`
	}
	if common.Unmarshal(body, &result) != nil || !result.Success {
		return
	}
	matched := 0
	var quota int64
	for _, row := range result.Data {
		if row.RequestID == id && row.Type == 2 && row.Model == "gpt-image-2.5-flare" && row.Quota >= 0 {
			matched++
			quota = row.Quota
		}
	}
	if matched != 1 {
		return
	}
	c.Set("canvas_supplier_charge", map[string]any{"status": "verified", "amount": fmt.Sprintf("%.6f", float64(quota)/500000), "currency": "CNY", "request_id": id, "quota": quota, "quota_per_unit": 500000, "source": "https://api.qidong.autos/api/log/token", "snapshot_sha256": fmt.Sprintf("%x", sha256.Sum256(body)), "cash_paid_status": "unknown"})
}

const CanvasImageGroup = "canvas-image"
const canvasImageCooldownReason = "canvas_image_cooldown"
const canvasImageCooldown = 5 * time.Minute

var canvasImageHealthLock sync.Mutex

// Opt-in for this project only. Existing text users keep their current policy.
func IsCanvasImageRequest(c *gin.Context) bool {
	return os.Getenv("CANVAS_IMAGE_ROUTING_ENABLED") == "true" && c != nil && c.Request != nil &&
		common.GetContextKeyString(c, constant.ContextKeyUsingGroup) == CanvasImageGroup &&
		(c.Request.URL.Path == "/v1/images/generations" || c.Request.URL.Path == "/v1/images/edits")
}

func CanvasImageRetryLimit(c *gin.Context, current int) int {
	if IsCanvasImageRequest(c) {
		return 1 // At most the primary and one independently approved backup.
	}
	return current
}

func decideCanvasImageRetry(c *gin.Context, err *types.NewAPIError, remaining int) PolicyDecision {
	stop := PolicyDecision{Action: "stop", Reason: "image_result_unknown_or_not_retryable", Source: "canvas_image"}
	if remaining <= 0 || RequestPolicy(c).Attempts >= 2 {
		stop.Reason = "image_attempt_budget_exhausted"
		return stop
	}
	if types.IsSkipRetryError(err) {
		return stop
	}
	// Only a local pre-send channel failure or explicit authentication rejection.
	// 403, 429, 5xx, malformed responses, and transport failures never replay.
	preSend := err.GetErrorCode() == types.ErrorCodeChannelNoAvailableKey || err.GetErrorCode() == types.ErrorCodeChannelInvalidKey || err.GetErrorCode() == types.ErrorCodeChannelParamOverrideInvalid || err.GetErrorCode() == types.ErrorCodeChannelHeaderOverrideInvalid || err.GetErrorCode() == types.ErrorCodeChannelModelMappedError
	if preSend || err.StatusCode == 401 && err.GetErrorType() == types.ErrorTypeOpenAIError {
		return PolicyDecision{Action: "retry", Reason: "image_request_not_accepted", Source: "canvas_image"}
	}
	return stop
}

// Reuse native persistent channel status, abilities and priority selection.
// Recovery occurs on demand; no scheduled billable channel tests are added.
func RecoverCanvasImageChannels(c *gin.Context) error {
	if !IsCanvasImageRequest(c) {
		return nil
	}
	canvasImageHealthLock.Lock()
	defer canvasImageHealthLock.Unlock()
	var channels []*model.Channel
	if err := model.DB.Where(map[string]any{"group": CanvasImageGroup, "status": common.ChannelStatusAutoDisabled}).Find(&channels).Error; err != nil {
		return err
	}
	for _, channel := range channels {
		info := channel.GetOtherInfo()
		stamp, _ := info["status_time"].(float64)
		if info["status_reason"] != canvasImageCooldownReason || stamp <= 0 || time.Since(time.Unix(int64(stamp), 0)) < canvasImageCooldown {
			continue
		}
		if err := changeCanvasImageStatus(channel, common.ChannelStatusEnabled, ""); err != nil {
			return err
		}
	}
	return nil
}

func coolCanvasImageChannel(c *gin.Context, channelID int, err *types.NewAPIError) bool {
	if !IsCanvasImageRequest(c) {
		return false
	}
	if err == nil || (err.StatusCode >= 400 && err.StatusCode < 500 && err.StatusCode != 401 && err.StatusCode != 403 && err.StatusCode != 429) {
		return true
	}
	// Content refusals must not disable an otherwise healthy supplier.
	code := strings.ToLower(string(err.GetErrorCode()))
	if strings.Contains(code, "policy") || strings.Contains(code, "prompt_blocked") || strings.Contains(code, "sensitive") {
		return true
	}
	canvasImageHealthLock.Lock()
	defer canvasImageHealthLock.Unlock()
	channel, getErr := model.GetChannelById(channelID, true)
	if getErr == nil && channel.Group == CanvasImageGroup && channel.Status == common.ChannelStatusEnabled {
		if updateErr := changeCanvasImageStatus(channel, common.ChannelStatusAutoDisabled, canvasImageCooldownReason); updateErr != nil {
			common.SysLog("canvas image cooldown persistence failed")
		}
	}
	return true // Never invoke global asynchronous auto-ban for this group.
}

func changeCanvasImageStatus(channel *model.Channel, status int, reason string) error {
	previous := channel.OtherInfo
	info := channel.GetOtherInfo()
	info["status_reason"], info["status_time"] = reason, common.GetTimestamp()
	channel.SetOtherInfo(info)
	result := model.DB.Model(&model.Channel{}).Where("id = ? AND status = ? AND other_info = ?", channel.Id, channel.Status, previous).
		Updates(map[string]any{"status": status, "other_info": channel.OtherInfo})
	if result.Error != nil || result.RowsAffected == 0 {
		return result.Error
	}
	if err := model.UpdateAbilityStatus(channel.Id, status == common.ChannelStatusEnabled); err != nil {
		return err
	}
	model.InitChannelCache()
	return nil
}
