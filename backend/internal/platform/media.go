package platform

import (
	"bytes"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func mustReadSeeker(r io.Reader) io.ReadSeeker { b, _ := io.ReadAll(r); return bytes.NewReader(b) }
func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	if s.require(w, r, true) == nil {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 11<<20)
	if e := r.ParseMultipartForm(11 << 20); e != nil {
		fail(w, bad("Choose a JPEG, PNG, or WebP image up to 10 MB."))
		return
	}
	defer r.MultipartForm.RemoveAll()
	file, header, e := r.FormFile("file")
	if e != nil {
		fail(w, bad("Choose an image."))
		return
	}
	defer file.Close()
	if header.Size > 10<<20 {
		fail(w, bad("Image exceeds 10 MB."))
		return
	}
	raw, e := io.ReadAll(io.LimitReader(file, (10<<20)+1))
	if e != nil {
		fail(w, e)
		return
	}
	cfg, _, e := image.DecodeConfig(bytes.NewReader(raw))
	if e != nil || cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 40_000_000 {
		fail(w, bad("Unsupported image or image exceeds 40 megapixels."))
		return
	}
	img, _, e := image.Decode(bytes.NewReader(raw))
	if e != nil {
		fail(w, bad("The image could not be decoded."))
		return
	}
	width, height := cfg.Width, cfg.Height
	if width > 2400 || height > 2400 {
		ratio := 2400 / float64(max(width, height))
		width = int(float64(width) * ratio)
		height = int(float64(height) * ratio)
		out := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.CatmullRom.Scale(out, out.Bounds(), img, img.Bounds(), draw.Over, nil)
		img = out
	}
	key := "upload-" + id()
	filename := key + ".jpg"
	path := filepath.Join(s.C.DataDir, "media", filename)
	output, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		fail(w, e)
		return
	}
	e = jpeg.Encode(output, img, &jpeg.Options{Quality: 88})
	closeErr := output.Close()
	if e == nil {
		e = closeErr
	}
	if e != nil {
		_ = os.Remove(path)
		fail(w, e)
		return
	}
	alt := strings.TrimSpace(r.FormValue("alt"))
	if len(alt) > 1000 {
		_ = os.Remove(path)
		fail(w, bad("Alternative text is too long."))
		return
	}
	if _, e = s.DB.Exec(r.Context(), "INSERT INTO assets(id,filename,alt,width,height) VALUES($1,$2,$3,$4,$5)", key, filename, alt, width, height); e != nil {
		_ = os.Remove(path)
		fail(w, e)
		return
	}
	send(w, 201, map[string]any{"image": map[string]any{"_type": "image", "asset": map[string]any{"_type": "reference", "_ref": key}, "alt": alt}, "id": key})
}
func (s *Server) mediaIndex(w http.ResponseWriter, r *http.Request) {
	if s.require(w, r, true) == nil {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	rows, e := s.DB.Query(r.Context(), `SELECT a.id,a.alt,a.caption,a.width,a.height,a.created_at,coalesce((SELECT jsonb_agg(jsonb_build_object('id',r.content_id,'state',r.state,'title',coalesce(c.document->>'title',c.document->>'name',r.content_id))) FROM asset_references r JOIN content c ON c.id=r.content_id AND c.state=r.state WHERE r.asset_id=a.id),'[]'),count(*) OVER() FROM assets a WHERE alt ILIKE '%'||$1||'%' OR caption ILIKE '%'||$1||'%' ORDER BY created_at DESC LIMIT 30 OFFSET $2`, r.URL.Query().Get("q"), (page-1)*30)
	if e != nil {
		fail(w, e)
		return
	}
	defer rows.Close()
	out := []any{}
	total := 0
	for rows.Next() {
		var key, alt, caption string
		var width, height int
		var date time.Time
		var used []any
		if e = rows.Scan(&key, &alt, &caption, &width, &height, &date, &used, &total); e != nil {
			fail(w, e)
			return
		}
		out = append(out, map[string]any{"id": key, "alt": alt, "caption": caption, "width": width, "height": height, "createdAt": date, "usedBy": used})
	}
	send(w, 200, map[string]any{"assets": out, "total": total, "page": page})
}
func (s *Server) mediaEdit(w http.ResponseWriter, r *http.Request) {
	if s.require(w, r, true) == nil {
		return
	}
	var in struct {
		Alt     string `json:"alt"`
		Caption string `json:"caption"`
	}
	if e := decode(w, r, &in); e != nil {
		fail(w, e)
		return
	}
	if len(in.Alt) > 1000 || len(in.Caption) > 1000 {
		fail(w, bad("Image text is too long."))
		return
	}
	if _, e := s.DB.Exec(r.Context(), "UPDATE assets SET alt=$2,caption=$3 WHERE id=$1", r.PathValue("id"), in.Alt, in.Caption); e != nil {
		fail(w, e)
		return
	}
	send(w, 200, map[string]any{"ok": true})
}
func (s *Server) mediaFile(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSuffix(strings.TrimSuffix(r.PathValue("filename"), ".webp"), ".jpg")
	if !validAsset.MatchString(key) {
		http.NotFound(w, r)
		return
	}
	var filename string
	if e := s.DB.QueryRow(r.Context(), "SELECT filename FROM assets WHERE id=$1", key).Scan(&filename); e != nil {
		http.NotFound(w, r)
		return
	}
	var public bool
	if e := s.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM asset_references WHERE asset_id=$1 AND state='published')", key).Scan(&public); e != nil {
		fail(w, e)
		return
	}
	if !public {
		u := s.user(r, "api")
		if !u.authenticated() || !u.Owner {
			http.NotFound(w, r)
			return
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Vary", "Cookie")
	w.Header().Set("Content-Security-Policy", "default-src 'none'")
	http.ServeFile(w, r, filepath.Join(s.C.DataDir, "media", filepath.Base(filename)))
}
