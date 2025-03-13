package model

import (
	"errors"
	"fmt"
	"strings"
)

// LdapGroup represents a group resource in LDAP
type LdapGroup struct {
	DN         string   `ldap:"dn"`        // DN (Distinguished Name)
	CN         string   `ldap:"cn"`        // Common Name
	OU         string   `ldap:"ou"`        // Organizational Unit
	GIDNumber  string   `ldap:"gidNumber"` // Group ID Number
	MemberUIDs []string `ldap:"memberUid"` // Member UIDs
}

// Validate validates the fields of LdapGroup
func (g *LdapGroup) Validate() error {
	if g.DN == "" {
		return errors.New("DN cannot be empty")
	}
	if g.CN == "" {
		return errors.New("CN cannot be empty")
	}
	if g.OU == "" {
		return errors.New("OU cannot be empty")
	}
	if g.GIDNumber == "" {
		return errors.New("GIDNumber cannot be empty")
	}
	return nil
}

// ParseDN parses CN and OU from DN
func (g *LdapGroup) ParseDN() error {
	parts := strings.Split(g.DN, ",")
	if len(parts) < 2 {
		return fmt.Errorf("invalid DN format: %s", g.DN)
	}

	for _, part := range parts {
		if strings.HasPrefix(part, "cn=") {
			g.CN = strings.TrimPrefix(part, "cn=")
		}
		if strings.HasPrefix(part, "ou=") {
			g.OU = strings.TrimPrefix(part, "ou=")
		}
	}

	if g.CN == "" || g.OU == "" {
		return fmt.Errorf("failed to parse CN or OU from DN: %s", g.DN)
	}

	return nil
}

// ToLDAPEntry converts LdapGroup to an LDAP entry
func (g *LdapGroup) ToLDAPEntry() map[string][]string {
	return map[string][]string{
		"dn":        {g.DN},
		"cn":        {g.CN},
		"ou":        {g.OU},
		"gidNumber": {g.GIDNumber},
		"memberUid": g.MemberUIDs,
	}
}
