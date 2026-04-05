package geo

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	downloadUrlTempl = "https://www.ip2location.com/download/?token=%s&file=DB9LITEMMDB"
)

// Locator handles IP-to-Location lookups via Database or API fallbacks.
type Locator struct {
	logger     zerolog.Logger
	httpClient *http.Client
	dbDir      string
	token      string

	db    *geoip2.Reader
	dbMtx sync.RWMutex
}

type Option func(*Locator)

func WithToken(token string) Option {
	return func(l *Locator) {
		l.token = token
	}
}

func WithStorageDir(path string) Option {
	return func(l *Locator) {
		l.dbDir = path
	}
}

// NewLocator creates a new Locator. It attempts to load a local DB immediately.
// If the DB is missing or old, and a token is provided, it triggers a background update.
func NewLocator(logger zerolog.Logger, opts ...Option) (*Locator, error) {
	l := &Locator{
		logger: logger,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		dbDir: defaultDBDir,
	}

	for _, opt := range opts {
		opt(l)
	}

	if err := l.loadLocalDatabase(); err != nil {
		l.logger.Debug().Err(err).Msg("Local database not available or too old")
	}

	if l.token != "" {
		go l.updateDatabaseBackground()
	}

	return l, nil
}

// GetLocation returns the latitude and longitude for a given network address.
func (l *Locator) GetLocation(addr net.Addr) (types.Location, error) {
	l.dbMtx.RLock()
	db := l.db
	l.dbMtx.RUnlock()

	if db != nil {
		loc, err := l.lookupDatabase(db, addr)
		if err == nil {
			return loc, nil
		}
		l.logger.Warn().Err(err).Msg("Database lookup failed, falling back to API")
	}

	return l.lookupAPI(addr)
}

// Close cleans up resources.
func (l *Locator) Close() error {
	l.dbMtx.Lock()
	defer l.dbMtx.Unlock()
	if l.db != nil {
		return l.db.Close()
	}
	return nil
}

//
// Database Logic
//

func (l *Locator) lookupDatabase(db *geoip2.Reader, addr net.Addr) (types.Location, error) {
	ipStr, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		ipStr = addr.String()
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return types.Location{}, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	record, err := db.City(ip)
	if err != nil {
		return types.Location{}, err
	}

	if record.Location.Latitude == 0 && record.Location.Longitude == 0 {
		return types.Location{}, fmt.Errorf("ip not found in database")
	}

	return types.Location{
		Latitude:  record.Location.Latitude,
		Longitude: record.Location.Longitude,
	}, nil
}

func (l *Locator) loadLocalDatabase() error {
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

	if newestFile == "" {
		return fmt.Errorf("no database files found")
	}

	if time.Since(newestTime) > dbMaxAge {
		return fmt.Errorf("database file is expired")
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
		l.db.Close()
	}
	l.db = db
	return nil
}

func (l *Locator) updateDatabaseBackground() {
	l.logger.Info().Msg("Starting background database update")

	zipBytes, err := l.downloadDatabase()
	if err != nil {
		l.logger.Error().Err(err).Msg("Failed to download database")
		return
	}

	mmdbBytes, err := extractMMDB(zipBytes)
	if err != nil {
		l.logger.Error().Err(err).Msg("Failed to extract database from zip")
		return
	}

	if err := os.MkdirAll(l.dbDir, 0755); err != nil {
		l.logger.Error().Err(err).Msg("Failed to create db directory")
		return
	}

	filename := fmt.Sprintf("%s_bd9lite.mmdb", time.Now().Format("2006_01_02"))
	fullPath := filepath.Join(l.dbDir, filename)
	if err := os.WriteFile(fullPath, mmdbBytes, 0644); err != nil {
		l.logger.Error().Err(err).Msg("Failed to save database file")
		return
	}

	if err := l.mountDatabase(fullPath); err != nil {
		l.logger.Error().Err(err).Msg("Failed to mount new database")
		return
	}

	l.logger.Info().Msg("Database updated successfully")
}

func (l *Locator) downloadDatabase() ([]byte, error) {
	url := fmt.Sprintf(downloadUrlTempl, l.token)
	resp, err := l.httpClient.Get(url)
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
	return nil, fmt.Errorf("no .mmdb found in archive")
}

//
// API Logic
//

type provider struct {
	url       string
	parseFunc func([]byte) (types.Location, error)
}

func (l *Locator) lookupAPI(addr net.Addr) (types.Location, error) {
	ipStr, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		ipStr = addr.String()
	}
	parsedIP := net.ParseIP(ipStr)

	// Determine providers based on IP type
	// If Private/Loopback, we want the public IP of THIS machine (no IP arg in URL)
	// If Public, we want the location of THAT IP (IP arg in URL)
	isSelfLookup := parsedIP == nil || parsedIP.IsPrivate() || parsedIP.IsLoopback() || parsedIP.IsUnspecified()
	providers := l.getProviders(isSelfLookup, ipStr)

	var lastErr error
	for _, p := range providers {
		loc, err := l.fetchFromProvider(p)
		if err == nil {
			return loc, nil
		}
		lastErr = err
		l.logger.Debug().Err(err).Str("url", p.url).Msg("API provider failed, trying next")
	}

	return types.Location{}, fmt.Errorf("all API providers failed, last error: %v", lastErr)
}

func (l *Locator) fetchFromProvider(p provider) (types.Location, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", p.url, nil)
	if err != nil {
		return types.Location{}, err
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return types.Location{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return types.Location{}, fmt.Errorf("status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.Location{}, err
	}

	return p.parseFunc(body)
}

func (l *Locator) getProviders(isSelfLookup bool, targetIP string) []provider {
	u := func(base, path string) string {
		if isSelfLookup {
			return base
		}
		return fmt.Sprintf(path, targetIP)
	}

	return []provider{
		{
			url: u("http://ip-api.com/json/", "http://ip-api.com/json/%s"),
			parseFunc: func(b []byte) (types.Location, error) {
				var r struct {
					Lat float64 `json:"lat"`
					Lon float64 `json:"lon"`
				}
				if err := json.Unmarshal(b, &r); err != nil {
					return types.Location{}, err
				}
				if r.Lat == 0 && r.Lon == 0 {
					return types.Location{}, fmt.Errorf("empty coords")
				}
				return types.Location{Latitude: r.Lat, Longitude: r.Lon}, nil
			},
		},
		{
			url: u("http://ipwho.is/", "http://ipwho.is/%s"),
			parseFunc: func(b []byte) (types.Location, error) {
				var r struct {
					Lat float64 `json:"latitude"`
					Lon float64 `json:"longitude"`
				}
				if err := json.Unmarshal(b, &r); err != nil {
					return types.Location{}, err
				}
				return types.Location{Latitude: r.Lat, Longitude: r.Lon}, nil
			},
		},
		{
			url: u("https://ipinfo.io/json", "https://ipinfo.io/%s/json"),
			parseFunc: func(b []byte) (types.Location, error) {
				var r struct {
					Loc string `json:"loc"`
				}
				if err := json.Unmarshal(b, &r); err != nil {
					return types.Location{}, err
				}
				parts := strings.Split(r.Loc, ",")
				if len(parts) != 2 {
					return types.Location{}, fmt.Errorf("invalid loc format")
				}
				lat, _ := strconv.ParseFloat(parts[0], 64)
				lon, _ := strconv.ParseFloat(parts[1], 64)
				return types.Location{Latitude: lat, Longitude: lon}, nil
			},
		},
	}
}
