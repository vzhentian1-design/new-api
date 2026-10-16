package service

import (
	"errors"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestCanvasImageCatalogCannotCrossModelsOrSuppliers(t *testing.T) {
	for _, tc := range []struct {
		requested, upstream, channel string
		allowed                      bool
	}{
		{"gpt-image-2", "gpt-image-2", "canvas-packy-image2", true},
		{"gpt-image-2", "gpt-image-2.5-flare", "canvas-packy-flare", false},
		{"gpt-image-2", "gpt-image-2", "canvas-qidong-flare", false},
		{"gpt-image-2.5-flare", "gpt-image-2.5-flare", "canvas-qidong-flare", true},
		{"gpt-image-2.5-sunburst", "gpt-image-2.5-sunburst", "canvas-packy-sunburst", true},
		{"gpt-image-2.5-sunburst", "gpt-image-2.5-flare", "canvas-packy-flare", false},
		{"canvas-image", "gpt-image-2.5-flare", "canvas-packy-flare", true},
		{"canvas-image-sunburst", "gpt-image-2.5-sunburst", "canvas-qidong-sunburst", true},
		{"unknown", "gpt-image-2.5-flare", "canvas-packy-flare", false},
	} {
		assert.Equal(t, tc.allowed, CanvasImageModelAllowed(tc.requested, tc.upstream, tc.channel), "%+v", tc)
	}
}

func TestCanvasImageOptInUnknownFailoverAndRefusals(t *testing.T) {
	t.Setenv("CANVAS_IMAGE_AMBIGUOUS_FAILOVER_ENABLED", "true")
	for _, tc := range []struct {
		name    string
		status  int
		code    types.ErrorCode
		message string
		want    string
	}{
		{"unavailable", 503, "503", "unavailable", "retry"},
		{"timeout", 504, "504", "timeout", "retry"},
		{"disconnect", 500, types.ErrorCodeDoRequestFailed, "EOF", "retry"},
		{"bad_parameter", 400, "invalid_request", "bad size", "stop"},
		{"forbidden", 403, "403", "forbidden", "stop"},
		{"safety_5xx", 503, "content_policy_violation", "safety refusal", "stop"},
		{"invalid_parameter", 422, "422", "invalid", "stop"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := canvasImageContext(t)
			c.Set("channel_name", "canvas-qidong-flare")
			err := types.NewOpenAIError(errors.New(tc.message), tc.code, tc.status)
			assert.Equal(t, tc.want, DecideRelayRetry(c, err, 1).Action)
			RequestPolicy(c).Attempts = 2
			assert.Equal(t, "stop", DecideRelayRetry(c, err, 1).Action)
		})
	}
	c := canvasImageContext(t)
	err := types.NewOpenAIError(errors.New("synthetic outage"), "503", 503)
	c.Set("channel_name", "canvas-packy-flare")
	assert.Equal(t, "stop", DecideRelayRetry(c, err, 1).Action)
	c.Set("channel_name", "canvas-qidong-flare")
	c.Set("canvas_supplier_charge", map[string]any{"status": "verified"})
	assert.Equal(t, "image_original_supplier_already_charged", DecideRelayRetry(c, err, 1).Reason)
	c.Set("canvas_supplier_charge", nil)
	c.Writer.WriteHeader(200)
	_, writeErr := c.Writer.Write([]byte("partial image"))
	require.NoError(t, writeErr)
	assert.Equal(t, "stop", DecideRelayRetry(c, err, 1).Action)
}

func TestCanvasImageSeparateModelCooldownSettings(t *testing.T) {
	t.Setenv("CANVAS_IMAGE_FLARE_COOLDOWN_SECONDS", "301")
	t.Setenv("CANVAS_IMAGE_SUNBURST_COOLDOWN_SECONDS", "601")
	assert.Equal(t, 301*time.Second, canvasImageCooldown("canvas-qidong-flare"))
	assert.Equal(t, 601*time.Second, canvasImageCooldown("canvas-qidong-sunburst"))
	t.Setenv("CANVAS_IMAGE_SUNBURST_COOLDOWN_SECONDS", "-1")
	assert.Equal(t, 300*time.Second, canvasImageCooldown("canvas-qidong-sunburst"))
	c := canvasImageContext(t)
	c.Set("channel_name", "canvas-qidong-flare")
	assert.Equal(t, 180*time.Second, CanvasImageAttemptTimeout(c))
	c.Set("channel_name", "canvas-packy-sunburst")
	assert.Equal(t, 240*time.Second, CanvasImageAttemptTimeout(c))
}

func TestCanvasImageSupplierBillRequiresUniqueUnrefundedReceipt(t *testing.T) {
	for _, tc := range []struct{ body, amount string }{
		{`{"data":[{"request_id":"one","type":2,"quota":15000,"model_name":"gpt-image-2.5-flare"}]}`, "0.030000"},
		{`{"data":[{"request_id":"other","type":2,"quota":15000,"model_name":"gpt-image-2.5-flare"}]}`, ""},
		{`{"data":[{"request_id":"one","type":2,"quota":15000,"model_name":"gpt-image-2.5-sunburst"}]}`, ""},
		{`{"data":[{"request_id":"one","type":2,"quota":15000,"model_name":"gpt-image-2.5-flare"},{"request_id":"one","type":6,"quota":15000,"model_name":"gpt-image-2.5-flare"}]}`, ""},
		{`{"data":[{"request_id":"one","type":2,"quota":15000,"model_name":"gpt-image-2.5-flare"},{"request_id":"one","type":2,"quota":15000,"model_name":"gpt-image-2.5-flare"}]}`, ""},
	} {
		var bills canvasSupplierBills
		require.NoError(t, common.UnmarshalJsonStr(tc.body, &bills))
		bills.currency = "CNY"
		got := bills.find("one", "gpt-image-2.5-flare")
		if tc.amount == "" {
			assert.Nil(t, got)
		} else {
			require.NotNil(t, got)
			assert.Equal(t, tc.amount, got["amount"])
		}
	}
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
	oldDB, oldLogDB, oldMemory, oldMaster, oldType := model.DB, model.LOG_DB, common.MemoryCacheEnabled, common.IsMasterNode, common.MainDatabaseType()
	t.Cleanup(func() {
		model.DB, model.LOG_DB, common.MemoryCacheEnabled, common.IsMasterNode = oldDB, oldLogDB, oldMemory, oldMaster
		common.SetMainDatabaseType(oldType)
	})
	common.MemoryCacheEnabled = true
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if dsn := os.Getenv("CANVAS_POLICY_DATABASE_DSN"); dsn != "" {
		// This opt-in DSN must name a disposable, isolated fixture database.
		t.Setenv("SQL_DSN", dsn)
		common.IsMasterNode = false
		require.NoError(t, model.InitDB())
		db = model.DB
	}
	model.DB, model.LOG_DB = db, db
	if err := db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.Log{}); err != nil {
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
	require.NoError(t, db.Create(&[]model.Log{{RequestId: "own-a", TokenId: 41, Type: 2}, {RequestId: "foreign-b", TokenId: 42, Type: 2}, {RequestId: "own-c", TokenId: 41, Type: 5}}).Error)
	logs, err := model.GetLogByTokenId(41, "own-a", "foreign-b")
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, "own-a", logs[0].RequestId)
	logs, err = model.GetLogByTokenId(41, "own-a", "own-c")
	require.NoError(t, err)
	assert.Len(t, logs, 2)
}
