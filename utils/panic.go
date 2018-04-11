package utils

// https://stackoverflow.com/questions/25190971/golang-copy-exec-output-to-log

// PanicIfError panics if err is not null
func PanicIfError(err error) {
	if nil != err {
		panic(err)
	}
}
