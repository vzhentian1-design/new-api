package service

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func canvasImageContext(t *testing.T) *gin.Context {
	t.Helper()
	t.Setenv("CANVAS_IMAGE_ROUTING_ENABLED", "true")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/images/generations", nil)
	common.SetContextKey(c, constant.ContextKeyUsingGroup, CanvasImageGroup)
	RequestPolicy(c).Attempts = 1
	return c
}

func TestCanvasImageRetryIsBoundedAndStopsAmbiguousResults(t *testing.T) {
	c := canvasImageContext(t)
	for _, status := range []int{400, 403, 408, 409, 429, 500, 502, 503, 504, 524} {
		err := types.NewOpenAIError(errors.New("synthetic"), types.ErrorCodeBadResponseStatusCode, status)
		if got := DecideRelayRetry(c, err, 1); got.Action != "stop" {
			t.Fatalf("status %d must stop: %+v", status, got)
		}
	}
	for _, code := range []types.ErrorCode{types.ErrorCodeDoRequestFailed, types.ErrorCodeBadResponseBody, types.ErrorCodeReadResponseBodyFailed, types.ErrorCodeChannelResponseTimeExceeded} {
		if got := DecideRelayRetry(c, types.NewOpenAIError(errors.New("synthetic"), code, 500), 1); got.Action != "stop" {
			t.Fatalf("ambiguous %s must stop", code)
		}
	}
	err := types.NewOpenAIError(errors.New("invalid token"), "invalid_api_key", 401)
	if got := DecideRelayRetry(c, err, 1); got.Action != "retry" {
		t.Fatalf("explicit auth rejection: %+v", got)
	}
	RequestPolicy(c).Attempts = 2
	if got := DecideRelayRetry(c, err, 1); got.Action != "stop" {
		t.Fatal("third attempt was permitted")
	}
	if CanvasImageRetryLimit(c, 0) != 1 {
		t.Fatal("image policy must have its own bounded retry limit")
	}
	common.SetContextKey(c, constant.ContextKeyUsingGroup, "default")
	if CanvasImageRetryLimit(c, 0) != 0 || IsCanvasImageRequest(c) {
		t.Fatal("default group changed")
	}
}

func TestCanvasImageNativeCooldownPersistsAndRecoversOnDemand(t *testing.T) {
	c := canvasImageContext(t)
	oldDB, oldMemory := model.DB, common.MemoryCacheEnabled
	t.Cleanup(func() { model.DB, common.MemoryCacheEnabled = oldDB, oldMemory })
	common.MemoryCacheEnabled = true
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	model.DB = db
	if err := db.AutoMigrate(&model.Channel{}, &model.Ability{}); err != nil {
		t.Fatal(err)
	}
	channel := &model.Channel{Id: 9001, Name: "test-primary", Group: CanvasImageGroup, Status: common.ChannelStatusEnabled, Key: "synthetic-only"}
	if err := db.Create(channel).Error; err != nil {
		t.Fatal(err)
	}
	primary, backup := int64(100), int64(10)
	db.Model(channel).Updates(map[string]any{"priority": primary, "models": "canvas-image"})
	if err := db.Create(&model.Channel{Id: 9002, Name: "test-backup", Group: CanvasImageGroup, Status: common.ChannelStatusEnabled, Key: "synthetic-only"}).Error; err != nil {
		t.Fatal(err)
	}
	db.Model(&model.Channel{}).Where("id = ?", 9002).Updates(map[string]any{"priority": backup, "models": "canvas-image"})
	if err := db.Create(&[]model.Ability{{Group: CanvasImageGroup, Model: "canvas-image", ChannelId: 9001, Enabled: true, Priority: &primary, Weight: 100}, {Group: CanvasImageGroup, Model: "canvas-image", ChannelId: 9002, Enabled: true, Priority: &backup, Weight: 100}}).Error; err != nil {
		t.Fatal(err)
	}
	model.InitChannelCache()
	selected, err := model.GetRandomSatisfiedChannel(CanvasImageGroup, "canvas-image", 0, nil)
	if err != nil || selected.Id != 9001 {
		t.Fatal("primary priority not respected")
	}
	if !coolCanvasImageChannel(c, channel.Id, types.NewOpenAIError(errors.New("synthetic"), "503", 503)) {
		t.Fatal("not handled")
	}
	var saved model.Channel
	db.First(&saved, channel.Id)
	if saved.Status != common.ChannelStatusAutoDisabled {
		t.Fatal("failure not persisted")
	}
	selected, err = model.GetRandomSatisfiedChannel(CanvasImageGroup, "canvas-image", 0, nil)
	if err != nil || selected.Id != 9002 {
		t.Fatal("next new task did not select backup")
	}
	if err := RecoverCanvasImageChannels(c); err != nil {
		t.Fatal(err)
	}
	db.First(&saved, channel.Id)
	if saved.Status != common.ChannelStatusAutoDisabled {
		t.Fatal("cooldown ended early")
	}
	info := saved.GetOtherInfo()
	info["status_time"] = time.Now().Add(-6 * time.Minute).Unix()
	saved.SetOtherInfo(info)
	db.Model(&saved).Update("other_info", saved.OtherInfo)
	if err := RecoverCanvasImageChannels(c); err != nil {
		t.Fatal(err)
	}
	db.First(&saved, channel.Id)
	if saved.Status != common.ChannelStatusEnabled {
		t.Fatal("primary not made eligible")
	}
	selected, err = model.GetRandomSatisfiedChannel(CanvasImageGroup, "canvas-image", 0, nil)
	if err != nil || selected.Id != 9001 {
		t.Fatal("recovered primary priority not restored")
	}
	db.Model(&saved).Updates(map[string]any{"status": common.ChannelStatusManuallyDisabled, "other_info": `{}`})
	if err := RecoverCanvasImageChannels(c); err != nil {
		t.Fatal(err)
	}
	db.First(&saved, channel.Id)
	if saved.Status != common.ChannelStatusManuallyDisabled {
		t.Fatal("manual hold was overridden")
	}
}
