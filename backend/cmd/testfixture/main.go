// Test-only fixture CLI. Never included in the production Docker image.
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"net/url"
	"os"
	"strings"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	u, e := url.Parse(dsn)
	if e != nil || !strings.HasSuffix(u.Path, "_test") || os.Getenv("APP_ENV") == "production" {
		panic("test fixtures require an isolated *_test database")
	}
	ctx := context.Background()
	db, e := pgxpool.New(ctx, dsn)
	if e != nil {
		panic(e)
	}
	defer db.Close()
	password := "local-browser-test-password"
	pw, e := bcrypt.GenerateFromPassword([]byte(password), 12)
	if e != nil {
		panic(e)
	}
	owner := "45bce303-7014-40c6-ac2d-ccae923829c0"
	visitor := "6973d17e-e3a8-4a2a-b7f1-53cbd88b9013"
	secret := "JBSWY3DPEHPK3PXP"
	for _, a := range []struct {
		id, email, name string
		owner           bool
	}{{owner, "owner@example.test", "Test owner", true}, {visitor, "visitor@example.test", "Test visitor", false}} {
		_, e = db.Exec(ctx, `INSERT INTO accounts(id,email,name,password_hash,verified,owner,totp_secret,totp_confirmed) VALUES($1,$2,$3,$4,true,$5,$6,$5) ON CONFLICT(id) DO UPDATE SET password_hash=$4,totp_secret=$6,totp_confirmed=$5,totp_last=0`, a.id, a.email, a.name, string(pw), a.owner, secret)
		if e != nil {
			panic(e)
		}
	}
	tokens := map[string]string{"password": password, "totpSecret": secret, "ownerID": owner, "visitorID": visitor}
	for name, uid := range map[string]string{"owner": owner, "visitor": visitor} {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		token := base64.RawURLEncoding.EncodeToString(b)
		h := sha256.Sum256([]byte(token))
		_, e = db.Exec(ctx, "INSERT INTO sessions(hash,account_id,kind,mfa,expires_at) VALUES($1,$2,'api',$3,now()+interval '2 hours')", hex.EncodeToString(h[:]), uid, name == "owner")
		if e != nil {
			panic(e)
		}
		tokens[name+"Token"] = token
	}
	if e = json.NewEncoder(os.Stdout).Encode(tokens); e != nil {
		panic(fmt.Errorf("fixture output: %w", e))
	}
}
