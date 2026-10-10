package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) Command(ctx context.Context, args []string) error {
	switch args[0] {
	case "owner":
		if len(args) != 4 {
			return fmt.Errorf("usage: server owner EMAIL NAME PASSWORD_FILE")
		}
		email := strings.ToLower(args[1])
		if !validEmail(email) {
			return bad("Invalid owner email.")
		}
		pw, e := os.ReadFile(args[3])
		if e != nil {
			return e
		}
		hashed, e := passwordHash(strings.TrimSpace(string(pw)))
		if e != nil {
			return e
		}
		uid := id()
		if _, e = s.DB.Exec(ctx, "INSERT INTO accounts(id,email,name,password_hash,verified,owner) VALUES($1,$2,$3,$4,true,true)", uid, email, args[2], hashed); e != nil {
			return e
		}
		fmt.Printf("Owner created: %s\nSign in at %s/login with your email and password, optionally enable an authenticator from your account settings.\n", uid, s.C.Issuer)
		return nil
	case "rotate-keys":
		if e := s.Keys.Rotate(); e != nil {
			return e
		}
		fmt.Println("Signing key rotated; previous public keys retained. Restart the Go service to load it.")
		return nil
	case "map-legacy-owner":
		if len(args) != 3 {
			return fmt.Errorf("usage: server map-legacy-owner LEGACY_UID ACCOUNT_UUID")
		}
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			return e
		}
		defer tx.Rollback(ctx)
		if _, e = tx.Exec(ctx, "INSERT INTO legacy_ownership_map(legacy_uid,account_id) VALUES($1,$2)", args[1], args[2]); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE guestbook SET author_uid=$2 WHERE author_uid=$1", args[1], args[2]); e != nil {
			return e
		}
		return tx.Commit(ctx)
	case "import-media":
		files, e := os.ReadDir(filepath.Join(s.C.DataDir, "media"))
		if e != nil {
			return e
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			ref := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
			if !validAsset.MatchString(ref) {
				continue
			}
			file, e := os.Open(filepath.Join(s.C.DataDir, "media", f.Name()))
			if e != nil {
				return e
			}
			cfg, _, e := image.DecodeConfig(file)
			file.Close()
			if e != nil {
				return e
			}
			if _, e = s.DB.Exec(ctx, "INSERT INTO assets(id,filename,width,height) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING", ref, f.Name(), cfg.Width, cfg.Height); e != nil {
				return e
			}
		}
		fmt.Println("Existing media registered; asset references preserved.")
		return nil
	case "check":
		var counts map[string]int = map[string]int{}
		for _, table := range []string{"content", "accounts", "assets", "schedules", "email_jobs"} {
			var n int
			if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n); e != nil {
				return e
			}
			counts[table] = n
		}
		return json.NewEncoder(os.Stdout).Encode(counts)
	default:
		return fmt.Errorf("commands: owner, rotate-keys, map-legacy-owner, import-media, check")
	}
}
