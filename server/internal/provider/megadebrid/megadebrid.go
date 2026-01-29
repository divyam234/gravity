package megadebrid

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rclone/rclone/lib/rest"
	"gravity/internal/client"
	"gravity/internal/model"
	"gravity/internal/provider"
)

const baseURL = "https://www.mega-debrid.eu/api.php"

type MegaDebridProvider struct {
	username   string
	password   string
	proxyURL   string
	token      string
	tokenMutex sync.Mutex
	client     *client.Client
	regexes    []*regexp.Regexp
}

func New() *MegaDebridProvider {
	return &MegaDebridProvider{}
}

func (p *MegaDebridProvider) Name() string        { return "megadebrid" }
func (p *MegaDebridProvider) DisplayName() string { return "MegaDebrid.eu" }
func (p *MegaDebridProvider) Type() model.ProviderType {
	return model.ProviderTypeDebrid
}

func (p *MegaDebridProvider) ConfigSchema() []provider.ConfigField {
	return []provider.ConfigField{
		{
			Key:         "username",
			Label:       "Username",
			Type:        "text",
			Required:    false,
			Description: "Your MegaDebrid.eu username (optional if Token is provided)",
		},
		{
			Key:         "password",
			Label:       "Password",
			Type:        "password",
			Required:    false,
			Description: "Your MegaDebrid.eu password (optional if Token is provided)",
		},
		{
			Key:         "token",
			Label:       "Token",
			Type:        "password",
			Required:    false,
			Description: "API Token (optional, if you have a permanent one)",
		},
		{
			Key:         "proxy_url",
			Label:       "Proxy URL",
			Type:        "text",
			Required:    false,
			Description: "Optional proxy (http://user:pass@host:port)",
		},
	}
}

func (p *MegaDebridProvider) Configure(ctx context.Context, config map[string]string) error {
	p.username = config["username"]
	p.password = config["password"]
	p.proxyURL = config["proxy_url"]
	if t, ok := config["token"]; ok && t != "" {
		p.token = t
	} else {
		p.token = ""
	}

	opts := []client.Option{
		client.WithTimeout(15 * time.Second),
	}

	if p.proxyURL != "" {
		opts = append(opts, client.WithProxy(p.proxyURL))
	}

	p.client = client.New(ctx, baseURL, opts...)

	// Fetch supported hosts/regexes synchronously with timeout
	tCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := p.fetchRegexes(tCtx); err != nil {
		fmt.Printf("MegaDebrid: failed to fetch supported hosts: %v\n", err)
	}

	return nil
}

func (p *MegaDebridProvider) IsConfigured() bool {
	return (p.username != "" && p.password != "") || p.token != ""
}

func (p *MegaDebridProvider) getToken(ctx context.Context) (string, error) {
	p.tokenMutex.Lock()
	defer p.tokenMutex.Unlock()

	if p.token != "" {
		return p.token, nil
	}

	form := url.Values{}
	form.Set("login", p.username)
	form.Set("password", p.password)

	opts := rest.Opts{
		Method:      "POST",
		Path:        "",
		Parameters:  url.Values{"action": {"connectUser"}},
		ContentType: "application/x-www-form-urlencoded",
		Body:        strings.NewReader(form.Encode()),
	}

	var result struct {
		ResponseCode string `json:"response_code"`
		ResponseText string `json:"response_text"`
		Token        string `json:"token"`
	}

	_, err := p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return "", err
	}

	if result.ResponseCode != "ok" {
		return "", fmt.Errorf("megadebrid error: %s", result.ResponseText)
	}

	p.token = result.Token
	return p.token, nil
}

func (p *MegaDebridProvider) Supports(rawURL string) bool {
	if strings.HasPrefix(rawURL, "magnet:") {
		return true
	}

	if len(p.regexes) > 0 {
		for _, re := range p.regexes {
			if re.MatchString(rawURL) {
				return true
			}
		}
		return false
	}

	return strings.Contains(rawURL, "mega-debrid.eu")
}

func (p *MegaDebridProvider) fetchRegexes(ctx context.Context) error {
	token, err := p.getToken(ctx)
	if err != nil {
		return err
	}

	opts := rest.Opts{
		Method: "GET",
		Path:   "",
		Parameters: url.Values{
			"action": {"getHostersList"},
			"token":  {token},
		},
	}

	var result struct {
		ResponseCode string   `json:"response_code"`
		ResponseText string   `json:"response_text"`
		Hosters      []string `json:"hosters"`
	}

	_, err = p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return err
	}

	if result.ResponseCode != "ok" {
		return fmt.Errorf("megadebrid error: %s", result.ResponseText)
	}

	var compiled []*regexp.Regexp
	for _, domain := range result.Hosters {
		pattern := `https?://(www\.)?` + regexp.QuoteMeta(domain)
		re, err := regexp.Compile(pattern)
		if err == nil {
			compiled = append(compiled, re)
		}
	}

	p.regexes = compiled
	return nil
}

func (p *MegaDebridProvider) Priority() int {
	return 60
}

func (p *MegaDebridProvider) Resolve(ctx context.Context, rawURL string, headers map[string]string) (*provider.ResolveResult, error) {
	token, err := p.getToken(ctx)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("link", rawURL)

	opts := rest.Opts{
		Method:      "POST",
		Path:        "",
		Parameters:  url.Values{"action": {"getLink"}, "token": {token}},
		ContentType: "application/x-www-form-urlencoded",
		Body:        strings.NewReader(form.Encode()),
	}

	var result struct {
		ResponseCode string `json:"response_code"`
		ResponseText string `json:"response_text"`
		DebridLink   string `json:"debridLink"`
		Filename     string `json:"filename"`
		Filesize     int64  `json:"filesize"`
	}

	_, err = p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return nil, err
	}

	if result.ResponseCode == "error_token" {
		p.tokenMutex.Lock()
		p.token = ""
		p.tokenMutex.Unlock()
		return p.Resolve(ctx, rawURL, headers)
	}

	if result.ResponseCode != "ok" {
		return nil, fmt.Errorf("megadebrid error: %s", result.ResponseText)
	}

	return &provider.ResolveResult{
		URL:  result.DebridLink,
		Name: result.Filename,
		Size: result.Filesize,
	}, nil
}

func (p *MegaDebridProvider) Test(ctx context.Context) (*model.AccountInfo, error) {
	_, err := p.getToken(ctx)
	if err != nil {
		return nil, err
	}

	return &model.AccountInfo{
		Username:  p.username,
		IsPremium: true,
	}, nil
}

func (p *MegaDebridProvider) GetHosts(ctx context.Context) ([]string, error) {
	return []string{}, nil
}
