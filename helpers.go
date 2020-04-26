package main

import (
	"fmt"
	"regexp"
)

func extractPlusMinusEventData(message string) []string {
	regex := *regexp.MustCompile(`@([!-=?-~]+)>?\s*(\+{2}|-{2}|—{1}|={2})`)
	res := regex.FindAllStringSubmatch(message, -1)

	var i, j int
	for i = 0; i < len(res); i++ {
		for j = 0; j < len(res[i]); j++ {
			fmt.Printf("res[%d][%d] = %d\n", i, j, res[i][j])
		}
	}

	if res != nil {
		return []string{res[0][1], res[0][2]}
	}

	return nil
}
