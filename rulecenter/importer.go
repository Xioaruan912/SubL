package rulecenter

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type ImportOptions struct {
	ProviderName   string
	URL            string
	Policy         string
	Behavior       string
	Format         string
	Proxy          string
	Path           string
	ConflictPolicy string
	Interval       int
}

type ImportResult struct {
	Text       string   `json:"-"`
	Provider   string   `json:"provider"`
	Rule       string   `json:"rule"`
	Changed    bool     `json:"changed"`
	Conflict   bool     `json:"conflict"`
	Warnings   []string `json:"warnings,omitempty"`
}

func ImportClashProvider(text string, opt ImportOptions) (ImportResult, error) {
	if strings.TrimSpace(opt.ProviderName)=="" || strings.TrimSpace(opt.URL)=="" || strings.TrimSpace(opt.Policy)=="" { return ImportResult{},errors.New("provider、URL 和策略组不能为空") }
	if opt.Behavior=="" { opt.Behavior="classical" }; if opt.Format=="" { opt.Format="yaml" }; if opt.Interval<=0 {opt.Interval=3600}; if opt.Path=="" { opt.Path="./rules/"+safeProviderName(opt.ProviderName)+".yaml" }; if opt.ConflictPolicy==""{opt.ConflictPolicy="keep"}
	var doc yaml.Node
	if err:=yaml.Unmarshal([]byte(text),&doc);err!=nil{return ImportResult{},err}
	if len(doc.Content)==0{return ImportResult{},errors.New("模板为空")}
	root:=doc.Content[0]; if root.Kind!=yaml.MappingNode{return ImportResult{},errors.New("Clash 模板顶层必须为 YAML mapping")}
	providers:=ensureMapping(root,"rule-providers")
	provider,exists:=mappingValue(providers,opt.ProviderName)
	conflict:=false
	if exists {
		oldURL:=mappingScalar(provider,"url")
		if oldURL!=opt.URL { conflict=true; switch opt.ConflictPolicy {case "keep": case "update-url": setMappingScalar(provider,"url",opt.URL); applyProviderOptions(provider,opt); case "replace": replaceProvider(provider,opt); default:return ImportResult{},fmt.Errorf("未知冲突策略: %s",opt.ConflictPolicy)} } else { applyProviderOptions(provider,opt) }
	} else { appendProvider(providers,opt) }
	ruleLine:="RULE-SET,"+opt.ProviderName+","+opt.Policy
	rulesNode:=ensureSequence(root,"rules")
	inserted:=insertRuleBeforeMatch(rulesNode,ruleLine)
	if !exists { inserted=true }
	out,err:=marshalYAML(&doc);if err!=nil{return ImportResult{},err}
	return ImportResult{Text:out,Provider:opt.ProviderName,Rule:ruleLine,Changed:inserted || (conflict && opt.ConflictPolicy!="keep"),Conflict:conflict},nil
}

func ensureMapping(root *yaml.Node,key string)*yaml.Node{if v,ok:=mappingValue(root,key);ok&&v.Kind==yaml.MappingNode{return v};k:=&yaml.Node{Kind:yaml.ScalarNode,Tag:"!!str",Value:key};v:=&yaml.Node{Kind:yaml.MappingNode,Tag:"!!map"};root.Content=append(root.Content,k,v);return v}
func ensureSequence(root *yaml.Node,key string)*yaml.Node{if v,ok:=mappingValue(root,key);ok&&v.Kind==yaml.SequenceNode{return v};k:=&yaml.Node{Kind:yaml.ScalarNode,Tag:"!!str",Value:key};v:=&yaml.Node{Kind:yaml.SequenceNode,Tag:"!!seq"};root.Content=append(root.Content,k,v);return v}
func mappingValue(m *yaml.Node,key string)(*yaml.Node,bool){if m==nil||m.Kind!=yaml.MappingNode{return nil,false};for i:=0;i+1<len(m.Content);i+=2{if m.Content[i].Value==key{return m.Content[i+1],true}};return nil,false}
func mappingScalar(m *yaml.Node,key string)string{v,ok:=mappingValue(m,key);if !ok{return ""};return v.Value}
func setMappingScalar(m *yaml.Node,key,value string){for i:=0;i+1<len(m.Content);i+=2{if m.Content[i].Value==key{m.Content[i+1].Kind=yaml.ScalarNode;m.Content[i+1].Tag="!!str";m.Content[i+1].Value=value;return}};m.Content=append(m.Content,&yaml.Node{Kind:yaml.ScalarNode,Tag:"!!str",Value:key},&yaml.Node{Kind:yaml.ScalarNode,Tag:"!!str",Value:value})}
func setMappingInt(m *yaml.Node,key string,value int){for i:=0;i+1<len(m.Content);i+=2{if m.Content[i].Value==key{m.Content[i+1].Kind=yaml.ScalarNode;m.Content[i+1].Tag="!!int";m.Content[i+1].Value=fmt.Sprint(value);return}};m.Content=append(m.Content,&yaml.Node{Kind:yaml.ScalarNode,Tag:"!!str",Value:key},&yaml.Node{Kind:yaml.ScalarNode,Tag:"!!int",Value:fmt.Sprint(value)})}
func replaceProvider(n *yaml.Node,opt ImportOptions){n.Kind=yaml.MappingNode;n.Tag="!!map";n.Content=nil;setMappingScalar(n,"type","http");setMappingScalar(n,"behavior",opt.Behavior);setMappingInt(n,"interval",opt.Interval);setMappingScalar(n,"format",opt.Format);setMappingScalar(n,"proxy",opt.Proxy);setMappingScalar(n,"path",opt.Path);setMappingScalar(n,"url",opt.URL)}
func applyProviderOptions(n *yaml.Node,opt ImportOptions){setMappingScalar(n,"type","http");setMappingScalar(n,"behavior",opt.Behavior);setMappingInt(n,"interval",opt.Interval);setMappingScalar(n,"format",opt.Format);setMappingScalar(n,"proxy",opt.Proxy);setMappingScalar(n,"path",opt.Path)}
func appendProvider(providers *yaml.Node,opt ImportOptions){k:=&yaml.Node{Kind:yaml.ScalarNode,Tag:"!!str",Value:opt.ProviderName};v:=&yaml.Node{Kind:yaml.MappingNode,Tag:"!!map"};replaceProvider(v,opt);providers.Content=append(providers.Content,k,v)}
func safeProviderName(s string)string{r:=strings.NewReplacer("/","_","\\","_"," ","-");return r.Replace(strings.TrimSpace(s))}
func insertRuleBeforeMatch(seq *yaml.Node,line string)bool{for _,n:=range seq.Content{if strings.TrimSpace(n.Value)==line{return false}};item:=&yaml.Node{Kind:yaml.ScalarNode,Tag:"!!str",Value:line};idx:=len(seq.Content);for i,n:=range seq.Content{if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(n.Value)),"MATCH,"){idx=i;break}};seq.Content=append(seq.Content,nil);copy(seq.Content[idx+1:],seq.Content[idx:]);seq.Content[idx]=item;return true}
func marshalYAML(doc *yaml.Node)(string,error){var b strings.Builder;e:=yaml.NewEncoder(&b);e.SetIndent(2);if err:=e.Encode(doc);err!=nil{return "",err};_ = e.Close();return b.String(),nil}
