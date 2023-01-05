package error

import "log"

func CheckError(err error, message ...string) {
	if err != nil {
		log.Fatal(err, message)
	}
}
