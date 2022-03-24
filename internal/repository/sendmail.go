package repository

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	log "mdata/pkg/logging"

	"github.com/go-mail/mail"
)

type attachment struct {
	name        string
	contentType string
	withFile    bool
}

type Message struct {
	from        string
	to          []string
	cc          []string
	bcc         []string
	subject     string
	body        string
	contentType string
	attachment  attachment
}

type Mail interface {
	Auth()
	Send(message Message) error
}

type sendMail struct {
	user     string
	password string
	host     string
	port     string
	auth     smtp.Auth
}

func SendMailToRecipient(to []string, BccAdmin, subject, body, attachment string) error {

	// cfg := &dom.Config{}
	// cfg.MailUser = ReadConfig("mail.user")
	// cfg.MailPassword = ReadConfig("mail.password")
	// cfg.MailHost = ReadConfig("mail.host")
	// cfg.MailPort = ReadConfig("mail.port")
	// cfg.MailFrom = ReadConfig("mail.from")

	cfg := GetCfg()

	curPort, _ := strconv.Atoi(cfg.MailPort)
	d := mail.Dialer{Host: cfg.MailHost, Port: curPort, StartTLSPolicy: mail.NoStartTLS}
	m := mail.NewMessage()
	m.SetHeaders(map[string][]string{
		"From":    {cfg.MailFrom},
		"To":      to,
		"Subject": {subject},
	})
	m.SetAddressHeader("Bcc", BccAdmin, "admin")

	body = body + "\n" +
		"---------- \n" +
		"Your notifications center"

	m.SetBody("text/plain", body)
	if attachment != "" {
		m.Attach(attachment)
	}
	err := d.DialAndSend(m)
	if err != nil {
		log.Error("repository.SendMailToRecipient d.DialAndSend(m)) error: %v", err)
		return err
	}
	return nil
}

func SendEmailTo1CAdmins(ins *Instance, body string) error {
	admins, err := ins.GetUserEmailsByNotificationsTypes(1)
	if err != nil {
		log.Error("SendEmailTo1CAdmins GetBccAdmin error: %v", err)
		return err
	}

	err = SendMailToRecipient(admins, "", "From MD", body, "")
	return err
}

func SendMailToRecipient_old____________(buchFilePath string, to []string) {

	// cfg := &dom.Config{}
	// cfg.MailUser = ReadConfig("mail.user")
	// cfg.MailPassword = ReadConfig("mail.password")
	// cfg.MailHost = ReadConfig("mail.host")
	// cfg.MailPort = ReadConfig("mail.port")
	// cfg.MailFrom = ReadConfig("mail.from")

	cfg := GetCfg()
	//var mail Mail
	mail := &sendMail{user: "", //cfg.MailUser,
		password: "", //cfg.MailPassword,
		host:     cfg.MailHost,
		port:     cfg.MailPort}
	message := Message{from: cfg.MailFrom,
		to:          to,
		cc:          []string{},
		bcc:         []string{},
		subject:     "Emails file",
		body:        "",
		contentType: "text/plain;charset=utf-8",
		attachment: attachment{
			name:        buchFilePath,
			contentType: "text/plain", //"image/jpg",
			withFile:    true,
		},
	}
	err := mail.Send(message)
	if err != nil {
		log.Error("repository SendMailToRecipient mail.Send(message) error: %v", err)
	}
}

func (mail *sendMail) Auth() {
	//mail.auth = smtp.PlainAuth("", mail.user, mail.password, mail.host)
	mail.auth = smtp.PlainAuth("", "", "", mail.host)
}

func (mail sendMail) Send(message Message) error {
	//mail.Auth()
	buffer := bytes.NewBuffer(nil)
	boundary := "GoBoundary"
	Header := make(map[string]string)
	Header["From"] = message.from
	Header["To"] = strings.Join(message.to, ";")
	Header["Cc"] = strings.Join(message.cc, ";")
	Header["Bcc"] = strings.Join(message.bcc, ";")
	Header["Subject"] = message.subject
	Header["Content-Type"] = "multipart/mixed;boundary=" + boundary
	Header["Mime-Version"] = "1.0"
	Header["Date"] = time.Now().String()
	mail.writeHeader(buffer, Header)

	body := "\r\n--" + boundary + "\r\n"
	body += "Content-Type:" + message.contentType + "\r\n"
	body += "\r\n" + message.body + "\r\n"
	buffer.WriteString(body)

	if message.attachment.withFile {
		attachment := "\r\n--" + boundary + "\r\n"
		attachment += "Content-Transfer-Encoding:base64\r\n"
		attachment += "Content-Disposition:attachment\r\n"
		attachment += "Content-Type:" + message.attachment.contentType + ";name=\"" + message.attachment.name + "\"\r\n"
		buffer.WriteString(attachment)
		defer func() {
			if err := recover(); err != nil {
				log.Error("repository.SendMail recover error: %v", err)
			}
		}()
		mail.writeFile(buffer, message.attachment.name)
	}

	buffer.WriteString("\r\n--" + boundary + "--")
	err := smtp.SendMail(mail.host+":"+mail.port, mail.auth, message.from, message.to, buffer.Bytes())
	if err != nil {
		log.Error("repository smtp.SendMail error: %v", err)
		return err
	}
	return nil
}

func (mail sendMail) writeHeader(buffer *bytes.Buffer, Header map[string]string) string {
	header := ""
	for key, value := range Header {
		header += key + ":" + value + "\r\n"
	}
	header += "\r\n"
	buffer.WriteString(header)
	return header
}

// read and write the file to buffer
func (mail sendMail) writeFile(buffer *bytes.Buffer, fileName string) {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic(err.Error())
	}
	payload := make([]byte, base64.StdEncoding.EncodedLen(len(file)))
	base64.StdEncoding.Encode(payload, file)
	buffer.WriteString("\r\n")
	for index, line := 0, len(payload); index < line; index++ {
		buffer.WriteByte(payload[index])
		if (index+1)%76 == 0 {
			buffer.WriteString("\r\n")
		}
	}
}
