package ldap

import (
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"tiansuoVM/pkg/model"

	"github.com/go-ldap/ldap/v3"
)

// LDAPClient holds the connection and configuration to interact with the LDAP server
type LDAPClient struct {
	conn *ldap.Conn
	opts *Options
}

// NewLDAPClient creates and returns a new LDAPClient, establishing the connection
func NewLDAPClient(opts *Options) (*LDAPClient, error) {
	// Attempt to establish a connection with the LDAP server
	ldapURL := fmt.Sprintf("ldap://%s:%d", opts.Host, opts.Port)
	conn, err := ldap.DialURL(ldapURL, ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: true}))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %v", err)
	}

	// Set the timeout for any LDAP operations
	conn.SetTimeout(30 * time.Second)

	// Try to bind using the provided credentials (assuming simple authentication)
	err = conn.Bind(opts.LDAPUserName, opts.LDAPPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to LDAP server: %v", err)
	}

	// Return the new LDAP client
	return &LDAPClient{
		conn: conn,
		opts: opts,
	}, nil
}

// Close closes the LDAP connection
func (c *LDAPClient) Close() {
	if c.conn != nil {
		err := c.conn.Close()
		if err != nil {
			log.Printf("failed to close LDAP connection: %v", err)
		}
	}
}

// Search performs an LDAP search query
func (c *LDAPClient) Search(filter string, attributes []string) ([]*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		c.opts.BaseDN,
		ldap.ScopeWholeSubtree, // Search the entire subtree
		ldap.NeverDerefAliases, // Never dereference aliases
		0,                      // No limit on the number of entries
		0,                      // No time limit
		false,                  // Don't typesafe
		filter,                 // Filter to apply
		attributes,             // Attributes to fetch
		nil,                    // Controls (optional)
	)

	// Execute the search
	searchResult, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %v", err)
	}

	return searchResult.Entries, nil
}

// Modify performs an LDAP modify operation
func (c *LDAPClient) Modify(dn string, modifyRequest *ldap.ModifyRequest) error {
	// Perform the modify operation
	err := c.conn.Modify(modifyRequest)
	if err != nil {
		return fmt.Errorf("LDAP modify failed: %v", err)
	}
	return nil
}

// Add performs an LDAP add operation
func (c *LDAPClient) Add(entry *ldap.AddRequest) error {
	// Perform the add operation
	err := c.conn.Add(entry)
	if err != nil {
		return fmt.Errorf("LDAP add failed: %v", err)
	}
	return nil
}

// Delete performs an LDAP delete operation
func (c *LDAPClient) Delete(dn string) error {
	// Perform the delete operation
	delRequest := ldap.NewDelRequest(dn, nil)
	err := c.conn.Del(delRequest)
	if err != nil {
		return fmt.Errorf("LDAP delete failed: %v", err)
	}
	return nil
}

// FindUserByUID 根据用户ID查找用户
func (c *LDAPClient) FindUserByUID(uid string) (*model.LdapUser, error) {
	// 构建过滤器
	filter := fmt.Sprintf("(cn=%s)", uid)

	// 定义需要获取的属性
	attributes := []string{
		"dn", "cn", "uid", "sn", "givenName", "telephoneNumber", "mail",
		"gidNumber", "uidNumber", "homeDirectory", "userPassword",
	}

	// 执行搜索
	entries, err := c.Search(filter, attributes)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %v", err)
	}

	// 检查是否找到用户
	if len(entries) == 0 {
		return nil, fmt.Errorf("user not found: %s", uid)
	}
	if len(entries) > 1 {
		return nil, fmt.Errorf("multiple users found with uid: %s", uid)
	}

	// 获取用户信息
	entry := entries[0]
	user := &model.LdapUser{
		DN:              entry.DN,
		CN:              entry.GetAttributeValue("cn"),
		UID:             entry.GetAttributeValue("uid"),
		SN:              entry.GetAttributeValue("sn"),
		GivenName:       entry.GetAttributeValue("givenName"),
		TelephoneNumber: entry.GetAttributeValue("telephoneNumber"),
		Mail:            entry.GetAttributeValue("mail"),
		GIDNumber:       entry.GetAttributeValue("gidNumber"),
		UIDNumber:       entry.GetAttributeValue("uidNumber"),
		HomeDirectory:   entry.GetAttributeValue("homeDirectory"),
		UserPassword:    entry.GetEqualFoldRawAttributeValue("userPassword"),
	}

	// 解析DN
	if err := user.ParseDN(); err != nil {
		return nil, fmt.Errorf("failed to parse DN: %v", err)
	}

	return user, nil
}

// Bind 绑定LDAP用户（用于验证密码）
func (c *LDAPClient) Bind(userDN, password string) error {
	// 创建新的连接，避免影响当前连接
	conn, err := ldap.DialURL(fmt.Sprintf("ldap://%s:%d", c.opts.Host, c.opts.Port),
		ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: true}))
	if err != nil {
		return fmt.Errorf("failed to connect to LDAP server: %v", err)
	}
	defer conn.Close()

	// 设置超时
	conn.SetTimeout(10 * time.Second)

	// 尝试绑定
	err = conn.Bind(userDN, password)
	if err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}

	return nil
}

func (c *LDAPClient) FindAllGroups() ([]*model.LdapGroup, error) {
	// 构建过滤器
	filter := fmt.Sprintf("(objectClass=posixGroup)")

	// 定义需要获取的属性
	attributes := []string{
		"dn", "ou", "cn", "gidNumber", "memberUid",
	}

	// 执行搜索
	entries, err := c.Search(filter, attributes)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %v", err)
	}

	groups := make([]*model.LdapGroup, 0, len(entries))
	for _, entry := range entries {
		group := &model.LdapGroup{
			DN:         entry.DN,
			CN:         entry.GetAttributeValue("cn"),
			OU:         entry.GetAttributeValue("ou"),
			GIDNumber:  entry.GetAttributeValue("gidNumber"),
			MemberUIDs: entry.GetAttributeValues("memberUid"),
		}

		groups = append(groups, group)
	}

	return groups, nil
}
