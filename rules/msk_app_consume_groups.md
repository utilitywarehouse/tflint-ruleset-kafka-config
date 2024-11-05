# `msk_app_consume_groups`

Requires that any element of the `consume_groups` in a `tls-app` be prefixed wit
the name of the team using it, e.g. `my-team.my-consumer-group`

This is because MSK is a multi-tenant environment and we want to know to which
team a consumer group belongs. Additionally, in kafka-ui, access is given to
consumer groups based on the team prefixes.

## Examples

### Bad example

``` hcl
module "my_indexer" {
  source           = "../../../modules/tls-app"
  cert_common_name = "some-team/indexer"
  consume_topics = ["some-team.example"]

  consume_groups = [
    # BAD: group name not prefixed with consuming team
    "some-example-consumer"
  ]
}
```

### Good example

The simplest solution is to just add your team name as the prefix:

``` hcl
module "my_indexer" {
  source           = "../../../modules/tls-app"
  cert_common_name = "some-team/indexer"
  consume_topics = ["some-team.example"]

  consume_groups = [
    # GOOD: group name prefixed with team
    "some-team.some-example-consumer"
  ]
}
```
