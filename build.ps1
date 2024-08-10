# notice how we avoid spaces in $now to avoid quotation hell in go build command
$now = Get-Date -UFormat "%Y-%m-%d_%T"
$sha1 = (git rev-parse HEAD).Trim()

go build -o godfather.exe -ldflags "-X main.sha1ver=$sha1 -X main.buildTime=$now"