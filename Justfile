image_name := "dmi7ry/tado-exporter"
image_ver := `cat VERSION`
image_full_name := image_name + ":" + image_ver
app_name := "tado-exporter"
builder_name := "multiarch"

#run:
#  ./idex_exporter --query-staking \
#    --staking-wallet {{staking_wallet}} \
#    --staking-api-key {{staking_api_key}} \
#    --log.level=info

bin_build OS="darwin":
    CGO_ENABLED=0 GOOS={{OS}} go build -ldflags="-w -s" -o main *.go

vars:
    #!/bin/bash
    export $(grep -v '^#' .env | xargs)

bin_build_run:
    source .env
    just bin_build
    ./main

build:
    docker build --tag {{image_full_name}} -f Dockerfile .

# --progress=plain
multiarch_build *args:
    docker buildx build {{args}} --platform linux/arm64,linux/amd64 --tag {{image_full_name}} --builder {{builder_name}} -f Dockerfile .

dive:
    dive {{image_full_name}}

push:
    docker push {{image_full_name}}

run:
    docker run -it --env-file .env -p 9888:9888 --rm --name {{app_name}} {{image_full_name}}

rmf:
    docker ps -aq | xargs docker rm -f

bench TIME="1s":
    go test -bench=. -benchtime={{TIME}} -benchmem