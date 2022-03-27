package main

import (
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/andrewpillar/config"
)

type serverConfig struct {
	Pidfile string

	Net struct {
		Listen string

		WriteTimeout time.Duration `config:"write_timeout"`
		ReadTimeout  time.Duration `config:"read_timeout"`

		TLS struct {
			Cert string
			Key  string
		}
	}

	Log map[string]string

	Store struct {
		Path string

		ScanInterval time.Duration `config:"scan_interval"`
	}

	Driver map[string]struct {
		Categories []string

		Groups []struct {
			Name    string
			Pattern string
		}
	}
}

var drivers = map[string]struct{}{
	"qemu": {},
}

func mkpidfile(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	pidfile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)

	if err != nil {
		return "", err
	}

	if _, err := io.WriteString(pidfile, strconv.FormatInt(int64(os.Getpid()), 10)); err != nil {
		return "", err
	}

	pidfile.Close()
	return pidfile.Name(), nil
}

var logmask = os.O_WRONLY | os.O_APPEND | os.O_CREATE

func logger(logtab map[string]string) (*Logger, error) {
	var (
		level LogLevel
		file  string
	)

	for label, val := range logtab {
		lvl, ok := LogLevels[label]

		if !ok {
			return nil, errors.New("unknown log level " + label)
		}

		if lvl <= level || level == 0 {
			level = lvl
			file = val
		}
	}

	log := NewLog(os.Stdout)
	log.SetLevel(level.String())

	if file != os.Stdout.Name() {
		f, err := os.OpenFile(file, logmask, 0640)

		if err != nil {
			return nil, err
		}
		log.SetWriter(f)
	}

	log.Info.Println("logging initialized, writing to", file, "at level", level.String())

	return log, nil
}

func DecodeConfig(f *os.File) (*Server, func(), error) {
	var cfg serverConfig

	dec := config.NewDecoder(f.Name())

	if err := dec.Decode(&cfg, f); err != nil {
		return nil, nil, err
	}

	pidfile, err := mkpidfile(cfg.Pidfile)

	if err != nil {
		return nil, nil, err
	}

	srv := &http.Server{
		Addr:         cfg.Net.Listen,
		WriteTimeout: cfg.Net.WriteTimeout,
		ReadTimeout:  cfg.Net.ReadTimeout,
	}

	if cfg.Net.TLS.Cert != "" && cfg.Net.TLS.Key != "" {
		cert, err := tls.LoadX509KeyPair(cfg.Net.TLS.Cert, cfg.Net.TLS.Key)

		if err != nil {
			return nil, nil, err
		}

		srv.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{
				cert,
			},
		}
	}

	log, err := logger(cfg.Log)

	if err != nil {
		return nil, nil, err
	}

	sc := &Scanner{
		dir: cfg.Store.Path,
		errh: func(err error) {
			log.Error.Println("failed to scan images", err)
		},
		drivers: make(map[string]driver),
	}

	for name, cfg := range cfg.Driver {
		if _, ok := drivers[name]; !ok {
			return nil, nil, errors.New("unknown driver: " + name)
		}

		categories := make(map[string]struct{})

		for _, name := range cfg.Categories {
			categories[name] = struct{}{}
		}

		groups := make([]driverGroup, 0, len(cfg.Groups))

		for _, group := range cfg.Groups {
			re, err := regexp.Compile(group.Pattern)

			if err != nil {
				return nil, nil, err
			}

			groups = append(groups, driverGroup{
				name: group.Name,
				re:   re,
			})
		}

		sc.drivers[name] = driver{
			name:       name,
			categories: categories,
			groups:     groups,
		}
	}

	db, err := InitDB()

	if err != nil {
		return nil, nil, err
	}

	close := func() {
		db.Close()
		os.RemoveAll(pidfile)
	}

	return &Server{
		Server:       srv,
		DB:           db,
		Log:          log,
		Scanner:      sc,
		ScanInterval: cfg.Store.ScanInterval,
	}, close, nil
}
