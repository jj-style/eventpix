# show just recipes
_help:
    just --list --unsorted

# generate protos
proto:
    @buf generate
    @/bin/find frontend/src/gen/ -name "*.ts" -exec sed -i 's/_pb\.js/_pb.ts/g' {} \;
