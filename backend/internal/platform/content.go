package platform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type Document map[string]any

func str(d map[string]any, k string) string            { v, _ := d[k].(string); return v }
func object(d map[string]any, k string) map[string]any { v, _ := d[k].(map[string]any); return v }
func slug(d Document) string                           { return str(object(d, "slug"), "current") }

var validID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,100}$`)
var validSlug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var validAsset = regexp.MustCompile(`^(image-[a-zA-Z0-9]+-[0-9]+x[0-9]+-[a-zA-Z0-9]+|upload-[a-f0-9-]+)$`)
var editableTypes = map[string]bool{"pageInfo": true, "projects": true, "note": true, "photograph": true, "nowUpdate": true, "siteSettings": true}
var fields = map[string]int{"title": 160, "summary": 1500, "date": 10, "category": 80, "location": 150, "linkToBuild": 2048, "projectRole": 500, "problem": 5000, "decisions": 8000, "outcome": 5000, "name": 100, "role": 100, "email": 254, "intro": 1500, "bio": 8000, "homepageTitle": 200, "homepageSubtitle": 1000}

func validLink(v string) bool {
	u, e := url.Parse(v)
	return e == nil && ((u.Scheme == "https" || u.Scheme == "http") && u.Host != "" || u.Scheme == "" && strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "//"))
}
func validateImage(v any, publish bool) error {
	im, ok := v.(map[string]any)
	if !ok || str(im, "_type") != "image" || !validAsset.MatchString(str(object(im, "asset"), "_ref")) {
		return bad("Invalid image reference.")
	}
	for _, k := range []string{"alt", "caption"} {
		if x, exists := im[k]; exists {
			v, ok := x.(string)
			if !ok || utf8.RuneCountInString(v) > 1000 {
				return bad("Invalid image text.")
			}
		}
	}
	if publish && strings.TrimSpace(str(im, "alt")) == "" {
		return bad("Add alternative text to every image.")
	}
	return nil
}
func validateDocument(d Document, publish bool) error {
	typ := str(d, "_type")
	if !editableTypes[typ] {
		return bad("Invalid content type.")
	}
	for k, v := range d {
		if strings.HasPrefix(k, "_") {
			continue
		}
		if max, ok := fields[k]; ok {
			t, ok := v.(string)
			if !ok || utf8.RuneCountInString(t) > max {
				return bad("Invalid field: " + k)
			}
			continue
		}
		switch k {
		case "image", "heroImage", "profilePic":
			if e := validateImage(v, publish); e != nil {
				return e
			}
		case "slug":
			if len(slug(d)) > 160 || !validSlug.MatchString(slug(d)) {
				return bad("Use a lowercase URL slug with hyphens.")
			}
		case "order":
			n, ok := v.(float64)
			if !ok || n < 0 || n > 9999 || float64(int(n)) != n {
				return bad("Invalid display order.")
			}
		case "featured":
			if _, ok := v.(bool); !ok {
				return bad("Invalid featured value.")
			}
		case "tags", "interests":
			a, ok := v.([]any)
			if !ok || len(a) > 15 {
				return bad("Too many tags or interests.")
			}
			for _, x := range a {
				t, ok := x.(string)
				if !ok || len(t) > 100 {
					return bad("Invalid tag.")
				}
			}
		case "socials":
			if a, ok := v.([]any); !ok || len(a) > 30 {
				return bad("Invalid social references.")
			}
		case "technologies":
			a, ok := v.([]any)
			if !ok || len(a) > 50 {
				return bad("Invalid technologies.")
			}
			for _, x := range a {
				m, ok := x.(map[string]any)
				if !ok || len(str(m, "title")) > 100 || len(str(m, "_ref")) > 100 {
					return bad("Invalid technology.")
				}
			}
		case "modules":
			a, ok := v.([]any)
			if !ok || len(a) > 5 {
				return bad("Invalid homepage modules.")
			}
			seen := map[string]bool{}
			for _, x := range a {
				m, ok := x.(map[string]any)
				name := str(m, "id")
				if !ok || !strings.Contains("|work|photos|now|notes|guestbook|", "|"+name+"|") || name == "" || seen[name] {
					return bad("Invalid homepage module.")
				}
				seen[name] = true
				if _, ok := m["visible"].(bool); !ok {
					return bad("Invalid module visibility.")
				}
			}
		case "body":
			a, ok := v.([]any)
			if !ok || len(a) > 500 {
				return bad("Invalid article body.")
			}
			for _, x := range a {
				b, ok := x.(map[string]any)
				if !ok {
					return bad("Invalid article block.")
				}
				if str(b, "_type") == "image" {
					if e := validateImage(b, publish); e != nil {
						return e
					}
					continue
				}
				if str(b, "_type") != "block" {
					return bad("Unsupported article block.")
				}
				if st := str(b, "style"); st != "" && st != "normal" && st != "h2" && st != "h3" && st != "blockquote" {
					return bad("Unsupported text style.")
				}
				children, ok := b["children"].([]any)
				if !ok || len(children) > 1000 {
					return bad("Invalid text spans.")
				}
				for _, c := range children {
					span, ok := c.(map[string]any)
					if !ok || str(span, "_type") != "span" || len(str(span, "text")) > 120000 {
						return bad("Invalid text span.")
					}
				}
				if defs, ok := b["markDefs"].([]any); ok {
					for _, x := range defs {
						m, ok := x.(map[string]any)
						if !ok || str(m, "_type") != "link" || !validLink(str(m, "href")) {
							return bad("Use safe http(s) or internal article links.")
						}
					}
				}
			}
		case "isDraft", "hasPublished":
			delete(d, k)
		default:
			return bad("Unknown content field: " + k)
		}
	}
	if v := str(d, "linkToBuild"); v != "" && !validLink(v) {
		return bad("Invalid project link.")
	}
	if v := str(d, "date"); v != "" {
		if _, e := time.Parse("2006-01-02", v); e != nil {
			return bad("Invalid date.")
		}
	}
	if publish {
		if typ != "pageInfo" && typ != "siteSettings" && strings.TrimSpace(str(d, "title")) == "" {
			return bad("Add a title.")
		}
		if typ == "projects" || typ == "note" {
			if slug(d) == "" {
				return bad("Add a URL slug.")
			}
			if strings.TrimSpace(str(d, "summary")) == "" {
				return bad("Add a description before publishing.")
			}
		}
		if typ == "note" {
			a, _ := d["body"].([]any)
			if len(a) == 0 || str(d, "date") == "" {
				return bad("Add article content and a publication date.")
			}
		}
		if typ == "photograph" && d["image"] == nil {
			return bad("Add a photograph.")
		}
		if typ == "nowUpdate" && (str(d, "summary") == "" || str(d, "date") == "") {
			return bad("Add an update and a date.")
		}
	}
	return nil
}
func refs(v any, out map[string]bool) {
	switch x := v.(type) {
	case map[string]any:
		if str(x, "_type") == "image" {
			if ref := str(object(x, "asset"), "_ref"); validAsset.MatchString(ref) {
				out[ref] = true
			}
		}
		for _, y := range x {
			refs(y, out)
		}
	case Document:
		refs(map[string]any(x), out)
	case []any:
		for _, y := range x {
			refs(y, out)
		}
	}
}
func syncRefs(ctx context.Context, tx pgx.Tx, docID, state string, d map[string]any) error {
	if _, e := tx.Exec(ctx, "DELETE FROM asset_references WHERE content_id=$1 AND state=$2", docID, state); e != nil {
		return e
	}
	found := map[string]bool{}
	refs(d, found)
	for ref := range found {
		if _, e := tx.Exec(ctx, "INSERT INTO asset_references VALUES($1,$2,$3) ON CONFLICT DO NOTHING", ref, docID, state); e != nil {
			return e
		}
	}
	return nil
}
func unpack(d Document, rev string, updated time.Time) Document {
	d["_rev"] = rev
	d["_updatedAt"] = updated.UTC().Format(time.RFC3339Nano)
	return d
}
func (s *Server) published(w http.ResponseWriter, r *http.Request) {
	rows, e := s.DB.Query(r.Context(), "SELECT document,revision,updated_at FROM content WHERE state='published' ORDER BY updated_at,id")
	if e != nil {
		fail(w, e)
		return
	}
	defer rows.Close()
	docs := []Document{}
	for rows.Next() {
		var d Document
		var rev string
		var date time.Time
		if e = rows.Scan(&d, &rev, &date); e != nil {
			fail(w, e)
			return
		}
		docs = append(docs, unpack(d, rev, date))
	}
	if e = rows.Err(); e != nil {
		fail(w, e)
		return
	}
	send(w, 200, map[string]any{"documents": docs})
}
func (s *Server) editable(ctx context.Context, docID string) (Document, error) {
	var d Document
	var rev string
	var updated time.Time
	var draft, published bool
	e := s.DB.QueryRow(ctx, `SELECT document,revision,updated_at,state='draft',EXISTS(SELECT 1 FROM content p WHERE p.id=c.id AND p.state='published') FROM content c WHERE id=$1 ORDER BY state LIMIT 1`, docID).Scan(&d, &rev, &updated, &draft, &published)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, problem{404, "Content not found."}
	}
	if e != nil {
		return nil, e
	}
	unpack(d, rev, updated)
	d["isDraft"] = draft
	d["hasPublished"] = published
	return d, nil
}
func (s *Server) contentIndex(w http.ResponseWriter, r *http.Request) {
	if s.require(w, r, true) == nil {
		return
	}
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	typ, status, search := q.Get("type"), q.Get("status"), q.Get("q")
	args := []any{typ, search, status, (page - 1) * 25}
	rows, e := s.DB.Query(r.Context(), `WITH latest AS(SELECT DISTINCT ON(id) id,type,document,revision,updated_at,state FROM content ORDER BY id,state) SELECT l.id,count(*) OVER() FROM latest l WHERE type IN ('pageInfo','projects','note','photograph','nowUpdate','siteSettings') AND ($1='' OR type=$1) AND ($2='' OR coalesce(document->>'title',document->>'name','') ILIKE '%'||$2||'%') AND ($3='' OR ($3='draft' AND state='draft') OR ($3='published' AND EXISTS(SELECT 1 FROM content p WHERE p.id=l.id AND p.state='published'))) ORDER BY updated_at DESC,id LIMIT 25 OFFSET $4`, args...)
	if e != nil {
		fail(w, e)
		return
	}
	ids := []string{}
	total := 0
	for rows.Next() {
		var v string
		if e = rows.Scan(&v, &total); e != nil {
			rows.Close()
			fail(w, e)
			return
		}
		ids = append(ids, v)
	}
	rows.Close()
	if e = rows.Err(); e != nil {
		fail(w, e)
		return
	}
	docs := []Document{}
	for _, v := range ids {
		d, e := s.editable(r.Context(), v)
		if e != nil {
			fail(w, e)
			return
		}
		if editableTypes[str(d, "_type")] {
			docs = append(docs, d)
		}
	}
	send(w, 200, map[string]any{"documents": docs, "total": total, "page": page, "pageSize": 25})
}

type ContentMutation struct {
	Action   string   `json:"action"`
	Revision string   `json:"revision"`
	Document Document `json:"document"`
	Unset    []string `json:"unset"`
}

func (s *Server) contentItem(w http.ResponseWriter, r *http.Request) {
	u := s.require(w, r, true)
	if u == nil {
		return
	}
	docID := r.PathValue("id")
	if !validID.MatchString(docID) {
		fail(w, bad("Invalid identifier."))
		return
	}
	if r.Method == "GET" {
		d, e := s.editable(r.Context(), docID)
		if e != nil {
			fail(w, e)
			return
		}
		send(w, 200, map[string]any{"document": d})
		return
	}
	var input ContentMutation
	if e := decode(w, r, &input); e != nil {
		fail(w, e)
		return
	}
	d, e := s.mutateContent(r.Context(), docID, input)
	if e != nil {
		fail(w, e)
		return
	}
	s.audit(r.Context(), u.ID, "content."+input.Action, docID)
	send(w, 200, map[string]any{"document": d})
}
func (s *Server) mutateContent(ctx context.Context, docID string, in ContentMutation) (Document, error) {
	if in.Action != "save" && in.Action != "autosave" && in.Action != "publish" && in.Action != "unpublish" {
		return nil, bad("Invalid content action.")
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", docID); e != nil {
		return nil, e
	}
	var current Document
	var rev string
	e = tx.QueryRow(ctx, "SELECT document,revision FROM content WHERE id=$1 ORDER BY state LIMIT 1 FOR UPDATE", docID).Scan(&current, &rev)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return nil, e
	}
	if rev != in.Revision {
		return nil, problem{409, "Content changed elsewhere. Your edits are kept locally; reload and compare before saving."}
	}
	d := Document{}
	for k, v := range current {
		d[k] = v
	}
	for k, v := range in.Document {
		if !strings.HasPrefix(k, "_") || k == "_type" {
			d[k] = v
		}
	}
	for _, k := range in.Unset {
		if strings.HasPrefix(k, "_") {
			return nil, bad("Cannot remove system fields.")
		}
		delete(d, k)
	}
	typ := str(d, "_type")
	if current != nil && str(current, "_type") != typ {
		return nil, bad("Content type cannot change.")
	}
	if current == nil && (typ == "pageInfo" || typ == "siteSettings") {
		return nil, bad("Edit the existing profile or homepage.")
	}
	if in.Action == "unpublish" && (typ == "pageInfo" || typ == "siteSettings") {
		return nil, bad("Profile and homepage must stay published.")
	}
	var pub Document
	pubErr := tx.QueryRow(ctx, "SELECT document FROM content WHERE id=$1 AND state='published'", docID).Scan(&pub)
	if pubErr != nil && !errors.Is(pubErr, pgx.ErrNoRows) {
		return nil, pubErr
	}
	locked := str(current, "_publishedSlug")
	if locked == "" && pub != nil {
		locked = slug(pub)
	}
	if locked != "" && locked != slug(d) {
		return nil, problem{409, "Published URLs are permanent. Keep the original slug."}
	}
	if locked != "" {
		d["_publishedSlug"] = locked
	}
	if e = validateDocument(d, in.Action == "publish"); e != nil {
		return nil, e
	}
	if (typ == "projects" || typ == "note") && slug(d) != "" {
		if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", typ+":"+slug(d)); e != nil {
			return nil, e
		}
		var clash bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM content WHERE type=$1 AND id!=$2 AND document->'slug'->>'current'=$3)", typ, docID, slug(d)).Scan(&clash); e != nil {
			return nil, e
		}
		if clash {
			return nil, problem{409, "This URL slug is already in use."}
		}
	}
	d["_id"] = docID
	d["_rev"] = id()
	d["_updatedAt"] = time.Now().UTC().Format(time.RFC3339Nano)
	state := "draft"
	if in.Action == "publish" {
		state = "published"
		if slug(d) != "" {
			d["_publishedSlug"] = slug(d)
		}
		if e = s.checkReferences(ctx, tx, d); e != nil {
			return nil, e
		}
	}
	if e = writeDocument(ctx, tx, d, state); e != nil {
		return nil, e
	}
	if in.Action == "publish" {
		_, e = tx.Exec(ctx, "DELETE FROM content WHERE id=$1 AND state='draft'", docID)
	} else if in.Action == "unpublish" {
		_, e = tx.Exec(ctx, "DELETE FROM content WHERE id=$1 AND state='published'", docID)
	}
	if e != nil {
		return nil, e
	}
	if in.Action != "autosave" {
		raw, _ := json.Marshal(d)
		if _, e = tx.Exec(ctx, "INSERT INTO content_revisions(id,content_id,document,action) VALUES($1,$2,$3,$4)", d["_rev"], docID, raw, in.Action); e != nil {
			return nil, e
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	d["isDraft"] = state == "draft"
	d["hasPublished"] = state == "published" || pub != nil && in.Action != "unpublish"
	return d, nil
}
func writeDocument(ctx context.Context, tx pgx.Tx, d Document, state string) error {
	raw, e := json.Marshal(d)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO content(id,type,state,document,revision,updated_at) VALUES($1,$2,$3,$4,$5,now()) ON CONFLICT(id,state) DO UPDATE SET document=EXCLUDED.document,revision=EXCLUDED.revision,updated_at=now()`, d["_id"], d["_type"], state, raw, d["_rev"])
	if e != nil {
		return e
	}
	return syncRefs(ctx, tx, str(d, "_id"), state, d)
}
func (s *Server) checkReferences(ctx context.Context, tx pgx.Tx, d Document) error {
	found := map[string]bool{}
	refs(d, found)
	for ref := range found {
		if strings.HasPrefix(ref, "upload-") {
			var exists bool
			if e := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id=$1)", ref).Scan(&exists); e != nil {
				return e
			}
			if !exists {
				return bad("An image is missing from the media library.")
			}
		}
	}
	blocks, _ := d["body"].([]any)
	for _, v := range blocks {
		b, _ := v.(map[string]any)
		defs, _ := b["markDefs"].([]any)
		for _, v := range defs {
			m, _ := v.(map[string]any)
			href := str(m, "href")
			u, _ := url.Parse(href)
			if u == nil || u.IsAbs() || href == "" || strings.HasPrefix(href, "#") {
				continue
			}
			path := u.Path
			static := map[string]bool{"/": true, "/work": true, "/notes": true, "/about": true, "/now": true, "/photography": true, "/playground": true, "/guestbook": true, "/contact": true, "/privacy": true}
			if static[path] {
				continue
			}
			parts := strings.Split(strings.Trim(path, "/"), "/")
			if len(parts) != 2 || (parts[0] != "work" && parts[0] != "notes") {
				return bad("Broken internal link: " + path)
			}
			typ := "projects"
			if parts[0] == "notes" {
				typ = "note"
			}
			var exists bool
			if e := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM content WHERE state='published' AND type=$1 AND document->'slug'->>'current'=$2)", typ, parts[1]).Scan(&exists); e != nil {
				return e
			}
			if !exists && !(str(d, "_type") == typ && slug(d) == parts[1]) {
				return bad("Broken internal link: " + path)
			}
		}
	}
	return nil
}
func (s *Server) revisions(w http.ResponseWriter, r *http.Request) {
	if s.require(w, r, true) == nil {
		return
	}
	ctx := r.Context()
	docID := r.PathValue("id")
	if r.Method == "POST" {
		var in struct {
			RevisionID string `json:"revisionId"`
			Revision   string `json:"revision"`
		}
		if e := decode(w, r, &in); e != nil {
			fail(w, e)
			return
		}
		var d Document
		e := s.DB.QueryRow(ctx, "SELECT document FROM content_revisions WHERE id=$1 AND content_id=$2", in.RevisionID, docID).Scan(&d)
		if e != nil {
			fail(w, problem{404, "Revision not found."})
			return
		}
		current, e := s.editable(ctx, docID)
		if e != nil {
			fail(w, e)
			return
		}
		unset := []string{}
		for k := range current {
			if _, ok := d[k]; !ok && !strings.HasPrefix(k, "_") {
				unset = append(unset, k)
			}
		}
		out, e := s.mutateContent(ctx, docID, ContentMutation{Action: "save", Revision: in.Revision, Document: d, Unset: unset})
		if e != nil {
			fail(w, e)
			return
		}
		send(w, 200, map[string]any{"document": out})
		return
	}
	rows, e := s.DB.Query(ctx, "SELECT id,action,created_at,document FROM content_revisions WHERE content_id=$1 ORDER BY created_at DESC LIMIT 100", docID)
	if e != nil {
		fail(w, e)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var rev, action string
		var date time.Time
		var d Document
		if e = rows.Scan(&rev, &action, &date, &d); e != nil {
			fail(w, e)
			return
		}
		out = append(out, map[string]any{"id": rev, "action": action, "createdAt": date, "document": d})
	}
	send(w, 200, map[string]any{"revisions": out})
}
func (s *Server) schedules(w http.ResponseWriter, r *http.Request) {
	if s.require(w, r, true) == nil {
		return
	}
	ctx := r.Context()
	docID := r.PathValue("id")
	switch r.Method {
	case "POST":
		var in struct {
			RevisionID string    `json:"revisionId"`
			RunAt      time.Time `json:"runAt"`
		}
		if e := decode(w, r, &in); e != nil {
			fail(w, e)
			return
		}
		if in.RunAt.Before(time.Now().Add(time.Minute)) {
			fail(w, bad("Choose a time at least one minute in the future."))
			return
		}
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			fail(w, e)
			return
		}
		defer tx.Rollback(ctx)
		var d Document
		if e = tx.QueryRow(ctx, "SELECT document FROM content_revisions WHERE id=$1 AND content_id=$2", in.RevisionID, docID).Scan(&d); e != nil {
			fail(w, problem{404, "Save a revision before scheduling."})
			return
		}
		if e = validateDocument(d, true); e == nil {
			e = s.checkReferences(ctx, tx, d)
		}
		if e != nil {
			fail(w, e)
			return
		}
		_, e = tx.Exec(ctx, "INSERT INTO schedules(id,content_id,revision_id,run_at) VALUES($1,$2,$3,$4) ON CONFLICT(content_id) WHERE status='pending' DO UPDATE SET revision_id=EXCLUDED.revision_id,run_at=EXCLUDED.run_at", id(), docID, in.RevisionID, in.RunAt.UTC())
		if e == nil {
			e = tx.Commit(ctx)
		}
		if e != nil {
			fail(w, e)
			return
		}
	case "DELETE":
		if _, e := s.DB.Exec(ctx, "UPDATE schedules SET status='cancelled' WHERE content_id=$1 AND status='pending'", docID); e != nil {
			fail(w, e)
			return
		}
	}
	rows, e := s.DB.Query(ctx, "SELECT id,revision_id,run_at,status,coalesce(error,'') FROM schedules WHERE content_id=$1 ORDER BY created_at DESC LIMIT 20", docID)
	if e != nil {
		fail(w, e)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var sid, rev, status, message string
		var date time.Time
		if e = rows.Scan(&sid, &rev, &date, &status, &message); e != nil {
			fail(w, e)
			return
		}
		out = append(out, map[string]any{"id": sid, "revisionId": rev, "runAt": date, "status": status, "error": message})
	}
	send(w, 200, map[string]any{"schedules": out})
}
func (s *Server) organize(w http.ResponseWriter, r *http.Request) {
	if s.require(w, r, true) == nil {
		return
	}
	ctx := r.Context()
	docID := r.PathValue("id")
	var in struct {
		Action   string `json:"action"`
		Revision string `json:"revision"`
	}
	if e := decode(w, r, &in); e != nil {
		fail(w, e)
		return
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		fail(w, e)
		return
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", docID); e != nil {
		fail(w, e)
		return
	}
	var out Document
	switch in.Action {
	case "restore":
		var docs []struct {
			Document Document `json:"document"`
			State    string   `json:"state"`
		}
		if e = tx.QueryRow(ctx, "SELECT documents FROM content_trash WHERE id=$1 FOR UPDATE", docID).Scan(&docs); e != nil {
			fail(w, problem{404, "Trashed entry not found."})
			return
		}
		out = docs[0].Document
		for _, saved := range docs {
			if locked := str(saved.Document, "_publishedSlug"); locked != "" {
				out["_publishedSlug"] = locked
			}
		}
		out["_rev"] = id()
		if e = writeDocument(ctx, tx, out, "draft"); e == nil {
			_, e = tx.Exec(ctx, "DELETE FROM content_trash WHERE id=$1", docID)
		}
	case "trash", "duplicate":
		var d Document
		var rev string
		if e = tx.QueryRow(ctx, "SELECT document,revision FROM content WHERE id=$1 ORDER BY state LIMIT 1 FOR UPDATE", docID).Scan(&d, &rev); e != nil {
			fail(w, problem{404, "Content not found."})
			return
		}
		if rev != in.Revision {
			fail(w, problem{409, "Content changed. Reload before continuing."})
			return
		}
		if str(d, "_type") == "pageInfo" || str(d, "_type") == "siteSettings" {
			fail(w, bad("Profile and settings cannot be moved or duplicated."))
			return
		}
		if in.Action == "duplicate" {
			out = d
			out["_id"] = id()
			out["_rev"] = id()
			delete(out, "_publishedSlug")
			out["title"] = str(d, "title") + " (copy)"
			if slug(d) != "" {
				out["slug"] = map[string]any{"current": slug(d) + "-" + str(out, "_id")[:8]}
			}
			e = writeDocument(ctx, tx, out, "draft")
		} else {
			_, e = tx.Exec(ctx, "INSERT INTO content_trash(id,documents) SELECT $1,jsonb_agg(jsonb_build_object('document',document,'state',state) ORDER BY state) FROM content WHERE id=$1", docID)
			if e == nil {
				_, e = tx.Exec(ctx, "DELETE FROM content WHERE id=$1", docID)
			}
			if e == nil {
				_, e = tx.Exec(ctx, "UPDATE schedules SET status='cancelled' WHERE content_id=$1 AND status='pending'", docID)
			}
		}
	default:
		e = bad("Invalid organization action.")
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		fail(w, e)
		return
	}
	send(w, 200, map[string]any{"ok": true, "document": out})
}
func (s *Server) trash(w http.ResponseWriter, r *http.Request) {
	if s.require(w, r, true) == nil {
		return
	}
	rows, e := s.DB.Query(r.Context(), "SELECT id,documents->0->'document',deleted_at FROM content_trash ORDER BY deleted_at DESC LIMIT 100")
	if e != nil {
		fail(w, e)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var key string
		var d Document
		var date time.Time
		if e = rows.Scan(&key, &d, &date); e != nil {
			fail(w, e)
			return
		}
		out = append(out, map[string]any{"id": key, "document": d, "deletedAt": date})
	}
	send(w, 200, map[string]any{"entries": out})
}
