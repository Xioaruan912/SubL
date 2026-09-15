package api

import (
	"strings"
	"testing"

	"ppeelink/node"
)

const clashSourceYAML = `proxies:
  - name: yaml-node
    type: ss
    server: 1.2.3.4
    port: 8388
    cipher: aes-256-gcm
    password: pass123
`

func TestNodesFromSubscriptionBodyLinkList(t *testing.T) {
	body := []byte("vless://uuid-1@1.2.3.4:443?encryption=none&security=tls&type=tcp#node-a\n\n")
	nodes, format, err := nodesFromSubscriptionBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != "link-list" {
		t.Fatalf("unexpected format: %s", format)
	}
	if len(nodes) != 1 || nodes[0].Name != "node-a" {
		t.Fatalf("unexpected nodes: %#v", nodes)
	}
}

func TestNodesFromSubscriptionBodyBase64(t *testing.T) {
	raw := "vless://uuid-2@1.2.3.4:443?encryption=none&security=tls&type=tcp#node-b"
	body := []byte(node.Base64Encode(raw))
	nodes, format, err := nodesFromSubscriptionBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != "base64-link-list" {
		t.Fatalf("unexpected format: %s", format)
	}
	if len(nodes) != 1 || nodes[0].Name != "node-b" {
		t.Fatalf("unexpected nodes: %#v", nodes)
	}
}

func TestNodesFromSubscriptionBodyClashYAML(t *testing.T) {
	nodes, format, err := nodesFromSubscriptionBody([]byte(clashSourceYAML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != "clash-yaml" {
		t.Fatalf("unexpected format: %s", format)
	}
	if len(nodes) != 1 || nodes[0].Name != "yaml-node" {
		t.Fatalf("unexpected nodes: %#v", nodes)
	}
	if !strings.HasPrefix(nodes[0].Link, "ss://") {
		t.Fatalf("unexpected link: %s", nodes[0].Link)
	}
}

func TestNodesFromSubscriptionBodyGarbage(t *testing.T) {
	_, _, err := nodesFromSubscriptionBody([]byte("not a subscription at all"))
	if err == nil {
		t.Fatal("expected error for unrecognized body")
	}
}
