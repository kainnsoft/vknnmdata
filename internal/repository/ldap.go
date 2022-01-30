package repository

import (
	"fmt"
	"mdata/internal/domain"
	"strings"

	"github.com/go-ldap/ldap"
)

// Для работы с AD
type LdapConn struct {
	Conn *ldap.Conn
}

func LdapSetConnection(cfg *domain.Config) (*ldap.Conn, error) {

	conn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%s", cfg.ADHost, cfg.ADPort))
	if err != nil {
		return nil, fmt.Errorf("repository.LdapSetConnection error: failed to connect: %s", err)
	}
	err = conn.Bind(cfg.ADUsername, cfg.ADPassword)
	if err != nil {
		// error in ldap bind
		return nil, fmt.Errorf("repository.LdapSetConnection error: failed conn.Bind: %s", err)
	}

	return conn, nil
}

func (c *LdapConn) GetUserEmailByID(userId string) (string, error) {
	var filterDN string = "(employeeNumber={usertabno})" // userId - это табельный номер
	var baseDN string = "OU=ПОЛЬЗОВАТЕЛИ VKNN,dc=VODOKANAL-NN,dc=RU"

	filterDN = strings.Replace(
		filterDN,
		"{usertabno}",
		"*"+userId,
		-1,
	)

	result, err := c.Conn.Search(ldap.NewSearchRequest(
		baseDN, //baseDN,
		ldap.ScopeWholeSubtree,
		0, //ldap.NeverDerefAliases,
		0,
		0,
		false,
		filterDN,
		[]string{"mailNickname", "mail"}, // "mailNickname" "sn", "givenName"},"dn", "sAMAccountName", "employeeNumber",
		nil,
	))

	if err != nil {
		return "", fmt.Errorf("repository.GetUserEmailByID active directory failed to search user: %v", err)
	}

	var currEmail string
	for _, entry := range result.Entries {
		// правило от Бочкарёва: "если заполнено поле mailNickname, значит почта есть - берем ее из mail"
		curmailNickname := entry.GetAttributeValue("mailNickname")
		if len(strings.TrimSpace(curmailNickname)) > 0 {
			currEmail = entry.GetAttributeValue("mail")
		}
	}
	return currEmail, nil
}
