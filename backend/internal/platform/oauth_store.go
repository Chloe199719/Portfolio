package platform

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/openid"
	"time"
)

type queryer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}
type oauthTxKey struct{}
type OAuthStore struct{ S *Server }

func (o *OAuthStore) db(ctx context.Context) queryer {
	if tx, ok := ctx.Value(oauthTxKey{}).(pgx.Tx); ok {
		return tx
	}
	return o.S.DB
}
func (o *OAuthStore) GetClient(ctx context.Context, key string) (fosite.Client, error) {
	var raw []byte
	e := o.db(ctx).QueryRow(ctx, "SELECT document FROM oauth_clients WHERE id=$1 AND NOT disabled", key).Scan(&raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, fosite.ErrNotFound
	}
	if e != nil {
		return nil, e
	}
	c := &fosite.DefaultOpenIDConnectClient{DefaultClient: &fosite.DefaultClient{}}
	e = json.Unmarshal(raw, c)
	return c, e
}
func (o *OAuthStore) ClientAssertionJWTValid(context.Context, string) error {
	return fosite.ErrRequestUnauthorized
}
func (o *OAuthStore) SetClientAssertionJWT(context.Context, string, time.Time) error {
	return fosite.ErrRequestUnauthorized
}
func (o *OAuthStore) put(ctx context.Context, kind, key string, r fosite.Requester) error {
	raw, e := json.Marshal(r)
	if e != nil {
		return e
	}
	life := 31 * 24 * time.Hour
	if kind == "code" || kind == "pkce" || kind == "oidc" {
		life = 10 * time.Minute
	}
	_, e = o.db(ctx).Exec(ctx, "INSERT INTO oauth_sessions(kind,signature,request_id,account_id,client_id,document,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7)", kind, hash(key), r.GetID(), r.GetSession().GetSubject(), r.GetClient().GetID(), raw, time.Now().Add(life))
	return e
}
func (o *OAuthStore) get(ctx context.Context, kind, key string) (fosite.Requester, error) {
	var raw []byte
	var active bool
	var clientID string
	e := o.db(ctx).QueryRow(ctx, "SELECT document,active,client_id FROM oauth_sessions WHERE kind=$1 AND signature=$2 AND expires_at>now()", kind, hash(key)).Scan(&raw, &active, &clientID)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, fosite.ErrNotFound
	}
	if e != nil {
		return nil, e
	}
	client, e := o.GetClient(ctx, clientID)
	if e != nil {
		return nil, e
	}
	r := &fosite.Request{Client: &fosite.DefaultOpenIDConnectClient{DefaultClient: &fosite.DefaultClient{}}, Session: openid.NewDefaultSession()}
	if e = json.Unmarshal(raw, r); e != nil {
		return nil, e
	}
	r.Client = client
	r.Session.(*openid.DefaultSession).IDTokenHeaders().Add("kid", o.S.Keys.CurrentID())
	if !active {
		if kind == "code" {
			return r, fosite.ErrInvalidatedAuthorizeCode
		}
		return r, fosite.ErrInactiveToken
	}
	return r, nil
}
func (o *OAuthStore) remove(ctx context.Context, kind, key string) error {
	_, e := o.db(ctx).Exec(ctx, "DELETE FROM oauth_sessions WHERE kind=$1 AND signature=$2", kind, hash(key))
	return e
}
func (o *OAuthStore) revoke(ctx context.Context, requestID string) error {
	_, e := o.db(ctx).Exec(ctx, "UPDATE oauth_sessions SET active=false WHERE request_id=$1", requestID)
	return e
}
func (o *OAuthStore) CreateAuthorizeCodeSession(c context.Context, k string, r fosite.Requester) error {
	return o.put(c, "code", k, r)
}
func (o *OAuthStore) GetAuthorizeCodeSession(c context.Context, k string, _ fosite.Session) (fosite.Requester, error) {
	return o.get(c, "code", k)
}
func (o *OAuthStore) InvalidateAuthorizeCodeSession(c context.Context, k string) error {
	tag, e := o.db(c).Exec(c, "UPDATE oauth_sessions SET active=false WHERE kind='code' AND signature=$1 AND active", hash(k))
	if e == nil && tag.RowsAffected() != 1 {
		return fosite.ErrInvalidatedAuthorizeCode
	}
	return e
}
func (o *OAuthStore) CreateAccessTokenSession(c context.Context, k string, r fosite.Requester) error {
	return o.put(c, "access", k, r)
}
func (o *OAuthStore) GetAccessTokenSession(c context.Context, k string, _ fosite.Session) (fosite.Requester, error) {
	return o.get(c, "access", k)
}
func (o *OAuthStore) DeleteAccessTokenSession(c context.Context, k string) error {
	return o.remove(c, "access", k)
}
func (o *OAuthStore) CreateRefreshTokenSession(c context.Context, k, _ string, r fosite.Requester) error {
	return o.put(c, "refresh", k, r)
}
func (o *OAuthStore) GetRefreshTokenSession(c context.Context, k string, _ fosite.Session) (fosite.Requester, error) {
	return o.get(c, "refresh", k)
}
func (o *OAuthStore) DeleteRefreshTokenSession(c context.Context, k string) error {
	return o.remove(c, "refresh", k)
}
func (o *OAuthStore) RotateRefreshToken(c context.Context, requestID, k string) error {
	return o.revoke(c, requestID)
}
func (o *OAuthStore) RevokeRefreshToken(c context.Context, k string) error { return o.revoke(c, k) }
func (o *OAuthStore) RevokeAccessToken(c context.Context, k string) error  { return o.revoke(c, k) }
func (o *OAuthStore) CreateOpenIDConnectSession(c context.Context, k string, r fosite.Requester) error {
	return o.put(c, "oidc", k, r)
}
func (o *OAuthStore) GetOpenIDConnectSession(c context.Context, k string, _ fosite.Requester) (fosite.Requester, error) {
	return o.get(c, "oidc", k)
}
func (o *OAuthStore) DeleteOpenIDConnectSession(c context.Context, k string) error {
	return o.remove(c, "oidc", k)
}
func (o *OAuthStore) CreatePKCERequestSession(c context.Context, k string, r fosite.Requester) error {
	return o.put(c, "pkce", k, r)
}
func (o *OAuthStore) GetPKCERequestSession(c context.Context, k string, _ fosite.Session) (fosite.Requester, error) {
	return o.get(c, "pkce", k)
}
func (o *OAuthStore) DeletePKCERequestSession(c context.Context, k string) error {
	return o.remove(c, "pkce", k)
}
