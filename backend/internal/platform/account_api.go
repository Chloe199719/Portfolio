package platform

import (
	"context"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

type AccountMutation struct {
	Action   string `json:"action"`
	Current  string `json:"current"`
	Password string `json:"password"`
	Provider string `json:"provider"`
	ID       string `json:"id"`
}

func (s *Server) changeAccount(ctx context.Context, u *User, in AccountMutation) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	// Serialize grant revocation with token exchanges so a concurrent refresh
	// cannot issue a new token after the account operation revoked its family.
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(718299)"); e != nil {
		return e
	}
	var old string
	if e = tx.QueryRow(ctx, "SELECT password_hash FROM accounts WHERE id=$1 FOR UPDATE", u.ID).Scan(&old); e != nil {
		return e
	}
	switch in.Action {
	case "password":
		if old != "" && bcrypt.CompareHashAndPassword([]byte(old), []byte(in.Current)) != nil {
			return bad("Current password is incorrect.")
		}
		pw, err := passwordHash(in.Password)
		if err != nil {
			return err
		}
		_, e = tx.Exec(ctx, "UPDATE accounts SET password_hash=$2 WHERE id=$1", u.ID, pw)
		if e == nil {
			_, e = tx.Exec(ctx, "DELETE FROM sessions WHERE account_id=$1 AND hash!=$2", u.ID, u.SessionHash)
		}
		if e == nil {
			_, e = tx.Exec(ctx, "UPDATE oauth_sessions SET active=false WHERE account_id=$1", u.ID)
		}
	case "unlink", "remove-passkey":
		var methods int
		e = tx.QueryRow(ctx, "SELECT (CASE WHEN password_hash!='' THEN 1 ELSE 0 END)+(SELECT count(*) FROM social_accounts WHERE account_id=$1)+(SELECT count(*) FROM passkeys WHERE account_id=$1) FROM accounts WHERE id=$1", u.ID).Scan(&methods)
		if e != nil {
			return e
		}
		if methods <= 1 {
			return bad("Keep at least one sign-in method.")
		}
		if in.Action == "unlink" {
			_, e = tx.Exec(ctx, "DELETE FROM social_accounts WHERE account_id=$1 AND provider=$2", u.ID, in.Provider)
		} else {
			_, e = tx.Exec(ctx, "DELETE FROM passkeys WHERE account_id=$1 AND id=$2", u.ID, in.ID)
		}
	case "revoke-session":
		_, e = tx.Exec(ctx, "DELETE FROM sessions WHERE account_id=$1 AND hash=$2", u.ID, in.ID)
	case "revoke-app":
		_, e = tx.Exec(ctx, "UPDATE oauth_sessions SET active=false WHERE account_id=$1 AND client_id=$2", u.ID, in.ID)
		if e == nil {
			_, e = tx.Exec(ctx, "DELETE FROM oauth_consents WHERE account_id=$1 AND client_id=$2", u.ID, in.ID)
		}
	default:
		return bad("Unknown account action.")
	}
	if e != nil {
		return e
	}
	if s.C.SMTPUser != "" {
		_, e = tx.Exec(ctx, "INSERT INTO email_jobs(id,recipient,subject,body) VALUES($1,$2,$3,$4)", id(), u.Email, "Chloe ID security update", "An account setting changed: "+in.Action+".\n\nReview your active sessions and sign-in methods at "+s.C.Issuer+"/account. If this was not you, reset your password.")
		if e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}

func (s *Server) accountAPI(w http.ResponseWriter, r *http.Request) {
	u := s.require(w, r, false)
	if u == nil {
		return
	}
	if u.Owner && !u.MFA {
		fail(w, problem{403, "Owner account access requires an additional factor."})
		return
	}
	ctx := r.Context()
	if r.Method == "PATCH" {
		if e := s.limit(ctx, "account:"+u.ID, 30, time.Hour); e != nil {
			fail(w, e)
			return
		}
		var in AccountMutation
		if e := decode(w, r, &in); e != nil {
			fail(w, e)
			return
		}
		if e := s.changeAccount(ctx, u, in); e != nil {
			fail(w, e)
			return
		}
		s.audit(ctx, u.ID, "account."+in.Action, "")
		send(w, 200, map[string]any{"ok": true})
		return
	}
	var data Document
	e := s.DB.QueryRow(ctx, `SELECT jsonb_build_object(
 'providers',coalesce((SELECT jsonb_agg(provider) FROM social_accounts WHERE account_id=$1),'[]'),
 'passkeys',coalesce((SELECT jsonb_agg(jsonb_build_object('id',id,'name',name)) FROM passkeys WHERE account_id=$1),'[]'),
 'sessions',coalesce((SELECT jsonb_agg(jsonb_build_object('id',hash,'kind',kind,'agent',agent,'createdAt',created_at)) FROM sessions WHERE account_id=$1 AND expires_at>now()),'[]'),
 'applications',coalesce((SELECT jsonb_agg(jsonb_build_object('id',c.id,'name',c.name)) FROM oauth_consents o JOIN oauth_clients c ON c.id=o.client_id WHERE o.account_id=$1),'[]'))`, u.ID).Scan(&data)
	if e != nil {
		fail(w, e)
		return
	}
	data["user"] = u
	send(w, 200, data)
}
