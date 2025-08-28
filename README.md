# helm-schema

derive a json schema for your chart

## usage

`helm plugin install https://github.com/nhomble/helm-schema`

### via plugin

```
helm schema ./chart/dir
cat ./chart/dir/values.schema.json
```

### via cli

```
helm-schema ./chart/dir
```

or from this repo `make test/example` to quicky see in action

## build

```
make
```