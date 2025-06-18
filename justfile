set dotenv-load

__git_version := `git rev-parse --short HEAD`
__module := 'github.com/jj-style/eventpix'
__cmd_module := __module / 'backend' / 'cmd'

[private]
help:
    just --list --unsorted

# generate code
generate:
    go generate ./...

build version=__git_version:
    @mkdir -p bin
    CGO_ENABLED=1 go build -ldflags="-w -X '{{__cmd_module}}.Version={{version}}'" -trimpath -o bin/eventpix main.go 

docker-build version=__git_version:
    docker build . -t eventpix -f docker/Dockerfile --build-arg VERSION={{version}}

test *flags:
    go test -cover {{flags}} $(go list ./... | grep -v mocks)

test-full: (test "-shuffle on -race")

run:
    watchexec -e go -e html -e js -e css -r go run main.go server

thumbnailer:
    watchexec -e go -e html -e js -e css -r go run main.go thumbnailer