# gstow

### info

Build actions for [gstow](https://github.com/gbraad-dotfiles/gstow), a cross-platform
GNU stow replacement written in Go.

### config
```ini
[compile]
    repo="https://github.com/gbraad-dotfiles/gstow"
    repo_path="/home/gbraad/Projects/gbraad-dotfiles/gstow"
    out_path="${HOME}/Projects/gbraad-dotfiles/gstow"
    out_dest="${HOME}/Uploads/gstow"
    flatten=1

[devenv]
    name="gobuild"
    from="gofedora"
```

### vars
```sh
# hardcode /home/gbraad instead of ~/
COMPILE_REPO_LOCAL=$(eval echo "${COMPILE_REPO_PATH}")
```

### local-build
```sh evaluate
go build -buildvcs=false -o gstow ./...
```

### build
One-shot ephemeral build for local arch.

```sh evaluate
devenv ${DEVENV_FROM} ephemeral -c "cd ${COMPILE_REPO_LOCAL} && go build -buildvcs=false -o gstow ./..."
```

### amd-build
Build for linux/amd64.

```sh
devenv ${DEVENV_FROM} ephemeral -c "cd ${COMPILE_REPO_LOCAL} && GOARCH=amd64 GOOS=linux go build -buildvcs=false -o gstow-amd64 ./..."
```

### arm-build
Build for linux/arm64.

```sh
devenv ${DEVENV_FROM} ephemeral -c "cd ${COMPILE_REPO_LOCAL} && GOARCH=arm64 GOOS=linux go build -buildvcs=false -o gstow-arm64 ./..."
```

### windows-build
Build for Windows amd64.

```sh
devenv ${DEVENV_FROM} ephemeral -c "cd ${COMPILE_REPO_LOCAL} && GOARCH=amd64 GOOS=windows go build -buildvcs=false -o gstow.exe ./..."
```

### test
```sh
devenv ${DEVENV_FROM} ephemeral -c "cd ${COMPILE_REPO_LOCAL} && go test ./..."
```

### fmt
```sh
devenv ${DEVENV_FROM} ephemeral -c "cd ${COMPILE_REPO_LOCAL} && go fmt ./..."
```

### tidy
```sh
devenv ${DEVENV_FROM} ephemeral -c "cd ${COMPILE_REPO_LOCAL} && go mod tidy"
```

### default alias build
```sh
app go build
```
