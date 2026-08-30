package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"ppeelink/models"
)

type safeBackupAlert struct { Enabled bool `json:"enabled"`; FailureThreshold int `json:"failureThreshold"`; MaintenanceStart string `json:"maintenanceStart"`; MaintenanceEnd string `json:"maintenanceEnd"` }
type safeBackupSubscription struct { Name string `json:"name"`; Config string `json:"config"`; Pipeline string `json:"pipeline"` }
type safeBackupAirport struct { Name string `json:"name"`; AutoCleanup bool `json:"autoCleanup"`; IsDedicated bool `json:"isDedicated"` }
type safeConfigBackup struct {
	Version int `json:"version"`; GeneratedAt time.Time `json:"generatedAt"`; Templates map[string]string `json:"templates"`; EgressTargets []models.EgressTarget `json:"egressTargets"`; Alert safeBackupAlert `json:"alert"`; Subscriptions []safeBackupSubscription `json:"subscriptions"`; Airports []safeBackupAirport `json:"airports"`
}

func buildSafeConfigBackup() (safeConfigBackup,error) {
	out:=safeConfigBackup{Version:1,GeneratedAt:time.Now(),Templates:map[string]string{},EgressTargets:[]models.EgressTarget{},Subscriptions:[]safeBackupSubscription{},Airports:[]safeBackupAirport{}}
	entries,err:=os.ReadDir("template");if err==nil{for _,entry:=range entries{if entry.IsDir(){continue};name:=entry.Name();path,err:=safeFilePath(name);if err!=nil{continue};body,err:=os.ReadFile(path);if err==nil{out.Templates[name]=string(body)}}}
	if err:=models.DB.Order("sort_order asc,id asc").Find(&out.EgressTargets).Error;err!=nil{return out,err};for i:=range out.EgressTargets{out.EgressTargets[i].ID=0;out.EgressTargets[i].CreatedAt=time.Time{};out.EgressTargets[i].UpdatedAt=time.Time{}}
	var alert models.AlertSetting;if models.DB.First(&alert).Error==nil{out.Alert=safeBackupAlert{Enabled:alert.Enabled,FailureThreshold:alert.FailureThreshold,MaintenanceStart:alert.MaintenanceStart,MaintenanceEnd:alert.MaintenanceEnd}}
	var subs []models.Subcription;if err:=models.DB.Select("name,config,pipeline").Find(&subs).Error;err!=nil{return out,err};for _,sub:=range subs{out.Subscriptions=append(out.Subscriptions,safeBackupSubscription{Name:sub.Name,Config:sub.Config,Pipeline:sub.Pipeline})}
	var airports []models.Airport;if err:=models.DB.Select("name,auto_cleanup,is_dedicated").Find(&airports).Error;err!=nil{return out,err};for _,a:=range airports{out.Airports=append(out.Airports,safeBackupAirport{Name:a.Name,AutoCleanup:a.AutoCleanup,IsDedicated:a.IsDedicated})}
	return out,nil
}

func ConfigBackupExport(c *gin.Context){backup,err:=buildSafeConfigBackup();if err!=nil{c.JSON(500,gin.H{"code":"50000","msg":"生成配置备份失败: "+err.Error()});return};c.Header("Content-Disposition","attachment; filename=sublinkx-safe-backup.json");c.JSON(200,backup)}

func ConfigBackupImport(c *gin.Context){var backup safeConfigBackup;if err:=c.ShouldBindJSON(&backup);err!=nil||backup.Version!=1{c.JSON(400,gin.H{"code":"40000","msg":"备份格式或版本无效"});return};templates,targets,subs,airports:=0,0,0,0
	for name,content:=range backup.Templates{name=strings.TrimSpace(name);if name==""||filepath.Base(name)!=name{continue};path,err:=safeFilePath(name);if err!=nil{continue};if old,readErr:=os.ReadFile(path);readErr==nil{_ = models.SaveTemplateVersion(name,string(old),"before_safe_backup_restore")};if err:=os.WriteFile(path,[]byte(content),0666);err==nil{_ = models.SaveTemplateVersion(name,content,"safe_backup_restore");templates++}}
	for _,item:=range backup.EgressTargets{models.NormalizeEgressTarget(&item);if validateEgressTarget(&item)!=""{continue};var existing models.EgressTarget;err:=models.DB.Where("key = ?",item.Key).First(&existing).Error;if err==nil{updates:=map[string]any{"name":item.Name,"domain":item.Domain,"group":item.Group,"icon":item.Icon,"path":item.Path,"method":item.Method,"expected_status":item.ExpectedStatus,"response_contains":item.ResponseContains,"require_egress_ip":item.RequireEgressIP,"timeout_seconds":item.TimeoutSeconds,"retries":item.Retries,"enabled":item.Enabled,"sort_order":item.SortOrder};if models.DB.Model(&existing).Updates(updates).Error==nil{targets++}}else{item.ID=0;if models.DB.Create(&item).Error==nil{targets++}}}
	for _,item:=range backup.Subscriptions{if strings.TrimSpace(item.Name)==""{continue};result:=models.DB.Model(&models.Subcription{}).Where("name = ?",item.Name).Updates(map[string]any{"config":item.Config,"pipeline":item.Pipeline});if result.Error==nil&&result.RowsAffected>0{subs++}}
	for _,item:=range backup.Airports{if strings.TrimSpace(item.Name)==""{continue};result:=models.DB.Model(&models.Airport{}).Where("name = ?",item.Name).Updates(map[string]any{"auto_cleanup":item.AutoCleanup,"is_dedicated":item.IsDedicated});if result.Error==nil&&result.RowsAffected>0{airports++}}
	var alert models.AlertSetting;if models.DB.First(&alert).Error==nil{_ = models.DB.Model(&alert).Updates(map[string]any{"enabled":backup.Alert.Enabled,"failure_threshold":backup.Alert.FailureThreshold,"maintenance_start":backup.Alert.MaintenanceStart,"maintenance_end":backup.Alert.MaintenanceEnd}).Error}
	c.JSON(200,gin.H{"code":"00000","data":gin.H{"templates":templates,"egressTargets":targets,"subscriptions":subs,"airports":airports},"msg":"安全配置已合并恢复；敏感凭据和节点关系未被修改"})}

func ConfigBackupInspect(c *gin.Context){backup,err:=buildSafeConfigBackup();if err!=nil{c.JSON(500,gin.H{"code":"50000","msg":"读取配置失败"});return};raw,_:=json.Marshal(backup);c.JSON(200,gin.H{"code":"00000","data":gin.H{"version":backup.Version,"templates":len(backup.Templates),"egressTargets":len(backup.EgressTargets),"subscriptions":len(backup.Subscriptions),"airports":len(backup.Airports),"bytes":len(raw)},"msg":"安全备份摘要"})}
