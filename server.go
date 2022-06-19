package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/query"
)

type Server struct {
	*http.Server

	DB DB

	Log *Logger

	Scanner *Scanner

	ScanInterval time.Duration
}

func (s *Server) scan(ctx context.Context, imgs chan<- []*Image) {
	t := time.NewTicker(s.ScanInterval)

	go func() {
		for {
			select {
			case <-ctx.Done():
				s.Log.Info.Println("stopping image scan")
				close(imgs)
				t.Stop()
				return
			case <-t.C:
				imgs <- s.Scanner.Scan()
			}
		}
	}()
}

func (s *Server) InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	s.Log.Error.Println(err)
	w.WriteHeader(http.StatusInternalServerError)
}

func (s *Server) NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func (s *Server) Handle(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")

	parts := strings.Split(r.URL.Path, "/")

	driver := parts[1]

	var category string

	if len(parts) > 2 {
		category = parts[2]

		if len(parts) > 3 {
			img, ok, err := s.DB.Image(driver, category, strings.Join(parts[3:], "/"))

			if err != nil {
				s.InternalServerError(w, r, err)
				return
			}

			if !ok {
				s.NotFound(w, r)
				return
			}

			if strings.HasPrefix(accept, "application/json") {
				json.NewEncoder(w).Encode(img)
				return
			}

			rsc, err := img.Data()

			if err != nil {
				s.InternalServerError(w, r, err)
				return
			}

			defer rsc.Close()

			w.Header().Set("Content-Type", "application/x-qemu-disk")
			http.ServeContent(w, r, img.Name, img.ModTime, rsc)
			return
		}
	}

	imgs, err := s.DB.Images(
		WhereDriver(driver),
		WhereCategory(category),
		WhereGroup(r.URL.Query().Get("group")),
		query.OrderAsc("driver", "category", "group_name", "path"),
	)

	if err != nil {
		s.InternalServerError(w, r, err)
		return
	}

	if strings.HasPrefix(accept, "application/json") {
		json.NewEncoder(w).Encode(imgs)
		return
	}

	var tree Tree

	for _, img := range imgs {
		tree.Put(img)
	}

	p := &Index{
		Tree:        &tree,
		DjinnServer: DJINN_SERVER,
	}

	page := p.Render()

	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(page)), 10))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, page)
}

func (s *Server) Serve(ctx context.Context) error {
	s.Handler = http.HandlerFunc(s.Handle)

	sync := make(chan []*Image)

	if err := s.DB.Load(s.Scanner.Scan()); err != nil {
		s.Log.Error.Println("failed to load images", err)
	}

	go func() {
		for imgs := range sync {
			s.Log.Debug.Println("syncing", len(imgs), "image(s)")

			n, err := s.DB.Sync(imgs)

			if err != nil {
				s.Log.Error.Println("failed to sync images", err)
				continue
			}
			s.Log.Debug.Println("synced", n, "image(s)")
		}
	}()

	s.scan(ctx, sync)

	ln, err := net.Listen("tcp", s.Addr)

	if err != nil {
		return err
	}

	if s.TLSConfig != nil {
		ln = tls.NewListener(ln, s.TLSConfig)
	}

	if err := s.Server.Serve(ln); err != nil {
		return err
	}
	return nil
}
