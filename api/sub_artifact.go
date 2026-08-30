package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"ppeelink/models"
	"ppeelink/node"
)

type subscriptionArtifactMeta struct {
	ArtifactID       uint   `json:"artifactId"`
	SubscriptionID   int    `json:"subscriptionId"`
	Client           string `json:"client"`
	Bytes            int    `json:"bytes"`
	SHA256           string `json:"sha256"`
	ValidationStatus string `json:"validationStatus"`
	TestStatus       string `json:"testStatus"`
	PromotedLKG      bool   `json:"promotedLastKnownGood"`
}

func shaHex(data []byte) string { sum:=sha256.Sum256(data);return hex.EncodeToString(sum[:]) }

func subscriptionTemplateForClient(sub *models.Subcription, client string) (string,string) {
	var cfg models.SubscriptionConfig; _=json.Unmarshal([]byte(sub.Config),&cfg);name:=""
	switch strings.ToLower(client){case "clash":name=cfg.Clash;case "surge":name=cfg.Surge;case "loon":name=cfg.Loon}
	if name==""{return "",""};clean:=strings.TrimPrefix(filepathToSlash(name),"./template/");path,err:=safeFilePath(clean);if err!=nil{return name,""};body,err:=os.ReadFile(path);if err!=nil{return name,""};return clean,string(body)
}

func filepathToSlash(value string) string { return strings.ReplaceAll(value,"\\","/") }

func subscriptionInputDigest(sub *models.Subcription, client string) (inputDigest,templateName,templateChecksum,rulesChecksum string,error error) {
	if err:=mergeGroupNodes(sub);err!=nil{return "","","","",err}
	templateName,content:=subscriptionTemplateForClient(sub,client);if content!=""{templateChecksum=shaHex([]byte(content))}
	parts:=[]string{strconv.Itoa(sub.ID),client,sub.Config,sub.Pipeline,sub.SourceURLs,sub.NodeOrder,templateChecksum}
	for _,n:=range sub.Nodes{parts=append(parts,strconv.Itoa(n.ID),n.Name,shaHex([]byte(n.Link)))}
	var catalogs []models.RuleCatalog;_ = models.DB.Where("checksum <> ''").Order("external_id asc").Find(&catalogs).Error;refs:=[]string{}
	for _,r:=range catalogs{if content!=""&&strings.Contains(strings.ToLower(content),strings.ToLower(r.Name)){refs=append(refs,r.ExternalID+":"+r.Checksum)}};sort.Strings(refs);rulesChecksum=shaHex([]byte(strings.Join(refs,"\n")));parts=append(parts,rulesChecksum)
	return shaHex([]byte(strings.Join(parts,"\x00"))),templateName,templateChecksum,rulesChecksum,nil
}

func validateSubscriptionContent(client string, body []byte) error {
	if len(body)==0{return fmt.Errorf("订阅产物为空")}
	switch strings.ToLower(client){case "clash":var root map[string]any;if err:=yaml.Unmarshal(body,&root);err!=nil{return fmt.Errorf("Clash YAML 无效: %w",err)};if len(root)==0{return fmt.Errorf("Clash YAML 为空")};case "surge":if !strings.Contains(string(body),"["){return fmt.Errorf("Surge 配置缺少段落")};case "loon":if !strings.Contains(string(body),"["){return fmt.Errorf("Loon 配置缺少段落")};case "v2ray":decoded:=node.Base64Decode(string(body));if decoded==""||!strings.Contains(decoded,"://"){return fmt.Errorf("V2Ray 订阅未解析到节点")};default:return fmt.Errorf("未知客户端 %s",client)};return nil
}

func buildAndSnapshotSubscription(ctx context.Context, subscriptionID int, client string) (subscriptionArtifactMeta,error) {
	body,err:=buildSubscriptionOutput(subscriptionID,client);if err!=nil{return subscriptionArtifactMeta{},err}
	var sub models.Subcription;if err:=models.DB.First(&sub,subscriptionID).Error;err!=nil{return subscriptionArtifactMeta{},err}
	inputDigest,templateName,templateChecksum,rulesChecksum,err:=subscriptionInputDigest(&sub,client);if err!=nil{return subscriptionArtifactMeta{},err}
	validationStatus:="valid";validationErr:=validateSubscriptionContent(client,body);if validationErr!=nil{validationStatus="invalid"}
	testStatus:="passed";report:=map[string]any{"validation":validationStatus}
	if validationErr!=nil{report["validationError"]=validationErr.Error();testStatus="failed"}
	if validationErr==nil&&strings.EqualFold(client,"clash"){
		plan,planErr:=runSubscriptionEgressPlanTask(ctx,subscriptionID);if planErr!=nil{testStatus="warning";report["egressError"]=planErr.Error()}else{passed,failed:=0,0;for _,item:=range plan.Items{if item.SelectedNode!=nil&&item.Result!=nil&&(item.Result.Status=="available"||item.Result.Status=="reachable"){passed++}else{failed++}};report["egressPassed"]=passed;report["egressFailed"]=failed;report["egressWarnings"]=plan.Warnings;if failed>0||passed==0{testStatus="warning"}}
	}
	reportJSON,_:=json.Marshal(report);artifact:=models.SubscriptionArtifact{SubscriptionID:subscriptionID,Client:strings.ToLower(client),InputDigest:inputDigest,TemplateName:templateName,TemplateChecksum:templateChecksum,RulesChecksum:rulesChecksum,ContentChecksum:shaHex(body),ByteSize:len(body),ValidationStatus:validationStatus,TestStatus:testStatus,TestReportJSON:string(reportJSON),Content:append([]byte(nil),body...)}
	if err:=models.DB.Create(&artifact).Error;err!=nil{return subscriptionArtifactMeta{},err}
	promoted:=validationStatus=="valid"&&testStatus!="failed"
	if promoted{pointer:=models.SubscriptionArtifactPointer{SubscriptionID:subscriptionID,Client:strings.ToLower(client),LastKnownGoodArtifactID:artifact.ID};var existing models.SubscriptionArtifactPointer;err:=models.DB.Where("subscription_id = ? AND client = ?",subscriptionID,strings.ToLower(client)).First(&existing).Error;if err==nil{_ = models.DB.Model(&existing).Update("last_known_good_artifact_id",artifact.ID).Error}else{_ = models.DB.Create(&pointer).Error}}
	return subscriptionArtifactMeta{ArtifactID:artifact.ID,SubscriptionID:subscriptionID,Client:client,Bytes:len(body),SHA256:artifact.ContentChecksum,ValidationStatus:validationStatus,TestStatus:testStatus,PromotedLKG:promoted},nil
}

func lastKnownGoodArtifact(subscriptionID int, client string) (*models.SubscriptionArtifact,error) { var pointer models.SubscriptionArtifactPointer;if err:=models.DB.Where("subscription_id = ? AND client = ?",subscriptionID,strings.ToLower(client)).First(&pointer).Error;err!=nil{return nil,err};var artifact models.SubscriptionArtifact;if err:=models.DB.First(&artifact,pointer.LastKnownGoodArtifactID).Error;err!=nil{return nil,err};return &artifact,nil }

func serveSubscriptionClient(c *gin.Context, subscriptionID int, client string) {
	client=strings.ToLower(client);body,err:=buildSubscriptionOutput(subscriptionID,client);liveValid:=err==nil&&validateSubscriptionContent(client,body)==nil
	if !liveValid { if artifact,lkgErr:=lastKnownGoodArtifact(subscriptionID,client);lkgErr==nil&&len(artifact.Content)>0 { body=artifact.Content;err=nil;c.Header("X-SubLinkX-Artifact","last-known-good");c.Header("X-SubLinkX-Artifact-ID",strconv.FormatUint(uint64(artifact.ID),10)) } }
	if err!=nil||len(body)==0 { c.String(502,"订阅生成失败，且没有可用的 Last Known Good 版本");return }
	var sub models.Subcription;_ = models.DB.First(&sub,subscriptionID).Error;ext:=map[string]string{"clash":"yaml","surge":"conf","loon":"conf","v2ray":"txt"}[client];filename:=sub.Name+"."+ext;c.Header("Content-Disposition","inline; filename*=utf-8''"+url.QueryEscape(filename));c.Header("X-SubLinkX-Artifact-Source",map[bool]string{true:"live",false:"fallback"}[liveValid]);c.Data(200,"text/plain; charset=utf-8",body)
}

func SubscriptionArtifacts(c *gin.Context){id,_:=strconv.Atoi(c.Query("id"));client:=strings.ToLower(strings.TrimSpace(c.Query("client")));q:=models.DB.Model(&models.SubscriptionArtifact{}).Select("id,subscription_id,client,input_digest,template_name,template_checksum,rules_checksum,content_checksum,byte_size,validation_status,test_status,test_report_json,created_at").Where("subscription_id = ?",id);if client!=""{q=q.Where("client = ?",client)};var items []models.SubscriptionArtifact;if err:=q.Order("id desc").Limit(100).Find(&items).Error;err!=nil{c.JSON(500,gin.H{"code":"50000","msg":"读取产物版本失败"});return};var pointers []models.SubscriptionArtifactPointer;_ = models.DB.Where("subscription_id = ?",id).Find(&pointers).Error;c.JSON(200,gin.H{"code":"00000","data":gin.H{"items":items,"pointers":pointers},"msg":"订阅产物版本"})}

func SubscriptionArtifactRollback(c *gin.Context){artifactID64,_:=strconv.ParseUint(c.Query("artifactId"),10,64);var artifact models.SubscriptionArtifact;if artifactID64==0||models.DB.First(&artifact,uint(artifactID64)).Error!=nil{c.JSON(404,gin.H{"code":"40400","msg":"产物版本不存在"});return};if artifact.ValidationStatus!="valid"{c.JSON(409,gin.H{"code":"40900","msg":"无效产物不能设为 Last Known Good"});return};var pointer models.SubscriptionArtifactPointer;err:=models.DB.Where("subscription_id = ? AND client = ?",artifact.SubscriptionID,artifact.Client).First(&pointer).Error;if err==nil{_ = models.DB.Model(&pointer).Update("last_known_good_artifact_id",artifact.ID).Error}else{_ = models.DB.Create(&models.SubscriptionArtifactPointer{SubscriptionID:artifact.SubscriptionID,Client:artifact.Client,LastKnownGoodArtifactID:artifact.ID}).Error};c.JSON(200,gin.H{"code":"00000","msg":"Last Known Good 已切换"})}
