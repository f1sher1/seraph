FROM golang:1.17.2 as builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    GOOS=linux GOARCH=amd64

WORKDIR /seraph

COPY . .

RUN go build -o api web/cmd/main.go &&\
    go build -o engine app/engine/cmd/main.go &&\
    go build -o plugin-nova plugins/nova/cmd/main.go &&\
    go build -o plugin-std plugins/standard/cmd/main.go &&\
    go build -o create-tables app/db/create_table/main.go &&\
    go build -o register plugins/cmd/register.go &&\
    go build -o tools app/api/cmd/main.go


FROM opensuse/leap:15.4

WORKDIR /seraph

VOLUME ["/tmp"]

COPY --from=builder ["/seraph/api", "/seraph/plugin-nova", "/seraph/plugin-std", "/seraph/engine", "/seraph/create-tables", "/seraph/register","/seraph/tools", "./"]
COPY --from=builder ["/seraph/web/yaml", "./web/yaml"]
COPY --from=builder ["/seraph/config/config.ini", "/tmp/config.ini.template"]
