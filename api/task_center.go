package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"ppeelink/models"
	"ppeelink/rulecenter"
)

var taskCancels sync.Map // task id -> context.CancelFunc

type nodeEgressTaskRequest struct { NodeID int `json:"nodeId"` }
type subscriptionEgressTaskRequest struct { SubscriptionID int `json:"subscriptionId"` }
type airportSyncTaskRequest struct { AirportID int `json:"airportId"` }
type ruleSyncTaskRequest struct { Source string `json:"source"` }
type subscriptionBuildTaskRequest struct { SubscriptionID int `json:"subscriptionId"`; Client string `json:"client"` }

func createTaskRun(base context.Context, kind, name string, request any, retryOf *uint) (*models.TaskRun, context.Context, error) {
	raw, _ := json.Marshal(request)
	now := time.Now()
	task := &models.TaskRun{Type:kind, Name:name, Status:"running", Progress:1, Message:"任务已启动", RequestJSON:string(raw), RetryOf:retryOf, StartedAt:&now}
	if err := models.DB.Create(task).Error; err != nil { return nil, nil, err }
	ctx, cancel := context.WithCancel(base)
	taskCancels.Store(task.ID, cancel)
	return task, ctx, nil
}

func updateTaskProgress(id uint, progress int, message string) {
	if progress < 0 { progress = 0 }; if progress > 100 { progress = 100 }
	_ = models.DB.Model(&models.TaskRun{}).Where("id = ?", id).Updates(map[string]any{"progress":progress,"message":message}).Error
}

func finishTaskRun(id uint, err error, result any) {
	taskCancels.Delete(id)
	now := time.Now(); updates := map[string]any{"finished_at":&now,"progress":100}
	if err != nil { updates["status"]="failed"; updates["error"]=err.Error(); updates["message"]="任务失败" } else { updates["status"]="success"; updates["error"]=""; updates["message"]="任务完成" }
	if result != nil { if raw, marshalErr := json.Marshal(result); marshalErr == nil { updates["result_json"] = string(raw) } }
	_ = models.DB.Model(&models.TaskRun{}).Where("id = ?", id).Updates(updates).Error
}

func markTaskCancelled(id uint) {
	taskCancels.Delete(id)
	now := time.Now()
	_ = models.DB.Model(&models.TaskRun{}).Where("id = ?", id).Updates(map[string]any{"status":"cancelled","message":"任务已取消","finished_at":&now}).Error
}

func executeStoredTask(task *models.TaskRun) {
	ctx, cancel := context.WithCancel(context.Background()); taskCancels.Store(task.ID, cancel); defer cancel()
	now := time.Now(); _ = models.DB.Model(task).Updates(map[string]any{"status":"running","progress":5,"started_at":&now,"message":"正在执行"}).Error
	var result any; var err error
	switch task.Type {
	case "node-egress":
		var req nodeEgressTaskRequest; err = json.Unmarshal([]byte(task.RequestJSON), &req); if err == nil { updateTaskProgress(task.ID,20,"正在检测目标出口"); result, err = runNodeEgressTask(ctx, req.NodeID) }
	case "subscription-egress":
		var req subscriptionEgressTaskRequest; err = json.Unmarshal([]byte(task.RequestJSON), &req); if err == nil { updateTaskProgress(task.ID,20,"正在按模板验证分流"); result, err = runSubscriptionEgressPlanTask(ctx, req.SubscriptionID) }
	case "airport-sync":
		var req airportSyncTaskRequest; err = json.Unmarshal([]byte(task.RequestJSON), &req); if err == nil { updateTaskProgress(task.ID,20,"正在拉取并测活机场节点"); err = SyncAirportNodeTask(req.AirportID) }
	case "rule-sync":
		var req ruleSyncTaskRequest; err = json.Unmarshal([]byte(task.RequestJSON), &req)
		if err == nil { syncCtx, syncCancel := context.WithTimeout(ctx,120*time.Second); defer syncCancel(); updateTaskProgress(task.ID,20,"正在同步规则源"); if req.Source == "" { err = rulecenter.SyncAll(syncCtx) } else { err = rulecenter.SyncSource(syncCtx,req.Source) } }
	case "subscription-build":
		var req subscriptionBuildTaskRequest; err = json.Unmarshal([]byte(task.RequestJSON), &req)
		if err == nil { updateTaskProgress(task.ID,20,"正在生成并验证订阅产物"); result, err = buildAndSnapshotSubscription(ctx, req.SubscriptionID, req.Client) }
	default:
		err = fmt.Errorf("任务类型不支持重试: %s", task.Type)
	}
	if ctx.Err() == context.Canceled { markTaskCancelled(task.ID); return }
	finishTaskRun(task.ID, err, result)
}

func buildSubscriptionOutput(subscriptionID int, client string) ([]byte, error) {
	client = strings.ToLower(strings.TrimSpace(client))
	if client != "clash" && client != "surge" && client != "loon" && client != "v2ray" { return nil, fmt.Errorf("不支持的客户端: %s", client) }
	var sub models.Subcription
	if err := models.DB.First(&sub, subscriptionID).Error; err != nil { return nil, err }
	recorder := httptest.NewRecorder(); c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/", nil); c.Set("subname", sub.Name)
	switch client { case "clash": GetClash(c); case "surge": GetSurge(c); case "loon": GetLoon(c); case "v2ray": GetV2ray(c) }
	body := recorder.Body.Bytes(); if len(body) == 0 { return nil, fmt.Errorf("订阅生成结果为空") }
	if recorder.Code >= 400 { return nil, fmt.Errorf("订阅生成失败 HTTP %d", recorder.Code) }
	return append([]byte(nil), body...), nil
}

func StartSubscriptionBuildTask(c *gin.Context) {
	var req subscriptionBuildTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil { req.SubscriptionID, _ = strconv.Atoi(c.PostForm("subscriptionId")); req.Client = c.PostForm("client") }
	if req.SubscriptionID <= 0 { c.JSON(400,gin.H{"code":"40000","msg":"订阅 id 格式错误"}); return }
	req.Client = strings.ToLower(strings.TrimSpace(req.Client)); if req.Client == "" { req.Client = "clash" }
	if req.Client != "clash" && req.Client != "surge" && req.Client != "loon" && req.Client != "v2ray" { c.JSON(400,gin.H{"code":"40000","msg":"客户端仅支持 clash/surge/loon/v2ray"}); return }
	var sub models.Subcription; if err := models.DB.First(&sub,req.SubscriptionID).Error; err != nil { c.JSON(404,gin.H{"code":"40400","msg":"订阅不存在"}); return }
	task, taskCtx, err := createTaskRun(context.Background(), "subscription-build", "订阅构建 · "+sub.Name+" · "+req.Client, req, nil)
	if err != nil { c.JSON(500,gin.H{"code":"50000","msg":"创建任务失败: "+err.Error()}); return }
	go func(){ updateTaskProgress(task.ID,20,"正在生成并验证订阅产物"); meta, runErr := buildAndSnapshotSubscription(taskCtx,req.SubscriptionID,req.Client); if taskCtx.Err()==context.Canceled { markTaskCancelled(task.ID); return }; finishTaskRun(task.ID,runErr,meta) }()
	c.JSON(200,gin.H{"code":"00000","data":gin.H{"taskId":task.ID},"msg":"订阅构建任务已启动"})
}

func TaskList(c *gin.Context) {
	limit := 80; if v, err := strconv.Atoi(c.DefaultQuery("limit","80")); err == nil && v >= 1 && v <= 300 { limit = v }
	var tasks []models.TaskRun
	if err := models.DB.Order("id desc").Limit(limit).Find(&tasks).Error; err != nil { c.JSON(500,gin.H{"code":"50000","msg":"读取任务失败"}); return }
	c.JSON(200,gin.H{"code":"00000","data":tasks,"msg":"任务中心"})
}

func TaskCancel(c *gin.Context) {
	id64, err := strconv.ParseUint(c.Query("id"),10,64); if err != nil || id64 == 0 { c.JSON(400,gin.H{"code":"40000","msg":"任务 id 格式错误"}); return }
	var task models.TaskRun; if err := models.DB.First(&task,uint(id64)).Error; err != nil { c.JSON(404,gin.H{"code":"40400","msg":"任务不存在"}); return }
	if task.Status != "running" && task.Status != "queued" { c.JSON(409,gin.H{"code":"40900","msg":"任务当前不可取消"}); return }
	if value, ok := taskCancels.Load(task.ID); ok { value.(context.CancelFunc)() }
	markTaskCancelled(task.ID)
	c.JSON(200,gin.H{"code":"00000","msg":"已请求取消"})
}

func TaskRetry(c *gin.Context) {
	id64, err := strconv.ParseUint(c.Query("id"),10,64); if err != nil || id64 == 0 { c.JSON(400,gin.H{"code":"40000","msg":"任务 id 格式错误"}); return }
	var old models.TaskRun; if err := models.DB.First(&old,uint(id64)).Error; err != nil { c.JSON(404,gin.H{"code":"40400","msg":"任务不存在"}); return }
	now := time.Now(); task := &models.TaskRun{Type:old.Type,Name:old.Name,Status:"queued",Progress:0,Message:"等待重试",RequestJSON:old.RequestJSON,RetryOf:&old.ID,CreatedAt:now}
	if err := models.DB.Create(task).Error; err != nil { c.JSON(500,gin.H{"code":"50000","msg":"创建重试任务失败"}); return }
	go executeStoredTask(task)
	c.JSON(200,gin.H{"code":"00000","data":task,"msg":"已创建重试任务"})
}
