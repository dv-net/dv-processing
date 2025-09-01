package util

import "strings"

func FormatNumberWithPrecision(number string, precision int) string {
	if precision <= 0 {
		return number
	}

	// Removing a possible point in the input line
	number = strings.ReplaceAll(number, ".", "")

	// If the number is less than 1, add zeros
	for len(number) <= precision {
		number = "0" + number
	}

	// Insert a point
	decimalIndex := len(number) - precision
	formattedNumber := number[:decimalIndex] + "." + number[decimalIndex:]

	return formattedNumber
}
