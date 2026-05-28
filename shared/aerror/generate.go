package aerror

//go:generate go tool stringer -type=ErrorCode

//go:generate go tool gen-template-for-enum -path=. -type=ErrorCode -output=./errorcode_i18n.g.go -tmpl_file=./_errorcode.go.tmpl
