package main

import (
	"regexp"
	"strings"
)

func extractPlusMinusEventData(message string) []string {
	regex := *regexp.MustCompile(`@([!-=?-~]+)>?\s*(\+{2}|-{2}|—{1}|={2})`)
	res := regex.FindAllStringSubmatch(message, -1)

	println("Message: " + message)
	if res != nil {
		res[0][1] = (*regexp.MustCompile(`(^\!)`)).ReplaceAllString(res[0][1], "")
		res[0][1] = strings.Replace(res[0][1], "/", "", -1)
		res[0][1] = strings.Replace(res[0][1], "\\", "", -1)
		res[0][1] = strings.Replace(res[0][1], "#", "", -1)
		res[0][1] = strings.Replace(res[0][1], "?", "", -1)
		println("Extracted Subject: " + res[0][1] + " Extracted Operation: " + res[0][2])
		return []string{res[0][1], res[0][2]}
	}

	return nil
}
