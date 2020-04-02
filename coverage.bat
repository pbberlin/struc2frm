REM go get golang.org/x/tools/cmd/cover
go test -coverprofile tmp-coverage.out   github.com/pbberlin/struc2frm
go tool cover -html=tmp-coverage.out     