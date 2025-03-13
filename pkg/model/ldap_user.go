package model

import (
	"errors"
	"fmt"
	"strings"
)

// LdapUser represents a user resource in LDAP
type LdapUser struct {
	DN              string `ldap:"dn"`              // DN (Distinguished Name)
	CN              string `ldap:"cn"`              // Common Name
	OU              string `ldap:"ou"`              // Organizational Unit
	UID             string `ldap:"uid"`             // User ID
	SN              string `ldap:"sn"`              // Surname
	GivenName       string `ldap:"givenName"`       // Given Name
	TelephoneNumber string `ldap:"telephoneNumber"` // Telephone Number
	Mail            string `ldap:"mail"`            // Email
	GIDNumber       string `ldap:"gidNumber"`       // Group ID Number
	UIDNumber       string `ldap:"uidNumber"`       // User ID Number
	HomeDirectory   string `ldap:"homeDirectory"`   // Home Directory
	UserPassword    []byte `ldap:"userPassword"`    // User Password (binary)
}

// Validate validates the fields of LdapUser
func (u *LdapUser) Validate() error {
	if u.DN == "" {
		return errors.New("DN cannot be empty")
	}
	if u.CN == "" {
		return errors.New("CN cannot be empty")
	}
	if u.OU == "" {
		return errors.New("OU cannot be empty")
	}
	if u.UID == "" {
		return errors.New("UID cannot be empty")
	}
	if u.SN == "" {
		return errors.New("SN cannot be empty")
	}
	if u.GIDNumber == "" {
		return errors.New("GIDNumber cannot be empty")
	}
	if u.UIDNumber == "" {
		return errors.New("UIDNumber cannot be empty")
	}
	if u.HomeDirectory == "" {
		return errors.New("HomeDirectory cannot be empty")
	}
	return nil
}

// ParseDN parses CN and OU from DN
func (u *LdapUser) ParseDN() error {
	parts := strings.Split(u.DN, ",")

	for _, part := range parts {
		if strings.HasPrefix(part, "cn=") {
			u.CN = strings.TrimPrefix(part, "cn=")
		}
		if strings.HasPrefix(part, "ou=") {
			u.OU = strings.TrimPrefix(part, "ou=")
		}
	}

	if u.CN == "" {
		return fmt.Errorf("failed to parse CN or OU from DN: %s", u.DN)
	}

	return nil
}

// ToLDAPEntry converts LdapUser to an LDAP entry
func (u *LdapUser) ToLDAPEntry() map[string][]string {
	return map[string][]string{
		"dn":              {u.DN},
		"cn":              {u.CN},
		"ou":              {u.OU},
		"uid":             {u.UID},
		"sn":              {u.SN},
		"givenName":       {u.GivenName},
		"telephoneNumber": {u.TelephoneNumber},
		"mail":            {u.Mail},
		"gidNumber":       {u.GIDNumber},
		"uidNumber":       {u.UIDNumber},
		"homeDirectory":   {u.HomeDirectory},
		"userPassword":    {string(u.UserPassword)}, // Note: Password is usually stored in binary format
	}
}
