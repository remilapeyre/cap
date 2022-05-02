package ldap

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/hashicorp/go-secure-stdlib/tlsutil"
)

const (
	// DefaultURL for the ClientConfig.URLs
	DefaultURL = "ldaps://127.0.0.1:686"

	// DefaultUserAttr is the "username" attribute of the entry's DN and is
	// typically either the cn in ActiveDirectory or uid in openLDAP  (default:
	// cn)
	DefaultUserAttr = "cn"

	// DefaultGroupFilter for the ClientConfig.GroupFilter
	DefaultGroupFilter = `(|(memberUid={{.Username}})(member={{.UserDN}})(uniqueMember={{.UserDN}}))`

	// DefaultGroupAttr for the ClientConfig.GroupAttr
	DefaultGroupAttr = "cn"

	// DefaultTLSMinVersion for the ClientConfig.TLSMinVersion
	DefaultTLSMinVersion = "tls12"

	// DefaultTLSMaxVersion for the ClientConfig.TLSMaxVersion
	DefaultTLSMaxVersion = "tls12"
)

type ClientConfig struct {
	// URLs are the URLs to use when connecting to a directory (default:
	// ldap://127.0.0.1).  When multiple URLs are specified; they are tried
	// in the order specified.
	URLs []string `json:"urls"`

	// UserDN is the base distinguished name to use when searching for users
	// (eg: ou=People,dc=example,dc=org)
	UserDN string `json:"userdn"`

	// AnonymousGroupSearch specifies that an anonymous bind should be used when
	// searching for groups (if true, the bind credentials will still be used
	// for the initial connection test).
	AnonymousGroupSearch bool `json:"anonymous_group_search"`

	// GroupDN is the distinguished name to use as base when searching for group
	// membership (eg: ou=Groups,dc=example,dc=org)
	GroupDN string `json:"groupdn"`

	// GroupFilter is a Go template for querying the group membership of user
	// (optional).  The template can access the following context variables:
	// UserDN, Username
	//
	// Example:
	// (&(objectClass=group)(member:1.2.840.113556.1.4.1941:={{.UserDN}}))
	// Default: (|(memberUid={{.Username}})(member={{.UserDN}})(uniqueMember={{.UserDN}}))`
	GroupFilter string `json:"groupfilter"`

	// GroupAttr is the attribute which identifies group members in entries
	// returned from GroupFilter queries.  Examples: for groupfilter queries
	// returning group objects, use: cn. For queries returning user objects,
	// use: memberOf.
	// Default: cn
	GroupAttr string `json:"groupattr"`

	// UPNDomain is the userPrincipalName domain, which enables a
	// userPrincipalDomain login with [username]@UPNDomain (optional)
	UPNDomain string `json:"upndomain"`

	// UserFilter (optional) is a Go template used to construct a ldap user
	// search filter. The template can access the following context variables:
	// [UserAttr, Username]. The default userfilter is
	// ({{.UserAttr}}={{.Username}}) or
	// (userPrincipalName={{.Username}}@UPNDomain) if the upndomain parameter
	// is set. The user search filter can be used to  restrict what user can
	// attempt to log in. For example, to limit login to users that are not
	// contractors, you could write
	// (&(objectClass=user)({{.UserAttr}}={{.Username}})(!(employeeType=Contractor)))
	UserFilter string `json:"userfilter"`

	// UserAttr is the "username" attribute of the entry's DN and is typically
	// either the cn in ActiveDirectory or uid in openLDAP  (default: cn)
	UserAttr string `json:"userattr"`

	// Certificate to use verify the identity of the directory service and is a
	// PEM encoded x509 (optional)
	Certificate string `json:"certificate"`

	// ClientTLSCert is the client certificate used with the ClientTLSKey to
	// authenticate the client to the directory service.  It must be PEM encoded
	// x509 (optional)
	ClientTLSCert string `json:"client_tls_cert"`

	// ClientTLSKey is the client certificate key used with the ClientTLSCert to
	// authenticate the client to the directory service.  It must be a PEM
	// encoded x509 (optional)
	ClientTLSKey string `json:"client_tls_key"`

	// InsecureTLS will skip the verification of the directory service's
	// certificate when making a client connection (optional).
	// Warning: this is insecure
	InsecureTLS bool `json:"insecure_tls"`

	// StartTLS will issue the StartTLS command after establishing an initial
	// non-TLS connection (optional)
	StartTLS bool `json:"starttls"`

	// BindDN is the distinguished name used when the client binds
	// (authenticates) to a directory service
	BindDN string `json:"binddn"`

	// BindPassword is the password used with the BindDN when the client binds
	// (authenticates) to a directory service (optional)
	BindPassword string `json:"bindpass"`

	// AllowEmptyPasswordBinds: if true it allows binds even if the user's
	// password is empty (zero length) (optional).
	AllowEmptyPasswordBinds bool `json:"allow_empty_passwd_bind"`

	// DiscoverDN: if true, it will use an anonymous bind with a search
	// to discover the bind DN of a user (optional)
	DiscoverDN bool `json:"discoverdn"`

	// TLSMinVersion version to use. Accepted values are
	// 'tls10', 'tls11', 'tls12' or 'tls13'. Defaults to 'tls12'
	TLSMinVersion string `json:"tls_min_version"`

	// TLSMaxVersion version to use. Accepted values are 'tls10', 'tls11',
	// 'tls12' or 'tls13'. Defaults to 'tls12'
	TLSMaxVersion string `json:"tls_max_version"`

	// UseTokenGroups: if true, use the Active Directory tokenGroups constructed
	// attribute of the user to find the group memberships. This will find all
	// security groups including nested ones.",
	UseTokenGroups bool `json:"use_token_groups"`

	// RequestTimeout in seconds, for the connection when making requests
	// against the server before returning back an error.
	RequestTimeout int `json:"request_timeout"`

	// DeprecatedVaultPre111GroupCNBehavior: if true, group searching reverts to
	// the pre 1.1.1 Vault behavior.
	// see: https://www.vaultproject.io/docs/upgrading/upgrade-to-1.1.1
	DeprecatedVaultPre111GroupCNBehavior *bool `json:"use_pre111_group_cn_behavior"`
}

func (c *ClientConfig) clone() (*ClientConfig, error) {
	clone := *c
	return &clone, nil
}

func (c *ClientConfig) validate() error {
	const op = "ldap.(ClientConfig).validate"
	if len(c.URLs) == 0 {
		return fmt.Errorf("%s: at least one url must be provided: %w", op, ErrInvalidParameter)
	}
	tlsMinVersion, ok := tlsutil.TLSLookup[c.TLSMinVersion]
	if !ok {
		return fmt.Errorf("%s: invalid 'tls_min_version' in config: %w", op, ErrInvalidParameter)
	}
	tlsMaxVersion, ok := tlsutil.TLSLookup[c.TLSMaxVersion]
	if !ok {
		return fmt.Errorf("%s: invalid 'tls_max_version' in config: %w", op, ErrInvalidParameter)
	}
	if tlsMaxVersion < tlsMinVersion {
		return fmt.Errorf("%s: 'tls_max_version' must be greater than or equal to 'tls_min_version': %w", op, ErrInvalidParameter)
	}
	if c.Certificate != "" {
		if err := validateCertificate([]byte(c.Certificate)); err != nil {
			return fmt.Errorf("%s: failed to parse server tls cert: %w", op, err)
		}
	}
	if (c.ClientTLSCert != "" && c.ClientTLSKey == "") ||
		(c.ClientTLSCert == "" && c.ClientTLSKey != "") {
		return fmt.Errorf("%s: both client_tls_cert and client_tls_key must be set in configuration: %w", op, ErrInvalidParameter)
	}
	if c.ClientTLSCert != "" && c.ClientTLSKey != "" {
		if _, err := tls.X509KeyPair([]byte(c.ClientTLSCert), []byte(c.ClientTLSKey)); err != nil {
			return fmt.Errorf("%s: failed to parse client X509 key pair: %w", op, err)
		}
	}
	return nil
}

func validateCertificate(pemBlock []byte) error {
	const op = "ldap.validateCertificate"
	if pemBlock == nil {
		return fmt.Errorf("%s: missing certificate pem block: %w", op, ErrInvalidParameter)
	}
	block, _ := pem.Decode([]byte(pemBlock))
	if block == nil || block.Type != "CERTIFICATE" {
		return fmt.Errorf("%s: failed to decode PEM block in the certificate: %w", op, ErrInvalidParameter)
	}
	_, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("%s: failed to parse certificate %w", op, err)
	}
	return nil
}