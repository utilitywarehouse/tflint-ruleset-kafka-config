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

| Name                                                | Description                                                                                                                      |
|-----------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------|
| [`msk_module_backend`](rules/msk_module_backend.md) | Requires an S3 backend to be defined, with a key that has as suffix the name of the team (taken from the current directory name) |
| [`msk_app_topics`](rules/msk_app_topics.md)         | Requires apps consume from and produce to only topics define in their module.                                                    |
| [`msk_topic_name`](rules/msk_topic_name.md)         | Requires defined topics in a module to belong to that team.                                                                      |


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

## Debugging

If you're developing some rules and want to see how they run on some actual
Terraform code you will need to.

1.  Install your changes: `make install`

2.  Enable the plugin in the repo, if the plugin is already installed, be sure
    to remove references to source and version (other wise it will install from
    there):
    
    ``` hcl
    plugin "uw-kafka-config" {
      enabled = true
    
      # comment this out if it exists, ensure we use the plugin
      # we just installed, and not building from upstream source
      #version = "1.1.0"
      #source  = "github.com/utilitywarehouse/tflint-ruleset-kafka-config"
    }
    ```

3.  Run the plugin

The plugin expects to be run from the directory containing the files you want
you'll need to change directory first and make sure you pass the :

``` 
$ cd ./path/to/debug
$ tflint --config=$(git rev-parse --show-toplevel)/.tflint.hcl
```

To view more logs you can set set the [`TFLINT_DEBUG` environment
variable](https://github.com/terraform-linters/tflint/blob/fc6795ce12fde842fc73f67e55369a63bdfc27d8/README.md#debugging),
combining all in one line:

    cd ./path/to/debug && TFLINT_LOG=debug tflint --config=$(git rev-parse --show-toplevel)/.tflint-msk.hcl ; cd -

Similarly if you want to debug the plugin via a `pre-commit` hook (assuming the
hook has name `terraform_tflint_msk`):

    TFLINT_LOG=debug pre-commit run --verbose --files ./path/to/debug/*.tf -- terraform_tflint_msk
