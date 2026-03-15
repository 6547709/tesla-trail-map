FROM golang:1.25-alpine AS build

WORKDIR /src

# 先只复制 go.mod 和 go.sum 文件
COPY go.mod go.sum ./

# 下载依赖（使用国内镜像如果网络有问题）
RUN go mod download

# 然后复制所有源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 go build -o app .

FROM alpine:latest

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=build /src/app /app/
COPY --from=build /src/map.html /app/

ENTRYPOINT ["./app"]