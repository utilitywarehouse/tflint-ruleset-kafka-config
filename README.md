# TFLint kafka-config ruleset

This is a [tflint](https://github.com/terraform-linters/tflint) [plugin](https://github.com/terraform-linters/tflint/blob/master/docs/developer-guide/plugins.md) for enforcing UW rules over our kafka config.

## Installation

You can install the plugin with `tflint --init`. Declare a config in `.tflint.hcl` as follows:

```hcl
plugin "uw-kafka-config" {
  enabled = true

  version = "x.y.z"
  source  = "github.com/utilitywarehouse/tflint-ruleset-kafka-config"
}
```

## Rules

| Name                                              | Description                                                                              |
|---------------------------------------------------|------------------------------------------------------------------------------------------|
| [msk_module_backend](rules/msk_module_backend.md) | Requires an S3 backend to be defined, with a key that has as suffix the name of the team (taken from the current directory name) |


## Building the plugin

Clone the repository locally and run the following command:

```
$ make
```

You can easily install locally the built plugin with the following:

```
$ make install
```

You can run the built plugin like the following:

```
$ cat << EOS > .tflint.hcl
plugin "kafka-config" {
  enabled = true
}
EOS
$ tflint
```

## Releasing

For releasing the binaries for the plugin you just need to create a Github release named _vx.y.z_, like v0.1.0.

[Goreleaser](https://goreleaser.com/) is used in the pipeline. See [config](.goreleaser.yaml)

## Linting

Linting is handled via `pre-commit`. Follow the [install
instructions](https://pre-commit.com/#install) then install the hooks:

``` console
$ pre-commit install
$ pre-commit run --all-hooks
```
