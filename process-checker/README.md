# Process Checker

# GoDoc
```
$ godoc -http=:8000
```
- 위 명령어를 실행한 후, http://localhost:8000/pkg/main/ 에서 확인할 수 있습니다.

# Build 
#### Linux Amd64
```bash
$ GOOS=linux GOARCH=amd64 go build main.go
```
#### Linux armv7
```bash
$ GOOS=linux GOARM=7 GOARCH=arm go build main.go
```

#### Winodw
```bash
$ GOOS=windows CGO_ENABLED=1 go build main.go
```
