package utils

import (	"fmt"
			"net/smtp"
			"strings"
		)


var message = map[string] string {
	"*" : "Here are your results based on your Amazebet:\n\nNumbers: %d hits\nStars: %d hits\n\nAmazebet",
	"pt" : "Aqui estão os resultados da sua aposta Amazebet:\n\nNúmeros: %d certos\nEstrelas: %d certos\n\nAmazebet",
}

var subject = map[string] string {
	"*" : "Euromillions Amazebet Results",
	"pt" : "Resultados Euromilhões Amazebet",
}


func sendMail(host, user, pwd, from, to, subject, body, mailType string) error {
	//auth := smtp.PlainAuth("", user, pwd, strings.Split(host, ":")[0])
	cntType := fmt.Sprintf("Content-Type: text/%s;charset=UTF-8", mailType)
	msg := fmt.Sprintf("To: %s\r\nFrom: %s<%s>\r\nSubject: %s\r\n%s\r\n\r\n%s",
        to, from, user, subject, cntType, body)

    return smtp.SendMail(host, nil, user, strings.Split(to, ","), []byte(msg))
}

func Notify(mail string, mh int, sh int, lang string) {
	body := fmt.Sprintf(message[lang], mh, sh)	 
	sendMail("localhost:25", "", "",  "webmaster@amazebet.com",  mail, "Amazebet results", body, "PLAIN")
} 