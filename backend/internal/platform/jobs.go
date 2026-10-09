package platform

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"
)

func (s *Server) RunJobs(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		if e := s.publishDue(ctx); e != nil {
			slog.Error("scheduled publishing", "error", e)
		}
		if e := s.deliverMail(ctx); e != nil {
			slog.Error("mail worker", "error", e)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (s *Server) publishDue(ctx context.Context) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var key, docID string
	var d Document
	e = tx.QueryRow(ctx, `SELECT s.id,s.content_id,r.document FROM schedules s JOIN content_revisions r ON r.id=s.revision_id WHERE s.status='pending' AND s.run_at<=now() ORDER BY s.run_at LIMIT 1 FOR UPDATE OF s SKIP LOCKED`).Scan(&key, &docID, &d)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", docID); e != nil {
		return e
	}
	e = validateDocument(d, true)
	if e == nil {
		e = s.checkReferences(ctx, tx, d)
	}
	var currentSlug string
	if e == nil {
		e = tx.QueryRow(ctx, "SELECT coalesce(document->>'_publishedSlug',CASE WHEN state='published' THEN document->'slug'->>'current' END,'') FROM content WHERE id=$1 ORDER BY state LIMIT 1", docID).Scan(&currentSlug)
	}
	if e == nil && currentSlug != "" && slug(d) != currentSlug {
		e = bad("The saved revision would change a published URL.")
	}
	if e != nil {
		_, e = tx.Exec(ctx, "UPDATE schedules SET status='failed',error=$2 WHERE id=$1", key, e.Error())
		if e != nil {
			return e
		}
		return tx.Commit(ctx)
	}
	if slug(d) != "" {
		d["_publishedSlug"] = slug(d)
	}
	savedRev := str(d, "_rev")
	d["_rev"] = id()
	if e = writeDocument(ctx, tx, d, "published"); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "DELETE FROM content WHERE id=$1 AND state='draft' AND revision=$2", docID, savedRev); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE schedules SET status='published',error=NULL WHERE id=$1", key); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Server) deliverMail(ctx context.Context) error {
	if s.C.SMTPUser == "" {
		return nil
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var key, to, subject, body string
	var attempts int
	e = tx.QueryRow(ctx, "SELECT id,recipient,subject,body,attempts FROM email_jobs WHERE status='pending' AND run_at<=now() ORDER BY run_at LIMIT 1 FOR UPDATE SKIP LOCKED").Scan(&key, &to, &subject, &body, &attempts)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	sendErr := s.smtpSend(key, to, subject, body)
	if sendErr == nil {
		_, e = tx.Exec(ctx, "UPDATE email_jobs SET status='sent',body='',attempts=attempts+1,last_error=NULL WHERE id=$1", key)
	} else {
		status := "pending"
		if attempts >= 7 {
			status = "failed"
		}
		_, e = tx.Exec(ctx, "UPDATE email_jobs SET status=$2,attempts=attempts+1,last_error=$3,run_at=$4 WHERE id=$1", key, status, sendErr.Error(), time.Now().Add(time.Duration(1<<min(attempts, 6))*time.Minute))
		slog.Error("mail delivery failed", "job", key, "error", sendErr)
	}
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Server) smtpSend(key, to, subject, body string) error {
	if !validEmail(to) || !validEmail(s.C.MailFrom) || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid mail headers")
	}
	host, _, e := net.SplitHostPort(s.C.SMTPAddr)
	if e != nil {
		return e
	}
	conn, e := net.DialTimeout("tcp", s.C.SMTPAddr, 10*time.Second)
	if e != nil {
		return e
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	c, e := smtp.NewClient(conn, host)
	if e != nil {
		return e
	}
	defer c.Close()
	if ok, _ := c.Extension("STARTTLS"); !ok {
		return fmt.Errorf("SMTP requires STARTTLS")
	}
	if e = c.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); e != nil {
		return e
	}
	if e = c.Auth(smtp.PlainAuth("", s.C.SMTPUser, s.C.SMTPPassword, host)); e != nil {
		return e
	}
	if e = c.Mail(s.C.MailFrom); e != nil {
		return e
	}
	if e = c.Rcpt(to); e != nil {
		return e
	}
	w, e := c.Data()
	if e != nil {
		return e
	}
	_, e = fmt.Fprintf(w, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMessage-ID: <%s@%s>\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", s.C.MailFrom, to, subject, key, strings.Split(s.C.MailFrom, "@")[1], strings.ReplaceAll(body, "\n", "\r\n"))
	if e != nil {
		return e
	}
	if e = w.Close(); e != nil {
		return e
	}
	return c.Quit()
}
