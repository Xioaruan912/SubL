package api

import (
	"testing"
	"time"
)

func TestLoginRateLimit(t *testing.T){key:="127.0.0.1|security-test";loginAttempts.Lock();delete(loginAttempts.Items,key);loginAttempts.Unlock();t.Cleanup(func(){loginAttempts.Lock();delete(loginAttempts.Items,key);loginAttempts.Unlock()});if blocked,_:=loginBlocked(key);blocked{t.Fatal("fresh key unexpectedly blocked")};for i:=0;i<loginMaxFailures;i++{recordLoginFailure(key)};blocked,remaining:=loginBlocked(key);if !blocked||remaining<=0||remaining>loginBlock+time.Second{t.Fatalf("expected login block, blocked=%v remaining=%v",blocked,remaining)};clearLoginFailures(key);if blocked,_:=loginBlocked(key);blocked{t.Fatal("clear should remove block")}}
