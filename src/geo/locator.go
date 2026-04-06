package geo

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/oschwald/geoip2-golang"
	"github.com/rs/zerolog"
)

const (
	defaultTimeout   = 5 * time.Second
	defaultDBDir     = "./ip2LocatorFiles"
	dbMaxAge         = 30 * 24 * time.Hour // 30 days
	downloadURLTempl = "https://www.ip2location.com/download/?token=%s&file=DB9LITEMMDB"
)

type Config struct {
	Token string
	Dir   string
}

// Locator handles IP-to-Location lookups via the local Database.
type Locator struct {
	logger zerolog.Logger
	dbDir  string
	token  string

	db    *geoip2.Reader
	dbMtx sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

// NewLocator creates a new Locator. It attempts to load a local DB immediately.
// If the DB is missing or old, and a token is provided, it triggers a background update.
func NewLocator(upstreamCtx context.Context, logger zerolog.Logger, config Config) (*Locator, error) {
	ctx, cancel := context.WithCancel(upstreamCtx)
	l := &Locator{
		logger: logger,
		dbDir:  defaultDBDir,

		ctx:    ctx,
		cancel: cancel,
	}
	l.dbDir = config.Dir
	l.token = config.Token

	return l, l.loadDatabase()
}

// GetLocation returns the latitude and longitude for a given network address.
func (l *Locator) GetLocation(addr net.Addr) (types.Location, error) {
	ipStr, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		ipStr = addr.String()
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return types.Location{}, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	l.dbMtx.RLock()
	defer l.dbMtx.RUnlock()

	record, err := l.db.City(ip)
	if err != nil {
		return types.Location{}, err
	}
	if record.Location.Latitude == 0 && record.Location.Longitude == 0 {
		return types.Location{}, errors.New("ip not found in database")
	}

	return types.Location{
		Latitude:  record.Location.Latitude,
		Longitude: record.Location.Longitude,
	}, nil
}

// Close cleans up resources.
func (l *Locator) Close() error {
	l.cancel()

	l.dbMtx.Lock()
	defer l.dbMtx.Unlock()

	if l.db != nil {
		return l.db.Close()
	}
	return nil
}

// loadDatabase finds the newest .mmdb file in the dbDir, checks its age, and loads it if valid.
func (l *Locator) loadDatabase() error {
	entries, err := os.ReadDir(l.dbDir)
	if err != nil {
		return err
	}

	var newestFile string
	var newestTime time.Time

	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".mmdb") {
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(newestTime) {
				newestTime = info.ModTime()
				newestFile = filepath.Join(l.dbDir, e.Name())
			}
		}
	}

	if newestFile == "" || time.Since(newestTime) > dbMaxAge {
		return l.updateDatabase()
	}

	return l.mountDatabase(newestFile)
}

func (l *Locator) mountDatabase(path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	db, err := geoip2.FromBytes(bytes)
	if err != nil {
		return err
	}

	l.dbMtx.Lock()
	defer l.dbMtx.Unlock()

	if l.db != nil {
		err = l.db.Close()
		if err != nil {
			return err
		}
	}
	l.db = db

	return nil
}

func (l *Locator) updateDatabase() error {
	l.logger.Info().Msg("Database is too old, updating....")

	zipBytes, err := l.downloadDatabase()
	if err != nil {
		return err
	}

	mmdbBytes, err := extractMMDB(zipBytes)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(l.dbDir, 0o750); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s_bd9lite.mmdb", time.Now().Format("2006_01_02"))
	fullPath := filepath.Join(l.dbDir, filename)
	if err := os.WriteFile(fullPath, mmdbBytes, 0o600); err != nil {
		return err
	}

	if err := l.mountDatabase(fullPath); err != nil {
		return err
	}

	l.logger.Info().Msg("Database updated successfully")
	return nil
}

func (l *Locator) downloadDatabase() ([]byte, error) {
	url := fmt.Sprintf(downloadURLTempl, l.token)

	httpClient := &http.Client{Timeout: defaultTimeout}

	req, err := http.NewRequestWithContext(l.ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func extractMMDB(zipData []byte) ([]byte, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, err
	}

	for _, file := range zipReader.File {
		if strings.HasSuffix(strings.ToLower(file.Name), ".mmdb") {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, errors.New("no .mmdb found in archive")
}
