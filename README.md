GOBF
====
Go path obfuscator... it may be easier to just replace identifying strings in the binary

```bash
$ go build *.go && ./main --src ./test/saas/sites/cmd/rest-server --root ./test/saas --target /tmp/scratch
ready to build:
GOPATH=/tmp/scratch go build -o rest-server VsgvD
$ GOPATH=/tmp/scratch go build -o rest-server VsgvD
```
