package rulecenter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProviderDefinition struct {
	Type     string `yaml:"type"`
	Behavior string `yaml:"behavior"`
	URL      string `yaml:"url"`
	Path     string `yaml:"path"`
}

func ProviderDefinitions(template string) map[string]ProviderDefinition {
	var root struct {
		Providers map[string]ProviderDefinition `yaml:"rule-providers"`
	}
	if yaml.Unmarshal([]byte(template), &root) != nil || root.Providers == nil {
		return map[string]ProviderDefinition{}
	}
	return root.Providers
}

func ResolveProviderRules(ctx context.Context, template, name string) ([]NormalizedRule, []string, error) {
	def, ok := ProviderDefinitions(template)[name]
	if !ok { return nil,nil,errors.New("模板中未找到 rule-provider: "+name) }
	var data []byte
	var format string
	var err error
	if strings.EqualFold(def.Type,"file") || (def.URL=="" && def.Path!="") {
		path, pathErr := safeLocalProviderPath(def.Path)
		if pathErr != nil { return nil,nil,pathErr }
		data,err=os.ReadFile(path); if err!=nil{return nil,nil,err}
		format=strings.TrimPrefix(strings.ToLower(filepath.Ext(path)),".")
	} else {
		if def.URL=="" { return nil,nil,errors.New("rule-provider 缺少 URL") }
		data,format,err=fetchProviderCached(ctx,def.URL)
		if err!=nil{return nil,nil,err}
	}
	rules,warnings,err:=ParseRules(data,format)
	if err!=nil{return nil,nil,err}
	return rules,warnings,nil
}

func safeLocalProviderPath(raw string)(string,error){
	clean:=filepath.Clean(strings.TrimSpace(raw)); if clean==""||clean=="."||filepath.IsAbs(clean){return "",errors.New("本地 provider path 无效")}
	clean=strings.TrimPrefix(clean,"."+string(os.PathSeparator)); clean=strings.TrimPrefix(filepath.ToSlash(clean),"./")
	if clean==".."||strings.HasPrefix(clean,"../") {return "",errors.New("本地 provider path 越权")}
	root,err:=filepath.Abs("template");if err!=nil{return "",err};full,err:=filepath.Abs(filepath.Join(root,filepath.FromSlash(clean)));if err!=nil{return "",err};rel,err:=filepath.Rel(root,full);if err!=nil||rel==".."||strings.HasPrefix(rel,".."+string(os.PathSeparator)){return "",errors.New("本地 provider path 越权")};return full,nil
}

func fetchProviderCached(ctx context.Context, rawURL string)([]byte,string,error){
	sum:=sha256.Sum256([]byte(rawURL));key:=hex.EncodeToString(sum[:]);ext:=strings.TrimPrefix(strings.ToLower(filepath.Ext(strings.Split(rawURL,"?")[0])),".");if ext==""{ext="yaml"};path:=filepath.Join(cacheRoot,"providers",key+"."+ext)
	body,_,err:=get(ctx,rawURL,4<<20);if err==nil{if writeErr:=atomicCacheWrite(path,body);writeErr!=nil{return nil,"",writeErr};return body,ext,nil}
	cached,readErr:=os.ReadFile(path);if readErr==nil{return cached,ext,nil};return nil,"",err
}
