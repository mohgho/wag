package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-playground/validator/v10"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type OIDC struct {
	IssuerURL           string
	ClientSecret        string
	ClientID            string
	GroupsClaimName     string
	DeviceUsernameClaim string `json:",omitempty"`
	Scopes              []string `json:"scopes,omitempty"`
}

type PAM struct {
	ServiceName string
}

type Webauthn struct {
	DisplayName string
	ID          string
	Origin      string
}

const (
	fullJsonConfigKey = "wag-config-full"

	helpMailKey          = "wag-config-general-help-mail"
	defaultWGFileNameKey = "wag-config-general-wg-filename"
	checkUpdatesKey      = "wag-config-general-check-updates"

	InactivityTimeoutKey = "wag-config-authentication-inactivity-timeout"
	SessionLifetimeKey   = "wag-config-authentication-max-session-lifetime"

	LockoutKey           = "wag-config-authentication-lockout"
	IssuerKey            = "wag-config-authentication-issuer"
	DomainKey            = "wag-config-authentication-domain"
	MFAMethodsEnabledKey = "wag-config-authentication-methods"
	DefaultMFAMethodKey  = "wag-config-authentication-default-method"

	OidcDetailsKey = "wag-config-authentication-oidc"
	PamDetailsKey  = "wag-config-authentication-pam"

	externalAddressKey = "wag-config-network-external-address"
	dnsKey             = "wag-config-network-dns"

	MembershipKey = "wag-membership"
)

func getString(key string) (ret string, err error) {
	resp, err := etcd.Get(context.Background(), key)
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) != 1 {
		return "", fmt.Errorf("incorrect number of %s keys", key)
	}

	err = json.Unmarshal(resp.Kvs[0].Value, &ret)
	if err != nil {
		return "", err
	}

	return ret, nil
}

func getInt(key string) (ret int, err error) {
	resp, err := etcd.Get(context.Background(), key)
	if err != nil {
		return 0, err
	}

	if len(resp.Kvs) != 1 {
		return 0, fmt.Errorf("incorrect number of %s keys", key)
	}

	err = json.Unmarshal(resp.Kvs[0].Value, &ret)
	if err != nil {
		return 0, err
	}

	return ret, nil
}

func GetPAM() (details PAM, err error) {

	response, err := etcd.Get(context.Background(), PamDetailsKey)
	if err != nil {
		return PAM{}, err
	}

	if len(response.Kvs) == 0 {
		return PAM{}, errors.New("no PAM settings found")
	}

	err = json.Unmarshal(response.Kvs[0].Value, &details)
	return
}

func GetOidc() (details OIDC, err error) {

	response, err := etcd.Get(context.Background(), OidcDetailsKey)
	if err != nil {
		return OIDC{}, err
	}

	if len(response.Kvs) == 0 {
		return OIDC{}, errors.New("no oidc settings found")
	}

	err = json.Unmarshal(response.Kvs[0].Value, &details)
	return
}

func GetWebauthn() (wba Webauthn, err error) {

	txn := etcd.Txn(context.Background())
	response, err := txn.Then(clientv3.OpGet(IssuerKey),
		clientv3.OpGet(DomainKey)).Commit()
	if err != nil {
		return wba, err
	}

	// Issuer: GetIssuer()
	if response.Responses[0].GetResponseRange().Count != 1 {
		return wba, errors.New("no issuer set")
	}

	if response.Responses[1].GetResponseRange().Count != 1 {
		return wba, errors.New("no domain set")
	}

	var urlData string
	json.Unmarshal(response.Responses[0].GetResponseRange().Kvs[0].Value, &wba.DisplayName)
	json.Unmarshal(response.Responses[1].GetResponseRange().Kvs[0].Value, &urlData)

	tunnelURL, err := url.Parse(urlData)
	if err != nil {
		return wba, errors.New("unable to parse Authenticators.DomainURL: " + err.Error())
	}

	wba.Origin = tunnelURL.String()
	wba.ID = strings.Split(tunnelURL.Host, ":")[0]

	return
}

func GetWireguardConfigName() string {
	k, err := getString(defaultWGFileNameKey)
	if err != nil {
		return "wg0.conf"
	}

	if k == "" {
		return "wg0.conf"
	}

	return k
}

func SetDefaultMfaMethod(method string) error {

	data, _ := json.Marshal(method)

	_, err := etcd.Put(context.Background(), DefaultMFAMethodKey, string(data))
	return err
}

func GetDefaultMfaMethod() (string, error) {
	return getString(DefaultMFAMethodKey)
}

func SetAuthenticationMethods(methods []string) error {
	data, _ := json.Marshal(methods)
	_, err := etcd.Put(context.Background(), MFAMethodsEnabledKey, string(data))
	return err
}

func GetAuthenicationMethods() (result []string, err error) {

	resp, err := etcd.Get(context.Background(), MFAMethodsEnabledKey)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) != 1 {
		return nil, fmt.Errorf("incorrect number of %s keys", MFAMethodsEnabledKey)
	}

	err = json.Unmarshal(resp.Kvs[0].Value, &result)
	if err != nil {
		return nil, err
	}

	return
}

func ShouldCheckUpdates() (bool, error) {

	resp, err := etcd.Get(context.Background(), checkUpdatesKey)
	if err != nil {
		return false, err
	}

	var ret bool

	err = json.Unmarshal(resp.Kvs[0].Value, &ret)
	if err != nil {
		return false, err
	}

	if len(resp.Kvs) != 1 {
		return false, fmt.Errorf("incorrect number of %s keys", checkUpdatesKey)
	}

	return ret, nil
}

func GetDomain() (string, error) {
	return getString(DomainKey)
}

func SetIssuer(issuer string) error {
	data, _ := json.Marshal(issuer)
	_, err := etcd.Put(context.Background(), IssuerKey, string(data))
	return err
}

func GetIssuer() (string, error) {
	return getString(IssuerKey)
}

func SetHelpMail(helpMail string) error {
	data, _ := json.Marshal(helpMail)
	_, err := etcd.Put(context.Background(), helpMailKey, string(data))
	return err
}

func GetHelpMail() string {

	mail, err := getString(helpMailKey)
	if err != nil {
		return "Server Error"
	}

	return mail
}

func GetExternalAddress() (string, error) {
	return getString(externalAddressKey)
}

func SetDNS(dns []string) error {
	jsonData, _ := json.Marshal(dns)
	_, err := etcd.Put(context.Background(), dnsKey, string(jsonData))
	return err
}

func GetDNS() ([]string, error) {
	resp, err := etcd.Get(context.Background(), dnsKey)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) != 1 {
		return nil, fmt.Errorf("incorrect number of %s keys", dnsKey)
	}

	var servers []string
	err = json.Unmarshal(resp.Kvs[0].Value, &servers)
	if err != nil {
		return nil, err
	}

	return servers, nil
}

type AllSettings struct {
	LoginSettings
	GeneralSettings
}

type LoginSettings struct {
	SessionInactivityTimeoutMinutes int `validate:"required,number"`
	MaxSessionLifetimeMinutes       int `validate:"required,number"`
	Lockout                         int `validate:"required,number"`

	DefaultMFAMethod  string   `validate:"required"`
	EnabledMFAMethods []string `validate:"required,lt=10,dive,required"`

	Domain string `validate:"required"`
	Issuer string `validate:"required"`

	OidcDetails OIDC
	PamDetails  PAM
}

func (lg *LoginSettings) Validate() error {
	lg.Domain = strings.TrimSpace(lg.Domain)
	lg.Issuer = strings.TrimSpace(lg.Issuer)

	validate := validator.New(validator.WithRequiredStructEnabled())

	return validate.Struct(lg)
}

func (lg *LoginSettings) ToWriteOps() (ret []clientv3.Op, err error) {

	if err := lg.Validate(); err != nil {
		return nil, err
	}

	b, _ := json.Marshal(lg.SessionInactivityTimeoutMinutes)
	ret = append(ret, clientv3.OpPut(InactivityTimeoutKey, string(b)))

	b, _ = json.Marshal(lg.MaxSessionLifetimeMinutes)
	ret = append(ret, clientv3.OpPut(SessionLifetimeKey, string(b)))

	b, _ = json.Marshal(lg.Lockout)
	ret = append(ret, clientv3.OpPut(LockoutKey, string(b)))

	b, _ = json.Marshal(lg.DefaultMFAMethod)
	ret = append(ret, clientv3.OpPut(DefaultMFAMethodKey, string(b)))

	b, _ = json.Marshal(lg.EnabledMFAMethods)
	ret = append(ret, clientv3.OpPut(MFAMethodsEnabledKey, string(b)))

	b, _ = json.Marshal(lg.Domain)
	ret = append(ret, clientv3.OpPut(DomainKey, string(b)))

	b, _ = json.Marshal(lg.Issuer)
	ret = append(ret, clientv3.OpPut(IssuerKey, string(b)))

	b, _ = json.Marshal(lg.OidcDetails)
	ret = append(ret, clientv3.OpPut(OidcDetailsKey, string(b)))

	b, _ = json.Marshal(lg.PamDetails)
	ret = append(ret, clientv3.OpPut(PamDetailsKey, string(b)))

	return
}

type GeneralSettings struct {
	HelpMail        string   `validate:"required,email"`
	ExternalAddress string   `validate:"required,hostname|hostname_port|ip"`
	DNS             []string `validate:"omitempty,dive,ip"`

	WireguardConfigFilename string `validate:"required"`
	CheckUpdates            bool
}

func (gs *GeneralSettings) Validate() error {

	gs.HelpMail = strings.TrimSpace(gs.HelpMail)
	gs.ExternalAddress = strings.TrimSpace(gs.ExternalAddress)
	gs.WireguardConfigFilename = strings.TrimSpace(gs.WireguardConfigFilename)
	for i := range gs.DNS {
		gs.DNS[i] = strings.TrimSpace(gs.DNS[i])
	}

	validate := validator.New(validator.WithRequiredStructEnabled())

	return validate.Struct(gs)
}

func (gs *GeneralSettings) ToWriteOps() (ret []clientv3.Op, err error) {

	if err := gs.Validate(); err != nil {
		return nil, err
	}

	b, _ := json.Marshal(gs.HelpMail)
	ret = append(ret, clientv3.OpPut(helpMailKey, string(b)))

	b, _ = json.Marshal(gs.ExternalAddress)
	ret = append(ret, clientv3.OpPut(externalAddressKey, string(b)))

	b, _ = json.Marshal(gs.DNS)
	ret = append(ret, clientv3.OpPut(dnsKey, string(b)))

	b, _ = json.Marshal(gs.WireguardConfigFilename)
	ret = append(ret, clientv3.OpPut(defaultWGFileNameKey, string(b)))

	b, _ = json.Marshal(gs.CheckUpdates)
	ret = append(ret, clientv3.OpPut(checkUpdatesKey, string(b)))

	return
}

func GetAllSettings() (s AllSettings, err error) {

	txn := etcd.Txn(context.Background())
	response, err := txn.Then(clientv3.OpGet(helpMailKey),
		clientv3.OpGet(externalAddressKey),
		clientv3.OpGet(InactivityTimeoutKey),
		clientv3.OpGet(SessionLifetimeKey),
		clientv3.OpGet(LockoutKey),
		clientv3.OpGet(dnsKey),
		clientv3.OpGet(IssuerKey),
		clientv3.OpGet(DomainKey),
		clientv3.OpGet(DefaultMFAMethodKey),
		clientv3.OpGet(MFAMethodsEnabledKey),
		clientv3.OpGet(checkUpdatesKey),
		clientv3.OpGet(OidcDetailsKey),
		clientv3.OpGet(PamDetailsKey),
		clientv3.OpGet(defaultWGFileNameKey)).Commit()
	if err != nil {
		return s, err
	}

	if response.Responses[0].GetResponseRange().Count == 1 {
		err = json.Unmarshal(response.Responses[0].GetResponseRange().Kvs[0].Value, &s.HelpMail)
		if err != nil {
			return s, err
		}
	}

	if response.Responses[1].GetResponseRange().Count == 1 {
		err = json.Unmarshal(response.Responses[1].GetResponseRange().Kvs[0].Value, &s.ExternalAddress)
		if err != nil {
			return s, err
		}
	}

	if response.Responses[2].GetResponseRange().Count == 1 {
		err = json.Unmarshal(response.Responses[2].GetResponseRange().Kvs[0].Value, &s.SessionInactivityTimeoutMinutes)
		if err != nil {
			return s, err
		}
	}

	if response.Responses[3].GetResponseRange().Count == 1 {
		err = json.Unmarshal(response.Responses[3].GetResponseRange().Kvs[0].Value, &s.MaxSessionLifetimeMinutes)
		if err != nil {
			return s, err
		}
	}

	if response.Responses[4].GetResponseRange().Count == 1 {
		err = json.Unmarshal(response.Responses[4].GetResponseRange().Kvs[0].Value, &s.Lockout)
		if err != nil {
			return s, err
		}
	}

	if response.Responses[5].GetResponseRange().Count == 1 {
		err = json.Unmarshal(response.Responses[5].GetResponseRange().Kvs[0].Value, &s.DNS)
		if err != nil {
			return s, err
		}
	}

	if response.Responses[6].GetResponseRange().Count == 1 {
		err = json.Unmarshal(response.Responses[6].GetResponseRange().Kvs[0].Value, &s.Issuer)
		if err != nil {
			return s, err
		}
	}

	if response.Responses[7].GetResponseRange().Count == 1 {
		err = json.Unmarshal(response.Responses[7].GetResponseRange().Kvs[0].Value, &s.Domain)
		if err != nil {
			return s, err
		}
	}

	if response.Responses[8].GetResponseRange().Count == 1 {
		err = json.Unmarshal(response.Responses[8].GetResponseRange().Kvs[0].Value, &s.DefaultMFAMethod)
		if err != nil {
			return s, err
		}
	}

	if response.Responses[9].GetResponseRange().Count == 1 {
		err = json.Unmarshal(response.Responses[9].GetResponseRange().Kvs[0].Value, &s.EnabledMFAMethods)
		if err != nil {
			return s, err
		}
	}

	if response.Responses[10].GetResponseRange().Count == 1 {
		err = json.Unmarshal(response.Responses[10].GetResponseRange().Kvs[0].Value, &s.CheckUpdates)
		if err != nil {
			return s, err
		}
	}

	if response.Responses[11].GetResponseRange().Count == 1 {
		s.OidcDetails.GroupsClaimName = "groups"
		err := json.Unmarshal(response.Responses[11].GetResponseRange().Kvs[0].Value, &s.OidcDetails)
		if err != nil {
			return s, err
		}
	}

	if response.Responses[12].GetResponseRange().Count == 1 {
		err := json.Unmarshal(response.Responses[12].GetResponseRange().Kvs[0].Value, &s.PamDetails)
		if err != nil {
			return s, err
		}
	}

	if response.Responses[13].GetResponseRange().Count == 1 {
		err := json.Unmarshal(response.Responses[13].GetResponseRange().Kvs[0].Value, &s.WireguardConfigFilename)
		if err != nil {
			return s, err
		}
	}

	return
}

func SetLoginSettings(loginSettings LoginSettings) error {

	writeOps, err := loginSettings.ToWriteOps()
	if err != nil {
		return err
	}

	txn := etcd.Txn(context.Background())
	_, err = txn.Then(writeOps...).Commit()
	return err
}

func SetGeneralSettings(generalSettings GeneralSettings) error {
	txn := etcd.Txn(context.Background())
	writeOPs, err := generalSettings.ToWriteOps()
	if err != nil {
		return err
	}
	_, err = txn.Then(writeOPs...).Commit()
	return err
}

func SetSessionLifetimeMinutes(lifetimeMinutes int) error {
	data, _ := json.Marshal(lifetimeMinutes)
	_, err := etcd.Put(context.Background(), SessionLifetimeKey, string(data))
	return err
}

func GetSessionLifetimeMinutes() (int, error) {
	sessionLifeTime, err := getInt(SessionLifetimeKey)
	if err != nil {
		return 0, err
	}

	return sessionLifeTime, nil
}

func SetSessionInactivityTimeoutMinutes(InactivityTimeout int) error {
	data, _ := json.Marshal(InactivityTimeout)

	_, err := etcd.Put(context.Background(), InactivityTimeoutKey, string(data))
	return err
}

func GetSessionInactivityTimeoutMinutes() (int, error) {
	inactivityTimeout, err := getInt(InactivityTimeoutKey)
	if err != nil {
		return 0, err
	}

	return inactivityTimeout, nil
}

// Get account lockout threshold setting
func GetLockout() (int, error) {
	lockout, err := getInt(LockoutKey)
	if err != nil {
		return 0, err
	}

	return lockout, nil
}
