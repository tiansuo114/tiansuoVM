package ldap

import "tiansuoVM/pkg/model"

func LDAPEntry2LDAPUser(entry map[string][]string) (*model.LdapUser, error) {
	user := &model.LdapUser{
		DN:              GetFirstValue(entry, "dn"),
		CN:              GetFirstValue(entry, "cn"),
		OU:              GetFirstValue(entry, "ou"),
		UID:             GetFirstValue(entry, "uid"),
		SN:              GetFirstValue(entry, "sn"),
		GivenName:       GetFirstValue(entry, "givenName"),
		TelephoneNumber: GetFirstValue(entry, "telephoneNumber"),
		Mail:            GetFirstValue(entry, "mail"),
		GIDNumber:       GetFirstValue(entry, "gidNumber"),
		UIDNumber:       GetFirstValue(entry, "uidNumber"),
		HomeDirectory:   GetFirstValue(entry, "homeDirectory"),
		UserPassword:    []byte(GetFirstValue(entry, "userPassword")), // 注意：密码通常以二进制形式存储
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	return user, nil
}

func LDAPEntry2LDAPGroup(entry map[string][]string) (*model.LdapGroup, error) {
	group := &model.LdapGroup{
		DN:         GetFirstValue(entry, "dn"),
		CN:         GetFirstValue(entry, "cn"),
		OU:         GetFirstValue(entry, "ou"),
		GIDNumber:  GetFirstValue(entry, "gidNumber"),
		MemberUIDs: entry["memberUid"],
	}

	if err := group.Validate(); err != nil {
		return nil, err
	}

	return group, nil
}

func GetFirstValue(entry map[string][]string, key string) string {
	if values, ok := entry[key]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}
