package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
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
// A missing/rejected lookup never establishes either failure or zero charge.
func VerifyCanvasImageSupplierBill(c *gin.Context) {
	if !IsCanvasImageRequest(c) {
		return
	}
	charge := lookupCanvasImageSupplierBill(c.Request.Context(), c.GetString("channel_name"), c.GetString(common.UpstreamRequestIdKey), c.GetString("canvas_supplier_model"), common.GetContextKeyString(c, constant.ContextKeyChannelKey))
	if charge != nil {
		c.Set("canvas_supplier_charge", charge)
	}
}

// Rechecking an existing receipt never starts generation. This lets Canvas's
// existing settlement poll capture a late supplier bill after gateway settlement.
func RefreshCanvasImageSupplierBills(c *gin.Context, logs []*model.Log) {
	if os.Getenv("CANVAS_IMAGE_ROUTING_ENABLED") != "true" || c.Query("request_id") == "" {
		return
	}
	cache := make(map[int]*canvasSupplierBills)
	for _, row := range logs {
		if row.Group != CanvasImageGroup || !slices.Contains(c.QueryArray("request_id"), row.RequestId) || row.UpstreamRequestId == "" {
			continue
		}
		var other map[string]any
		if common.UnmarshalJsonStr(row.Other, &other) != nil || other["supplier_charge"] != nil {
			continue
		}
		name, _ := other["image_supplier"].(string)
		upstreamModel, _ := other["upstream_model_name"].(string)
		channel, err := model.GetChannelById(row.ChannelId, true)
		if err != nil || channel.Group != CanvasImageGroup || channel.Name != name || channel.ChannelInfo.IsMultiKey {
			continue
		}
		bills, cached := cache[row.ChannelId]
		if !cached {
			bills = fetchCanvasImageSupplierBills(c.Request.Context(), name, channel.Key)
			cache[row.ChannelId] = bills
		}
		if charge := bills.find(row.UpstreamRequestId, upstreamModel); charge != nil {
			other["supplier_charge"] = charge
			if body, err := common.Marshal(other); err == nil {
				row.Other = string(body)
			}
		}
	}
}

func lookupCanvasImageSupplierBill(ctx context.Context, name, id, upstreamModel, key string) map[string]any {
	if id == "" || key == "" {
		return nil
	}
	return fetchCanvasImageSupplierBills(ctx, name, key).find(id, upstreamModel)
}

type canvasSupplierBills struct {
	Success bool `json:"success"`
	Data    []struct {
		RequestID string `json:"request_id"`
		Type      int    `json:"type"`
		Quota     int64  `json:"quota"`
		Model     string `json:"model_name"`
	} `json:"data"`
	source, currency, hash string
}

func fetchCanvasImageSupplierBills(ctx context.Context, name, key string) *canvasSupplierBills {
	source, currency := "", ""
	switch name {
	case "canvas-qidong-flare", "canvas-qidong-sunburst":
		source, currency = "https://api.qidong.autos/api/log/token", "CNY"
	case "canvas-packy-flare", "canvas-packy-sunburst":
		source, currency = "https://www.packyapi.com/api/log/token", "USD"
	default:
		return nil
	}
	if key == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+key)
	client := &http.Client{Timeout: 4 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil
	}
	var result canvasSupplierBills
	if common.Unmarshal(body, &result) != nil || !result.Success {
		return nil
	}
	result.source, result.currency, result.hash = source, currency, fmt.Sprintf("%x", sha256.Sum256(body))
	return &result
}

func (result *canvasSupplierBills) find(id, upstreamModel string) map[string]any {
	if result == nil || id == "" || (upstreamModel != "gpt-image-2.5-flare" && upstreamModel != "gpt-image-2.5-sunburst") {
		return nil
	}
	matched := 0
	var quota int64
	for _, row := range result.Data {
		if row.RequestID == id && row.Type == 6 {
			return nil // A refund requires reconciliation, never a guessed net.
		}
		if row.RequestID == id && row.Type == 2 && row.Model == upstreamModel && row.Quota >= 0 {
			matched++
			quota = row.Quota
		}
	}
	if matched != 1 {
		return nil
	}
	return map[string]any{"status": "verified", "amount": fmt.Sprintf("%.6f", float64(quota)/500000), "currency": result.currency, "request_id": id, "model": upstreamModel, "quota": quota, "quota_per_unit": 500000, "source": result.source, "snapshot_sha256": result.hash, "cash_paid_status": "unknown"}
}

const CanvasImageGroup = "canvas-image"
const canvasImageCooldownReason = "canvas_image_cooldown"

func CanvasImageAttemptTimeout(c *gin.Context) time.Duration {
	variable, fallback := "CANVAS_IMAGE_PRIMARY_TIMEOUT_SECONDS", 180
	if strings.HasPrefix(c.GetString("channel_name"), "canvas-packy-") {
		variable, fallback = "CANVAS_IMAGE_BACKUP_TIMEOUT_SECONDS", 240
	}
	seconds, err := strconv.Atoi(os.Getenv(variable))
	if err != nil || seconds < 1 || seconds > fallback {
		seconds = fallback
	}
	return time.Duration(seconds) * time.Second
}

func canvasImageCooldown(name string) time.Duration {
	variable := "CANVAS_IMAGE_FLARE_COOLDOWN_SECONDS"
	if strings.HasSuffix(name, "-sunburst") {
		variable = "CANVAS_IMAGE_SUNBURST_COOLDOWN_SECONDS"
	}
	seconds, err := strconv.Atoi(os.Getenv(variable))
	if err != nil || seconds < 1 || seconds > 86400 {
		seconds = 300
	}
	return time.Duration(seconds) * time.Second
}

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
	if c.Writer.Written() || c.Request.Context().Err() != nil {
		stop.Reason = "image_response_started_or_caller_cancelled"
		return stop
	}
	if charge, exists := c.Get("canvas_supplier_charge"); exists && charge != nil {
		stop.Reason = "image_original_supplier_already_charged"
		return stop
	}
	if canvasImageContentRefusal(err) || err.StatusCode == 400 || err.StatusCode == 403 || err.StatusCode == 422 {
		stop.Reason = "image_content_or_parameter_rejection"
		return stop
	}
	if strings.HasPrefix(c.GetString("channel_name"), "canvas-packy-") {
		stop.Reason = "image_backup_is_last_supplier"
		return stop
	}
	// Known pre-send/authentication failures are separate from the explicitly
	// opted-in unknown-result branch below.
	preSend := err.GetErrorCode() == types.ErrorCodeChannelNoAvailableKey || err.GetErrorCode() == types.ErrorCodeChannelInvalidKey || err.GetErrorCode() == types.ErrorCodeChannelParamOverrideInvalid || err.GetErrorCode() == types.ErrorCodeChannelHeaderOverrideInvalid || err.GetErrorCode() == types.ErrorCodeChannelModelMappedError
	if preSend || err.StatusCode == 401 && err.GetErrorType() == types.ErrorTypeOpenAIError {
		return PolicyDecision{Action: "retry", Reason: "image_request_not_accepted", Source: "canvas_image"}
	}
	// Explicitly deployed opt-in only. Unknown means the first attempt may
	// still complete and charge; it remains unknown in the failed attempt log.
	ambiguous := err.StatusCode >= 500 || err.StatusCode == 408 || err.StatusCode == 429 || err.GetErrorCode() == types.ErrorCodeDoRequestFailed || err.GetErrorCode() == types.ErrorCodeChannelResponseTimeExceeded
	if ambiguous && os.Getenv("CANVAS_IMAGE_AMBIGUOUS_FAILOVER_ENABLED") == "true" {
		return PolicyDecision{Action: "retry", Reason: "image_unknown_result_duplicate_charge_possible", Source: "canvas_image"}
	}
	return stop
}

// Eligibility after cooldown is only a probe. Health is confirmed by a real
// successful image request, independently for each model's own channel.
func ConfirmCanvasImageRecovery(c *gin.Context) {
	if !IsCanvasImageRequest(c) {
		return
	}
	canvasImageHealthLock.Lock()
	defer canvasImageHealthLock.Unlock()
	channel, err := model.GetChannelById(c.GetInt("channel_id"), true)
	if err == nil && channel.Status == common.ChannelStatusEnabled && channel.GetOtherInfo()["status_reason"] == "canvas_image_probe_pending" {
		if err := changeCanvasImageStatus(channel, common.ChannelStatusEnabled, ""); err != nil {
			common.SysLog("canvas image recovery confirmation failed")
		}
	}
}

func canvasImageContentRefusal(err *types.NewAPIError) bool {
	text := strings.ToLower(string(err.GetErrorCode()) + " " + err.Error())
	for _, value := range []string{"content_policy", "safety", "moderation", "prompt_blocked", "sensitive", "内容安全", "内容审核"} {
		if strings.Contains(text, value) {
			return true
		}
	}
	return false
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
		if info["status_reason"] != canvasImageCooldownReason || stamp <= 0 || time.Since(time.Unix(int64(stamp), 0)) < canvasImageCooldown(channel.Name) {
			continue
		}
		if err := changeCanvasImageStatus(channel, common.ChannelStatusEnabled, "canvas_image_probe_pending"); err != nil {
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
	if canvasImageContentRefusal(err) {
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
