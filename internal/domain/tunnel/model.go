package tunnel

import (
	"context"
	"errors"
	"net/netip"
	"regexp"
	"strings"
)

var (
	ErrInvalid     = errors.New("invalid tunnel configuration")
	ErrNotFound    = errors.New("tunnel not found")
	ErrConflict    = errors.New("tunnel operation conflicts with current state")
	ErrCredentials = errors.New("tunnel credentials unavailable")
	ErrUnavailable = errors.New("cloudflared unavailable or unsupported")
	ErrProcess     = errors.New("cloudflared process failed")
)

const Collection = "fh_cloudflare_tunnels"

var ValidID = regexp.MustCompile(`^[a-z0-9]{15}$`)

type ExtraArg struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type Config struct {
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	AutoStart          bool       `json:"autoStart"`
	AutoRestart        bool       `json:"autoRestart"`
	MaxRestartAttempts int        `json:"maxRestartAttempts"`
	Protocol           string     `json:"protocol"`
	Region             string     `json:"region"`
	PostQuantum        bool       `json:"postQuantum"`
	ConnectionRetries  int        `json:"connectionRetries"`
	GracePeriodSeconds int        `json:"gracePeriodSeconds"`
	EdgeIPVersion      string     `json:"edgeIPVersion"`
	EdgeBindAddress    string     `json:"edgeBindAddress"`
	MetricsPort        int        `json:"metricsPort"` // 0: OS assigned; always loopback
	ExtraArgs          []ExtraArg `json:"extraArgs"`
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 120 || len(c.Description) > 2000 {
		return ErrInvalid
	}
	if c.Protocol != "auto" && c.Protocol != "quic" && c.Protocol != "http2" {
		return ErrInvalid
	}
	if c.Region != "" && c.Region != "us" {
		return ErrInvalid
	}
	if c.PostQuantum && c.Protocol != "quic" {
		return ErrInvalid
	}
	if c.EdgeIPVersion != "auto" && c.EdgeIPVersion != "4" && c.EdgeIPVersion != "6" {
		return ErrInvalid
	}
	if c.EdgeBindAddress != "" {
		ip, e := netip.ParseAddr(c.EdgeBindAddress)
		if e != nil || ip.Zone() != "" || (c.EdgeIPVersion == "4" && !ip.Is4()) || (c.EdgeIPVersion == "6" && !ip.Is6()) {
			return ErrInvalid
		}
	}
	if c.MaxRestartAttempts < 0 || c.MaxRestartAttempts > 100 || c.ConnectionRetries < 0 || c.ConnectionRetries > 20 || c.GracePeriodSeconds < 1 || c.GracePeriodSeconds > 120 || c.MetricsPort < 0 || c.MetricsPort > 65535 {
		return ErrInvalid
	}
	if len(c.ExtraArgs) > 8 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, a := range c.ExtraArgs { // Only verified, non-secret, non-routing flags.
		if seen[a.Name] {
			return ErrInvalid
		}
		seen[a.Name] = true
		if (a.Name != "loglevel" || a.Value != "info") && (a.Name != "ha-connections" || (a.Value != "1" && a.Value != "2" && a.Value != "3" && a.Value != "4")) {
			return ErrInvalid
		}
	}
	return nil
}

type Record struct {
	ID string `json:"id"`
	Config
	Revision   int    `json:"revision"`
	HasToken   bool   `json:"hasToken"`
	Ciphertext string `json:"-"`
}
type Write struct {
	Config
	Revision int     `json:"revision"`
	Token    *string `json:"token,omitempty"`
}
type Repository interface {
	List() ([]Record, error)
	Get(string) (Record, error)
	Create(Record) (Record, error)
	Update(Record, int) (Record, error)
	Delete(string, int) error
}
type Vault interface {
	Encrypt(string, string) (string, error)
	Decrypt(string, string) (string, error)
}
type Capabilities struct {
	Available        bool     `json:"available"`
	Version          string   `json:"version"`
	Build            string   `json:"build"`
	TokenFile        bool     `json:"tokenFile"`
	TLSLocalOverride bool     `json:"tlsLocalOverride"`
	Flags            []string `json:"flags"`
}
type Log struct {
	Cursor  uint64 `json:"cursor"`
	Message string `json:"message"`
}
type Status struct {
	Phase          string `json:"phase"`
	Generation     uint64 `json:"generation"`
	Protocol       string `json:"protocol"`
	Ready          bool   `json:"ready"`
	Connections    int    `json:"connections"`
	RestartCount   int    `json:"restartCount"`
	NextRetry      string `json:"nextRetry,omitempty"`
	LastError      string `json:"lastError,omitempty"`
	MetricsAddress string `json:"metricsAddress,omitempty"`
	Revision       int    `json:"revision"`
}
type Runner interface {
	Capabilities() Capabilities
	Start(Record, string) error
	Stop(string) error
	Restart(Record, string) error
	Status(string) Status
	Logs(string, uint64) []Log
	LogEpoch() string
	Forget(string)
	Shutdown(context.Context) error
}
